package logstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var (
	ErrLogNotFound = fmt.Errorf("log not found")
)

type LogMetadata struct {
	ID             string
	ExpirationTime time.Time // Expiration time of the log, zero if no expiration
	OwnerOpID      int64     // ID of the operation that owns this log
}

type LogStore struct {
	dir           string
	inprogressDir string
	mu            shardedRWMutex
	dbpool        *sqlitex.Pool

	trackingMu  sync.Mutex     // guards refcount and subscribers
	refcount    map[string]int // id : refcount
	subscribers map[string][]chan struct{}
}

func NewLogStore(dir string) (*LogStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %v", err)
	}

	dbpath := filepath.Join(dir, "logs.sqlite")
	dbpool, err := sqlitex.NewPool(dbpath, sqlitex.PoolOptions{
		PoolSize: 16,
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}

	ls := &LogStore{
		dir:           dir,
		inprogressDir: filepath.Join(dir, ".inprogress"),
		mu:            newShardedRWMutex(64), // 64 shards should be enough to avoid much contention
		dbpool:        dbpool,
		subscribers:   make(map[string][]chan struct{}),
		refcount:      make(map[string]int),
	}
	if err := ls.init(); err != nil {
		return nil, fmt.Errorf("init log store: %v", err)
	}

	return ls, nil
}

func (ls *LogStore) init() error {
	if err := os.MkdirAll(ls.inprogressDir, 0755); err != nil {
		return fmt.Errorf("create inprogress dir: %v", err)
	}

	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	if err := sqlitex.ExecuteScript(conn, `
		PRAGMA auto_vacuum = 1;
		PRAGMA journal_mode=WAL;

		CREATE TABLE IF NOT EXISTS logs (
			id TEXT PRIMARY KEY,
			expiration_ts_unix INTEGER DEFAULT 0, -- unix timestamp of when the log will expire
			owner_opid INTEGER DEFAULT 0, -- id of the operation that owns this log; will be used for cleanup.
			data_fname TEXT, -- relative path to the file containing the log data
			data_gz BLOB -- compressed log data as an alternative to data_fname
		);
		
		CREATE INDEX IF NOT EXISTS logs_data_fname_idx ON logs (data_fname);
		CREATE INDEX IF NOT EXISTS logs_expiration_ts_unix_idx ON logs (expiration_ts_unix);

		CREATE TABLE IF NOT EXISTS version_info (
			version INTEGER NOT NULL
		);
		
		-- Create a table to store the schema version, will be used for migrations in the future
		INSERT INTO version_info (version)
		SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM version_info);
	`, nil); err != nil {
		return fmt.Errorf("execute init script: %v", err)
	}

	// loop through all inprogress files and finalize them if they are in the database
	files, err := os.ReadDir(ls.inprogressDir)
	if err != nil {
		return fmt.Errorf("read inprogress dir: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fname := file.Name()
		var id string
		if err := sqlitex.ExecuteTransient(conn, "SELECT id FROM logs WHERE data_fname = ?", &sqlitex.ExecOptions{
			Args: []any{fname},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				id = stmt.ColumnText(0)
				return nil
			},
		}); err != nil {
			return fmt.Errorf("select log: %v", err)
		}

		if id != "" {
			err := ls.finalizeLogFile(id, fname)
			if err != nil {
				zap.S().Warnf("sqlite log writer couldn't finalize dangling inprogress log file %v: %v", fname, err)
				continue
			}
		}
		if err := os.Remove(filepath.Join(ls.inprogressDir, fname)); err != nil {
			zap.S().Warnf("sqlite log writer couldn't remove dangling inprogress log file %v: %v", fname, err)
		}
	}

	return nil
}

func (ls *LogStore) Close() error {
	return ls.dbpool.Close()
}

func (ls *LogStore) Create(id string, parentOpID int64, ttl time.Duration) (io.WriteCloser, error) {
	ls.mu.Lock(id)
	defer ls.mu.Unlock(id)

	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	// potentially prune any expired logs
	if err := sqlitex.Execute(conn, "DELETE FROM logs WHERE expiration_ts_unix < ? AND expiration_ts_unix != 0", &sqlitex.ExecOptions{
		Args: []any{time.Now().Unix()},
	}); err != nil {
		return nil, fmt.Errorf("prune expired logs: %v", err)
	}

	// create a random file for the log while it's being written
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("generate random bytes: %v", err)
	}
	fname := hex.EncodeToString(randBytes) + ".log"
	f, err := os.Create(filepath.Join(ls.inprogressDir, fname))
	if err != nil {
		return nil, fmt.Errorf("create temp file: %v", err)
	}

	var expire_ts_unix int64 = 0
	if ttl != 0 {
		expire_ts_unix = time.Now().Add(ttl).Unix()
	}

	if err := sqlitex.Execute(conn, "INSERT INTO logs (id, expiration_ts_unix, owner_opid, data_fname) VALUES (?, ?, ?, ?)", &sqlitex.ExecOptions{
		Args: []any{id, expire_ts_unix, parentOpID, fname},
	}); err != nil {
		return nil, fmt.Errorf("insert log: %v", err)
	}

	ls.trackingMu.Lock()
	ls.subscribers[id] = make([]chan struct{}, 0)
	ls.refcount[id] = 1
	ls.trackingMu.Unlock()

	return &writer{
		ls:    ls,
		f:     f,
		fname: fname,
		id:    id,
	}, nil
}

func (ls *LogStore) GetMetadata(id string) (LogMetadata, error) {
	ls.mu.RLock(id)
	defer ls.mu.RUnlock(id)

	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return LogMetadata{}, fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)
	var metadata LogMetadata
	if err := sqlitex.Execute(conn, "SELECT expiration_ts_unix, owner_opid FROM logs WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			metadata.ID = id
			expireTsUnix := stmt.ColumnInt64(0)
			if expireTsUnix != 0 {
				metadata.ExpirationTime = time.Unix(expireTsUnix, 0)
			}
			metadata.OwnerOpID = stmt.ColumnInt64(1)
			return nil
		},
	}); err != nil {
		return LogMetadata{}, fmt.Errorf("select log metadata: %v", err)
	} else if metadata.ID == "" {
		return LogMetadata{}, ErrLogNotFound
	}
	return metadata, nil
}

func (ls *LogStore) Open(id string) (io.ReadCloser, error) {
	ls.mu.Lock(id)
	defer ls.mu.Unlock(id)

	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	var found bool
	var fname string
	var dataGz []byte
	if err := sqlitex.Execute(conn, "SELECT data_fname, data_gz FROM logs WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			if !stmt.ColumnIsNull(0) {
				fname = stmt.ColumnText(0)
			}
			if !stmt.ColumnIsNull(1) {
				dataGz = make([]byte, stmt.ColumnLen(1))
				stmt.ColumnBytes(1, dataGz)
			}
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("select log: %v", err)
	} else if !found {
		return nil, ErrLogNotFound
	}

	if fname != "" {
		f, err := os.Open(filepath.Join(ls.inprogressDir, fname))
		if err != nil {
			return nil, fmt.Errorf("open data file: %v", err)
		}
		ls.trackingMu.Lock()
		ls.refcount[id]++
		ls.trackingMu.Unlock()

		return &reader{
			ls:     ls,
			f:      f,
			id:     id,
			fname:  fname,
			closed: make(chan struct{}),
		}, nil
	} else if dataGz != nil {
		gzr, err := gzip.NewReader(bytes.NewReader(dataGz))
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %v", err)
		}

		return gzr, nil
	} else {
		return nil, errors.New("log has no associated data. This shouldn't be possible")
	}
}

func (ls *LogStore) Delete(id string) error {
	ls.mu.Lock(id)
	defer ls.mu.Unlock(id)

	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	if err := sqlitex.Execute(conn, "DELETE FROM logs WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{id},
	}); err != nil {
		return fmt.Errorf("delete log: %v", err)
	}

	if conn.Changes() == 0 {
		return ErrLogNotFound
	}
	return nil
}

func (ls *LogStore) DeleteWithParent(parentOpID int64) error {
	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	if err := sqlitex.Execute(conn, "DELETE FROM logs WHERE owner_opid = ?", &sqlitex.ExecOptions{
		Args: []any{parentOpID},
	}); err != nil {
		return fmt.Errorf("delete log: %v", err)
	}

	return nil
}

func (ls *LogStore) SelectAll(f func(id string, parentID int64)) error {
	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	return sqlitex.Execute(conn, "SELECT id, owner_opid FROM logs ORDER BY owner_opid", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			f(stmt.ColumnText(0), stmt.ColumnInt64(1))
			return nil
		},
	})
}

// Find logs owned by a specific operation ID.
func (ls *LogStore) FindLogsWithParent(parentOpID int64) ([]string, error) {
	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	var logs []string
	if err := sqlitex.Execute(conn, "SELECT id FROM logs WHERE owner_opid = ?", &sqlitex.ExecOptions{
		Args: []any{parentOpID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			logs = append(logs, stmt.ColumnText(0))
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("select logs: %v", err)
	}

	return logs, nil
}

func (ls *LogStore) subscribe(id string) chan struct{} {
	ls.trackingMu.Lock()
	defer ls.trackingMu.Unlock()

	subs, ok := ls.subscribers[id]
	if !ok {
		return nil
	}

	ch := make(chan struct{})
	ls.subscribers[id] = append(subs, ch)
	return ch
}

func (ls *LogStore) notify(id string) {
	ls.trackingMu.Lock()
	defer ls.trackingMu.Unlock()
	subs, ok := ls.subscribers[id]
	if !ok {
		return
	}
	for _, ch := range subs {
		close(ch)
	}
	ls.subscribers[id] = subs[:0]
}

func (ls *LogStore) finalizeLogFile(id string, fname string) error {
	conn, err := ls.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer ls.dbpool.Put(conn)

	f, err := os.Open(filepath.Join(ls.inprogressDir, fname))
	if err != nil {
		return err
	}
	defer f.Close()

	var dataGz bytes.Buffer
	gzw := gzip.NewWriter(&dataGz)
	if _, e := io.Copy(gzw, f); e != nil {
		return fmt.Errorf("compress log: %v", e)
	}
	if e := gzw.Close(); e != nil {
		return fmt.Errorf("close gzip writer: %v", err)
	}

	if e := sqlitex.ExecuteTransient(conn, "UPDATE logs SET data_fname = NULL, data_gz = ? WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{dataGz.Bytes(), id},
	}); e != nil {
		return fmt.Errorf("update log: %v", e)
	} else if conn.Changes() != 1 {
		return fmt.Errorf("expected 1 row to be updated for %q, got %d", id, conn.Changes())
	}

	return nil
}

func (ls *LogStore) maybeReleaseTempFile(id, fname string) error {
	ls.trackingMu.Lock()
	defer ls.trackingMu.Unlock()

	_, ok := ls.refcount[id]
	if ok {
		return nil
	}
	return os.Remove(filepath.Join(ls.inprogressDir, fname))
}

type writer struct {
	ls      *LogStore
	id      string
	fname   string
	f       *os.File
	onClose sync.Once
}

var _ io.WriteCloser = (*writer)(nil)

func (w *writer) Write(p []byte) (n int, err error) {
	w.ls.mu.Lock(w.id)
	defer w.ls.mu.Unlock(w.id)
	n, err = w.f.Write(p)
	if n != 0 {
		w.ls.notify(w.id)
	}
	return
}

func (w *writer) Close() error {
	err := w.f.Close()

	w.onClose.Do(func() {
		w.ls.mu.Lock(w.id)
		defer w.ls.mu.Unlock(w.id)
		defer w.ls.notify(w.id)

		if e := w.ls.finalizeLogFile(w.id, w.fname); e != nil {
			err = multierror.Append(err, fmt.Errorf("finalize %v: %w", w.fname, e))
		} else {
			w.ls.refcount[w.id]--
		}

		// manually close all subscribers and delete the subscriber entry from the map; there are no more writes coming.
		w.ls.trackingMu.Lock()
		if w.ls.refcount[w.id] == 0 {
			delete(w.ls.refcount, w.id)
		}
		subs := w.ls.subscribers[w.id]
		for _, ch := range subs {
			close(ch)
		}
		delete(w.ls.subscribers, w.id)
		w.ls.trackingMu.Unlock()
		w.ls.maybeReleaseTempFile(w.id, w.fname)
	})

	return err
}

type reader struct {
	ls      *LogStore
	id      string
	fname   string
	f       *os.File
	onClose sync.Once
	closed  chan struct{} // unblocks any read calls e.g. can be used for early cancellation
}

var _ io.ReadCloser = (*reader)(nil)

func (r *reader) Read(p []byte) (n int, err error) {
	r.ls.mu.RLock(r.id)
	n, err = r.f.Read(p)
	if err == io.EOF {
		waiter := r.ls.subscribe(r.id)
		r.ls.mu.RUnlock(r.id)
		if waiter != nil {
			select {
			case <-waiter:
			case <-r.closed:
				return 0, io.EOF
			}
		}
		r.ls.mu.RLock(r.id)
		n, err = r.f.Read(p)
	}
	r.ls.mu.RUnlock(r.id)

	return
}

func (r *reader) Close() error {
	r.ls.mu.Lock(r.id)
	defer r.ls.mu.Unlock(r.id)

	err := r.f.Close()

	r.onClose.Do(func() {
		r.ls.trackingMu.Lock()
		r.ls.refcount[r.id]--
		if r.ls.refcount[r.id] == 0 {
			delete(r.ls.refcount, r.id)
		}
		r.ls.trackingMu.Unlock()
		r.ls.maybeReleaseTempFile(r.id, r.fname)
		close(r.closed)
	})

	return err
}
