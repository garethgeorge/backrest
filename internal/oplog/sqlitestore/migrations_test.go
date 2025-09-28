package sqlitestore

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func TestNewSqliteStore(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	store, err := NewSqliteStore(tempDir + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}
	t.Cleanup(func() { store.Close() })
}

func TestMigrateExisting(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.sqlite"

	testOps := make([]*v1.Operation, 0, 10)
	for i := 0; i < 10; i++ {
		testOps = append(testOps, testutil.RandomOperation())
	}

	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}
	if err := store.Add(testOps...); err != nil {
		t.Fatalf("error adding test data: %s", err)
	}

	setSchemaVersion(t, store, 192393)

	if err := store.Close(); err != nil {
		t.Fatalf("error closing sqlite store: %s", err)
	}

	store2, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}
	defer store2.Close()

	gotOps := queryAllOperations(t, store2)

	if len(gotOps) != len(testOps) {
		t.Errorf("expected %d operations, got %d", len(testOps), len(gotOps))
	}

	if diff := cmp.Diff(
		&v1.OperationList{Operations: gotOps},
		&v1.OperationList{Operations: testOps},
		protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff in operations after migration: %v", diff)
	}

	verifySchemaVersion(t, store2)
}

func setSchemaVersion(t *testing.T, store *SqliteStore, version int) {
	conn, err := store.dbpool.Take(context.Background())
	if err != nil {
		t.Fatalf("error getting connection: %s", err)
	}
	defer store.dbpool.Put(conn)

	if err := sqlitex.ExecuteTransient(conn, fmt.Sprintf("PRAGMA user_version = %d", version), nil); err != nil {
		t.Fatalf("error setting user_version: %s", err)
	}
}

func queryAllOperations(t *testing.T, store *SqliteStore) []*v1.Operation {
	gotOps := make([]*v1.Operation, 0)
	if err := store.Query(oplog.Query{}, func(op *v1.Operation) error {
		gotOps = append(gotOps, op)
		return nil
	}); err != nil {
		t.Fatalf("error querying sqlite store: %s", err)
	}
	return gotOps
}

func verifySchemaVersion(t *testing.T, store *SqliteStore) {
	conn, err := store.dbpool.Take(context.Background())
	if err != nil {
		t.Fatalf("error getting connection: %s", err)
	}
	defer store.dbpool.Put(conn)

	if err := sqlitex.ExecuteTransient(conn, "PRAGMA user_version", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			if stmt.ColumnInt(0) != sqlSchemaVersion {
				return fmt.Errorf("expected user_version %d, got %d", sqlSchemaVersion, stmt.ColumnInt(0))
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("error verifying user_version: %s", err)
	}
}
