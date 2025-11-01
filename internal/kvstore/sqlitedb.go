package kvstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func NewSqliteDbForKvStore(db string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(db), 0700); err != nil {
		return nil, fmt.Errorf("create sqlite db directory: %v", err)
	}
	if !vfs.SupportsFileLocking {
		return nil, fmt.Errorf("file locking not supported")
	}
	dbpool, err := sql.Open("sqlite3", db)
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}
	if vfs.SupportsSharedMemory {
		_, err = dbpool.ExecContext(context.Background(), `
			PRAGMA journal_mode = WAL;
			PRAGMA synchronous = NORMAL;
		`)
		if err != nil {
			return nil, fmt.Errorf("run multiline query: %v", err)
		}
	}
	if runtime.GOOS == "darwin" {
		_, err = dbpool.ExecContext(context.Background(), "PRAGMA checkpoint_fullfsync = 1")
		if err != nil {
			return nil, fmt.Errorf("run multiline query: %v", err)
		}
	}
	return dbpool, nil
}

func NewInMemorySqliteDbForKvStore(t testing.TB) *sql.DB {
	dbpool, err := sql.Open("sqlite3", memdb.TestDB(t))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	_, err = dbpool.ExecContext(context.Background(), `
			PRAGMA journal_mode = WAL;
			PRAGMA synchronous = NORMAL;
		`)
	if err != nil {
		t.Fatalf("failed to set pragmas: %v", err)
	}
	return dbpool
}
