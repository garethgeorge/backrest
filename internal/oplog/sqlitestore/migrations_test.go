package sqlitestore

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
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

	testOps := []*v1.Operation{
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan1",
			RepoId:          "repo1",
			InstanceId:      "instance1",
			Op:              &v1.Operation_OperationBackup{},
			OriginalId:      1,
			OriginalFlowId:  2,
			Modno:           3,
			Status:          v1.OperationStatus_STATUS_INPROGRESS,
		},
	}

	store, err := NewSqliteStore(tempDir + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}

	// insert some test data
	if err := store.Add(testOps...); err != nil {
		t.Fatalf("error adding test data: %s", err)
	}

	gotOps := make([]*v1.Operation, 0)
	if err := store.Query(oplog.Query{}, func(op *v1.Operation) error {
		gotOps = append(gotOps, op)
		return nil
	}); err != nil {
		t.Fatalf("error querying sqlite store: %s", err)
	}

	if len(gotOps) != len(testOps) {
		t.Errorf("first check before migrations, expected %d operations, got %d", len(testOps), len(gotOps))
	}

	if err := store.Close(); err != nil {
		t.Fatalf("error closing sqlite store: %s", err)
	}

	// re-open the store
	store2, err := NewSqliteStore(tempDir + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}

	gotOps = gotOps[:0]
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

	t.Cleanup(func() { store2.Close() })
}
