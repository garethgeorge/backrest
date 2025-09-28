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
	_, err := store.dbpool.ExecContext(context.Background(), fmt.Sprintf("PRAGMA user_version = %d", version))
	if err != nil {
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
	var version int
	err := store.dbpool.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version)
	if err != nil {
		t.Fatalf("error verifying user_version: %s", err)
	}
	if version != sqlSchemaVersion {
		t.Fatalf("expected user_version %d, got %d", sqlSchemaVersion, version)
	}
}
