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
	tempDir := t.TempDir()
	store, err := NewSqliteStore(tempDir + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}
	t.Cleanup(func() { store.Close() })
}

func TestMigrateExisting(t *testing.T) {
	testutil.InstallZapLogger(t)
	tempDir := t.TempDir()

	testOps := []*v1.Operation{}
	for i := 0; i < 10; i++ {
		testOps = append(testOps, testutil.RandomOperation())
	}

	store, err := NewSqliteStore(tempDir + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}

	// insert some test data
	if err := store.Add(testOps...); err != nil {
		t.Fatalf("error adding test data: %s", err)
	}

	// Run a query to set the user_version to something that will trigger a migration
	conn, err := store.dbpool.Take(context.Background())
	if err != nil {
		t.Fatalf("error getting connection: %s", err)
	}
	if err := sqlitex.ExecuteTransient(conn, "PRAGMA user_version = 192393", nil); err != nil {
		t.Fatalf("error setting user_version: %s", err)
	}
	store.dbpool.Put(conn)

	if err := store.Close(); err != nil {
		t.Fatalf("error closing sqlite store: %s", err)
	}

	// re-open the store
	store2, err := NewSqliteStore(tempDir + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}

	gotOps := make([]*v1.Operation, 0)
	if err := store2.Query(oplog.Query{}, func(op *v1.Operation) error {
		gotOps = append(gotOps, op)
		return nil
	}); err != nil {
		t.Fatalf("error querying sqlite store: %s", err)
	}

	if len(gotOps) != len(testOps) {
		t.Errorf("expected %d operations, got %d", len(testOps), len(gotOps))
	}

	if diff := cmp.Diff(
		&v1.OperationList{Operations: gotOps},
		&v1.OperationList{Operations: testOps},
		protocmp.Transform()); diff != "" {
		t.Errorf("unexpected diff in operations back after migration: %v", diff)
	}

	// Finally, verify that the user_version is set to the latest version
	conn, err = store2.dbpool.Take(context.Background())
	if err != nil {
		t.Fatalf("error getting connection: %s", err)
	}
	defer store2.dbpool.Put(conn)
	if err := sqlitex.ExecuteTransient(conn, "PRAGMA user_version", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			if stmt.ColumnInt(0) != sqlSchemaVersion {
				return fmt.Errorf("expected user_version %d, got %d", sqlSchemaVersion, stmt.ColumnInt(0))
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("error getting user_version: %s", err)
	}

	t.Cleanup(func() { store2.Close() })
}
