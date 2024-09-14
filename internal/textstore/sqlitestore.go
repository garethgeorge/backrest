package textstore

import (
	"context"
	"fmt"
	"io"
	"sync"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const sqliteWriterChunkSize = 32 * 1024 // use a 32KB chunk size

type SqliteTextStore struct {
	mu     sync.Mutex
	dbpool *sqlitex.Pool

	// writeSubscribers is a set of channels that are closed when a write occurs for a file.
	writeSubscribers map[string][]chan struct{}
}

func NewSqliteTextStore(uri string) (*SqliteTextStore, error) {
	dbpool, err := sqlitex.NewPool(uri, sqlitex.PoolOptions{PoolSize: 16})
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}
	store := &SqliteTextStore{
		dbpool:           dbpool,
		writeSubscribers: make(map[string][]chan struct{}),
	}
	return store, store.init()
}

func (s *SqliteTextStore) init() error {
	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer s.dbpool.Put(conn)

	statements := []string{
		`PRAGMA auto_vacuum = 1;`, // enable shrinking the database on DELETE
		`CREATE TABLE IF NOT EXISTS files (
			storage_id INTEGER PRIMARY KEY, -- unique identifier
			id TEXT NOT NULL UNIQUE -- external identifier
		);`,
		`CREATE TABLE IF NOT EXISTS blobs (
			storage_id INTEGER,
			offset INTEGER NOT NULL,
			size INTEGER NOT NULL,
			data BLOB NOT NULL,
			compressed INTEGER DEFAULT 0, -- 1 if data is compressed
		 	PRIMARY KEY (storage_id, offset),
			FOREIGN KEY (storage_id) REFERENCES files(storage_id) ON DELETE CASCADE
		);`,
	}

	for _, stmt := range statements {
		if err := sqlitex.ExecuteTransient(conn, stmt, nil); err != nil {
			return fmt.Errorf("create initial tables: %v", err)
		}
	}

	return nil
}

func (s *SqliteTextStore) notifySubscribers(id string) {
	if subs, ok := s.writeSubscribers[id]; ok {
		for _, ch := range subs {
			close(ch)
		}
		s.writeSubscribers[id] = subs[:0] // clear slice
	}
}

func (s *SqliteTextStore) subscribe(id string) chan struct{} {
	subs, ok := s.writeSubscribers[id]
	if !ok {
		return nil
	}
	ch := make(chan struct{})
	s.writeSubscribers[id] = append(subs, ch)
	return ch
}

func (s *SqliteTextStore) Create(id string) (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer s.dbpool.Put(conn)

	if err := sqlitex.ExecuteTransient(conn, `INSERT INTO files (id) VALUES (?)`, &sqlitex.ExecOptions{Args: []any{id}}); err != nil {
		return nil, fmt.Errorf("insert file %q: %v", id, err)
	}
	storageId := conn.LastInsertRowID()

	s.writeSubscribers[id] = make([]chan struct{}, 0, 1)

	return &sqliteStoreWriter{
		store:     s,
		storageId: storageId,
		id:        id,
		chunkSize: sqliteWriterChunkSize,
	}, nil
}

func (s *SqliteTextStore) Open(id string) (io.ReadCloser, error) {
	return s.openInternal(id, true)
}

func (s *SqliteTextStore) openInternal(id string, streaming bool) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("take connection: %v", err)
	}
	defer s.dbpool.Put(conn)

	var storageId int64
	if err := sqlitex.ExecuteTransient(conn, `SELECT storage_id FROM files WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			storageId = stmt.ColumnInt64(0)
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("read file %q: %v", id, err)
	}

	return &sqliteStoreReader{
		store:     s,
		id:        id,
		storageId: storageId,
		streaming: streaming,
	}, nil
}

func (s *SqliteTextStore) Close() error {
	return s.dbpool.Close()
}

type sqliteStoreReader struct {
	store     *SqliteTextStore
	id        string
	storageId int64
	pos       int64

	curBlob   []byte
	curOffset int64

	streaming bool
}

var _ io.ReadCloser = (*sqliteStoreReader)(nil)

func (r *sqliteStoreReader) Read(p []byte) (n int, err error) {
	n, err = r.readInternal(p)
	if err == io.EOF && r.streaming {
		// check if we need to subscribe and wait
		r.store.mu.Lock()
		if _, ok := r.store.writeSubscribers[r.id]; ok {
			ch := r.store.subscribe(r.id)
			r.store.mu.Unlock()
			<-ch
			return r.readInternal(p)
		}
		r.store.mu.Unlock()
		return n, io.EOF
	}
	return
}

func (r *sqliteStoreReader) readInternal(p []byte) (n int, err error) {
	// service the read from the cached blob if we can
	blobPos := r.pos - r.curOffset
	if blobPos < int64(len(r.curBlob)) {
		n = copy(p, r.curBlob[blobPos:])
		r.pos += int64(n)
		return n, nil
	}

	conn, err := r.store.dbpool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("take connection: %v", err)
	}
	defer r.store.dbpool.Put(conn)

	r.curBlob = nil
	r.curOffset = 0

	// read the next blob
	if err := sqlitex.ExecuteTransient(conn, `SELECT offset, size, compressed, data FROM blobs WHERE storage_id = ? AND offset <= ? AND offset + size > ?`, &sqlitex.ExecOptions{
		Args: []any{r.storageId, r.pos, r.pos},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			offset := stmt.ColumnInt64(0)
			size := stmt.ColumnInt64(1)
			compressionMode := stmt.ColumnInt64(2)
			data := make([]byte, stmt.ColumnLen(3))
			stmt.ColumnBytes(3, data)

			// decompress if needed
			if compressionMode == 1 {
				var err error
				data, err = decompress(data)
				if err != nil {
					return fmt.Errorf("decompress blob: %v", err)
				}
			}

			data = headBytes(data, int(size))

			r.curBlob = data
			r.curOffset = offset
			return nil
		},
	}); err != nil {
		return 0, fmt.Errorf("read blob: %v", err)
	}

	blobPos = r.pos - r.curOffset
	if blobPos < int64(len(r.curBlob)) {
		n = copy(p, r.curBlob[blobPos:])
		r.pos += int64(n)
		return n, nil
	} else {
		return 0, io.EOF
	}
}

func (r *sqliteStoreReader) Close() error {
	return nil
}

type sqliteStoreWriter struct {
	store     *SqliteTextStore
	storageId int64
	id        string

	chunkRowID     int64
	chunkOffset    int64
	chunkSize      int64 // chunk size doubles each time up to 16KB
	chunkBytesLeft int64
}

var _ io.WriteCloser = (*sqliteStoreWriter)(nil)

// TODO: try to batch writes
func (w *sqliteStoreWriter) Write(p []byte) (written int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	var conn *sqlite.Conn
	conn, err = w.store.dbpool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("take connection: %v", err)
	}
	defer w.store.dbpool.Put(conn)

	err = withTransaction(conn, func() error {
		writeNow := headBytes(p, int(w.chunkBytesLeft))
		p = tailBytes(p, int(w.chunkBytesLeft))

		if len(writeNow) > 0 && w.chunkRowID != 0 {
			// append to the current chunk
			n, err := writeBlob(conn, w.chunkRowID, w.chunkSize-w.chunkBytesLeft, writeNow)
			if err != nil {
				return fmt.Errorf("write blob: %v", err)
			} else if n != len(writeNow) {
				return fmt.Errorf("write blob: wrote %d bytes, want %d", n, len(writeNow))
			}

			// update the chunk size
			if err := sqlitex.ExecuteTransient(conn, `UPDATE blobs SET size = ? WHERE rowid = ?`, &sqlitex.ExecOptions{
				Args: []any{w.chunkSize - w.chunkBytesLeft + int64(len(writeNow)), w.chunkRowID},
			}); err != nil {
				return fmt.Errorf("update blob size: %v", err)
			}
			w.chunkBytesLeft -= int64(len(writeNow))
			written += len(writeNow)
		}

		// create new chunks for any data that overflows the last in-progress chunk
		for len(p) > 0 {
			writeNow = headBytes(p, int(w.chunkSize))
			p = tailBytes(p, int(w.chunkSize))

			if err := sqlitex.ExecuteTransient(conn, `INSERT INTO blobs (storage_id, offset, size, data) VALUES (?, ?, ?, zeroblob(?))`, &sqlitex.ExecOptions{
				Args: []any{w.storageId, w.chunkOffset, len(writeNow), w.chunkSize},
			}); err != nil {
				return fmt.Errorf("write blob: %v", err)
			}

			w.chunkRowID = conn.LastInsertRowID()
			n, err := writeBlob(conn, w.chunkRowID, 0, writeNow) // note: returned count is ignored, can't be easily handled
			if err != nil {
				return fmt.Errorf("write blob: %v", err)
			} else if n != len(writeNow) {
				return fmt.Errorf("write blob: wrote %d bytes, want %d", n, len(writeNow))
			}

			w.chunkOffset += w.chunkSize
			w.chunkBytesLeft = w.chunkSize - int64(len(writeNow))
			written += len(writeNow)
		}

		return nil
	})

	// notify subscribers
	w.store.mu.Lock()
	w.store.notifySubscribers(w.id)
	w.store.mu.Unlock()
	return
}

// close consolidates all the blobs into a single blob and notifies subscribers that the file is now in a consistent state.
func (w *sqliteStoreWriter) Close() error {
	conn, err := w.store.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take connection: %v", err)
	}
	defer w.store.dbpool.Put(conn)

	// select all blobs for the file
	var blobRowIDs []int64
	if err := sqlitex.ExecuteTransient(conn, `SELECT rowid FROM blobs WHERE storage_id = ? AND compressed != 1`, &sqlitex.ExecOptions{
		Args: []any{w.storageId},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			blobRowIDs = append(blobRowIDs, stmt.ColumnInt64(0))
			return nil
		},
	}); err != nil {
		return fmt.Errorf("read blobs: %v", err)
	}

	// compress all blobs
	for _, rowID := range blobRowIDs {
		if err := compressBlob(conn, rowID); err != nil {
			return fmt.Errorf("compress chunk: %v", err)
		}
	}

	// finally notify subscribers that the file is in its final consistent state.
	w.store.mu.Lock()
	w.store.notifySubscribers(w.id)
	delete(w.store.writeSubscribers, w.id)
	w.store.mu.Unlock()

	return nil
}

func compressBlob(conn *sqlite.Conn, rowID int64) error {
	// open the last chunk
	blobBytes, err := readBlob(conn, rowID)
	if err != nil {
		return fmt.Errorf("read blob: %v", err)
	}

	compressedData, err := compress(blobBytes)
	if err != nil {
		return fmt.Errorf("compress blob: %v", err)
	}

	// compress the chunk
	if err := sqlitex.ExecuteTransient(conn, `UPDATE blobs SET data = ?, compressed = 1 WHERE rowid = ?`, &sqlitex.ExecOptions{
		Args: []any{compressedData, rowID},
	}); err != nil {
		return fmt.Errorf("update blob: %v", err)
	}

	return nil
}

func readBlob(conn *sqlite.Conn, blobRowID int64) ([]byte, error) {
	blob, err := conn.OpenBlob("", "blobs", "data", blobRowID, false)
	if err != nil {
		return nil, fmt.Errorf("open blob: %v", err)
	}
	defer blob.Close()

	return io.ReadAll(blob)
}

func writeBlob(conn *sqlite.Conn, blobRowID int64, pos int64, bytes []byte) (n int, err error) {
	blob, err := conn.OpenBlob("", "blobs", "data", blobRowID, true)
	if err != nil {
		return 0, fmt.Errorf("open blob: %v", err)
	}
	if pos != 0 {
		if _, err = blob.Seek(pos, io.SeekStart); err != nil {
			_ = blob.Close()
			return
		}
	}
	n, err = blob.Write(bytes)
	if err != nil {
		_ = blob.Close()
		return
	}
	err = blob.Close()
	return
}

func withTransaction(conn *sqlite.Conn, f func() error) error {
	var err error
	endFunc := sqlitex.Transaction(conn)
	err = f()
	endFunc(&err)
	return err
}
