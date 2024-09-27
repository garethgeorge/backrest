package logwriter2

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var (
	ErrLogNotFound = fmt.Errorf("log not found")
)

type LogWriter struct {
	dir           string
	inprogressDir string
	mu            shardedRWMutex
	dbpool        *sqlitex.Pool

	trackingMu  sync.Mutex     // guards refcount and subscribers
	refcount    map[string]int // id : refcount
	subscribers map[string][]chan struct{}
}

func NewLogWriter(dir string) (*LogWriter, error) {
	dbpath := filepath.Join(dir, "logs.sqlite")
	dbpool, err := sqlitex.NewPool(dbpath, sqlitex.PoolOptions{
		PoolSize: 16,
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}

	lw := &LogWriter{
		dir:           dir,
		inprogressDir: filepath.Join(dir, ".inprogress"),
		mu:            newShardedRWMutex(64), // 64 shards should be enough to avoid much contention
		dbpool:        dbpool,
		subscribers:   make(map[string][]chan struct{}),
		refcount:      make(map[string]int),
	}

	return lw, lw.init()
}

func (lw *LogWriter) init() error {
	if err := os.MkdirAll(lw.inprogressDir, 0755); err != nil {
		return fmt.Errorf("create inprogress dir: %v", err)
	}

	conn, err := lw.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer lw.dbpool.Put(conn)

	if err := sqlitex.ExecuteScript(conn, `
		PRAGMA auto_vacuum = 1;
		PRAGMA journal_mode=WAL;

		CREATE TABLE IF NOT EXISTS logs (
			id TEXT PRIMARY KEY,
			ttl INTEGER, -- time-to-live in seconds
			owner_opid INTEGER, -- id of the operation that owns this log; will be used for cleanup.
			data_fname TEXT, -- relative path to the file containing the log data
			data_gz BLOB -- compressed log data as an alternative to data_fname
		);
		
		CREATE INDEX IF NOT EXISTS logs_data_fname_idx ON logs (data_fname);

		CREATE TABLE IF NOT EXISTS system_info (
			version INTEGER NOT NULL
		);
		
		-- Create a table to store the schema version, will be used for migrations in the future
		INSERT INTO system_info (version)
		SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM system_info);
	`, nil); err != nil {
		return fmt.Errorf("execute init script: %v", err)
	}

	// loop through all inprogress files and finalize them if they are in the database
	files, err := os.ReadDir(lw.inprogressDir)
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
			err := lw.finalizeLogFile(id, fname)
			if err != nil {
				zap.S().Warnf("sqlite log writer couldn't finalize dangling inprogress log file %v: %v", fname, err)
				continue
			}
		}
		if err := os.Remove(filepath.Join(lw.inprogressDir, fname)); err != nil {
			zap.S().Warnf("sqlite log writer couldn't remove dangling inprogress log file %v: %v", fname, err)
		}
	}

	return nil
}

func (lw *LogWriter) Close() error {
	return lw.dbpool.Close()
}

func (lw *LogWriter) Create(id string, parentOpID int64) (io.WriteCloser, error) {
	lw.mu.Lock(id)
	defer lw.mu.Unlock(id)

	conn, err := lw.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer lw.dbpool.Put(conn)

	// create a random file for the log while it's being written
	fname := uuid.New().String()
	f, err := os.Create(filepath.Join(lw.inprogressDir, fname))
	if err != nil {
		return nil, fmt.Errorf("create temp file: %v", err)
	}

	if err := sqlitex.ExecuteTransient(conn, "INSERT INTO logs (id, owner_opid, data_fname) VALUES (?, ?, ?)", &sqlitex.ExecOptions{
		Args: []any{id, parentOpID, fname},
	}); err != nil {
		return nil, fmt.Errorf("insert log: %v", err)
	}

	lw.trackingMu.Lock()
	fmt.Printf("creating subscriber list for %v\n", id)
	lw.subscribers[id] = make([]chan struct{}, 0)
	lw.refcount[id] = 1
	lw.trackingMu.Unlock()

	return &writer{
		lw:    lw,
		f:     f,
		fname: fname,
		id:    id,
	}, nil
}

func (lw *LogWriter) Open(id string) (io.ReadCloser, error) {
	lw.mu.Lock(id)
	defer lw.mu.Unlock(id)

	conn, err := lw.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer lw.dbpool.Put(conn)

	var found bool
	var fname string
	var dataGz []byte
	if err := sqlitex.ExecuteTransient(conn, "SELECT data_fname, data_gz FROM logs WHERE id = ?", &sqlitex.ExecOptions{
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
		fmt.Printf("fname: %v\n", fname)
		f, err := os.Open(filepath.Join(lw.inprogressDir, fname))
		if err != nil {
			return nil, fmt.Errorf("open data file: %v", err)
		}
		lw.trackingMu.Lock()
		lw.refcount[id]++
		lw.trackingMu.Unlock()

		return &reader{
			lw:     lw,
			f:      f,
			id:     id,
			fname:  fname,
			closed: make(chan struct{}),
		}, nil
	} else if dataGz != nil {
		fmt.Printf("dataGz: %v\n", dataGz)

		gzr, err := gzip.NewReader(bytes.NewReader(dataGz))
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %v", err)
		}

		return gzr, nil
	} else {
		return nil, errors.New("log has no associated data. This shouldn't be possible")
	}
}

func (lw *LogWriter) subscribe(id string) chan struct{} {
	lw.trackingMu.Lock()
	defer lw.trackingMu.Unlock()

	subs, ok := lw.subscribers[id]
	if !ok {
		fmt.Printf("nothing to subscribe to for %v\n", id)
		return nil
	}

	ch := make(chan struct{})
	lw.subscribers[id] = append(subs, ch)
	return ch
}

func (lw *LogWriter) notify(id string) {
	lw.trackingMu.Lock()
	defer lw.trackingMu.Unlock()
	subs, ok := lw.subscribers[id]
	if !ok {
		return
	}
	for _, ch := range subs {
		close(ch)
	}
	lw.subscribers[id] = subs[:0]
}

func (lw *LogWriter) finalizeLogFile(id string, fname string) error {
	conn, err := lw.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer lw.dbpool.Put(conn)

	f, err := os.Open(filepath.Join(lw.inprogressDir, fname))
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
		return fmt.Errorf("expected 1 row to be updated, got %d", conn.Changes())
	}

	return nil
}

func (lw *LogWriter) maybeReleaseTempFile(fname string) error {
	lw.trackingMu.Lock()
	defer lw.trackingMu.Unlock()

	_, ok := lw.refcount[fname]
	if ok {
		return nil
	}
	return os.Remove(filepath.Join(lw.inprogressDir, fname))
}

type writer struct {
	lw      *LogWriter
	id      string
	fname   string
	f       *os.File
	onClose sync.Once
}

var _ io.WriteCloser = (*writer)(nil)

func (w *writer) Write(p []byte) (n int, err error) {
	w.lw.mu.Lock(w.id)
	defer w.lw.mu.Unlock(w.id)
	n, err = w.f.Write(p)
	if n != 0 {
		w.lw.notify(w.id)
	}
	return
}

func (w *writer) Close() error {
	err := w.f.Close()

	w.onClose.Do(func() {
		w.lw.mu.Lock(w.id)
		defer w.lw.mu.Unlock(w.id)
		defer w.lw.notify(w.id)

		// manually close all subscribers and delete the subscriber entry from the map; there are no more writes coming.
		w.lw.trackingMu.Lock()
		subs := w.lw.subscribers[w.id]
		for _, ch := range subs {
			close(ch)
		}
		delete(w.lw.subscribers, w.id)

		// try to finalize the log file
		if e := w.lw.finalizeLogFile(w.id, w.fname); e != nil {
			err = multierror.Append(err, fmt.Errorf("finalize %v: %w", w.fname, e))
			return // if we fail to finalize, we return early so that we dangle the temp file; maybe we can try again later.
		}

		w.lw.refcount[w.id]--
		if w.lw.refcount[w.id] == 0 {
			delete(w.lw.refcount, w.id)
		}
		w.lw.trackingMu.Unlock()

		w.lw.maybeReleaseTempFile(w.fname)
	})

	return err
}

type reader struct {
	lw      *LogWriter
	id      string
	fname   string
	f       *os.File
	onClose sync.Once
	closed  chan struct{} // unblocks any read calls e.g. can be used for early cancellation
}

var _ io.ReadCloser = (*reader)(nil)

func (r *reader) Read(p []byte) (n int, err error) {
	r.lw.mu.RLock(r.id)
	n, err = r.f.Read(p)
	if err == io.EOF {
		waiter := r.lw.subscribe(r.id)
		r.lw.mu.RUnlock(r.id)
		if waiter != nil {
			select {
			case <-waiter:
			case <-r.closed:
				return 0, io.EOF
			}
		}
		r.lw.mu.RLock(r.id)
		n, err = r.f.Read(p)
	}
	r.lw.mu.RUnlock(r.id)

	return
}

func (r *reader) Close() error {
	r.lw.mu.Lock(r.id)
	defer r.lw.mu.Unlock(r.id)

	err := r.f.Close()

	r.onClose.Do(func() {
		r.lw.trackingMu.Lock()
		r.lw.refcount[r.id]--
		if r.lw.refcount[r.id] == 0 {
			delete(r.lw.refcount, r.id)
		}
		r.lw.trackingMu.Unlock()
		r.lw.maybeReleaseTempFile(r.fname)
		close(r.closed)
	})

	return err
}
