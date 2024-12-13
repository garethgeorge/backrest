package sqlitestore

import (
	"context"
	"path/filepath"
	"testing"
)

func TestBackupRestore(t *testing.T) {
	tempDir := t.TempDir()
	opstore, err := NewSqliteStore(filepath.Join(tempDir, "oplog.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { opstore.Close() })

	conn := opstore.dbpool.Get(context.Background())
	t.Cleanup(func() { opstore.dbpool.Put(conn) })

	if err := createBackupFile(conn, filepath.Join(tempDir, "backup.binpb")); err != nil {
		t.Fatal(err)
	}

}
