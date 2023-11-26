package oplog

import (
	"slices"
	"testing"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/oplog/indexutil"
)

func TestCreate(t *testing.T) {
	// t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	t.Cleanup(func() { log.Close() })
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("error closing oplog: %s", err)
	}
}

func TestAddOperation(t *testing.T) {
	// t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	var tests = []struct {
		name    string
		op      *v1.Operation
		wantErr bool
	}{
		{
			name: "basic operation",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
			},
			wantErr: false,
		},
		{
			name: "basic backup operation",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: false,
		},
		{
			name: "basic snapshot operation",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				Op: &v1.Operation_OperationIndexSnapshot{
					OperationIndexSnapshot: &v1.OperationIndexSnapshot{
						Snapshot: &v1.ResticSnapshot{
							Id: "test",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "operation with ID",
			op: &v1.Operation{
				Id:              1,
				UnixTimeStartMs: 1234,
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: true,
		},
		{
			name: "operation with repo",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				RepoId:          "testrepo",
				Op:              &v1.Operation_OperationBackup{},
			},
		},
		{
			name: "operation with plan",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				PlanId:          "testplan",
				Op:              &v1.Operation_OperationBackup{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := log.Add(tc.op); (err != nil) != tc.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if tc.op.Id == 0 {
					t.Errorf("Add() did not set op ID")
				}
			}
		})
	}
}

func TestListOperation(t *testing.T) {
	// t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	// these should get assigned IDs 1-3 respectively by the oplog
	ops := []*v1.Operation{
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan1",
			RepoId:          "repo1",
			DisplayMessage:  "op1",
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan1",
			RepoId:          "repo2",
			DisplayMessage:  "op2",
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan2",
			RepoId:          "repo2",
			DisplayMessage:  "op3",
			Op:              &v1.Operation_OperationBackup{},
		},
	}

	for _, op := range ops {
		if err := log.Add(op); err != nil {
			t.Fatalf("error adding operation: %s", err)
		}
	}

	tests := []struct {
		name     string
		byPlan   bool
		byRepo   bool
		id       string
		expected []string
	}{
		{
			name:     "list plan1",
			byPlan:   true,
			id:       "plan1",
			expected: []string{"op1", "op2"},
		},
		{
			name:     "list plan2",
			byPlan:   true,
			id:       "plan2",
			expected: []string{"op3"},
		},
		{
			name:     "list repo1",
			byRepo:   true,
			id:       "repo1",
			expected: []string{"op1"},
		},
		{
			name:     "list repo2",
			byRepo:   true,
			id:       "repo2",
			expected: []string{"op2", "op3"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()
			var ops []*v1.Operation
			var err error
			if tc.byPlan {
				ops, err = log.GetByPlan(tc.id, indexutil.CollectAll())
			} else if tc.byRepo {
				ops, err = log.GetByRepo(tc.id, indexutil.CollectAll())
			} else {
				t.Fatalf("must specify byPlan or byRepo")
			}
			if err != nil {
				t.Fatalf("error listing operations: %s", err)
			}
			got := collectMessages(ops)
			if slices.Compare(got, tc.expected) != 0 {
				t.Errorf("want operations: %v, got unexpected operations: %v", tc.expected, got)
			}
		})
	}
}

func TestBigIO(t *testing.T) {
	t.Parallel()

	count := 10

	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	for i := 0; i < count; i++ {
		if err := log.Add(&v1.Operation{
			UnixTimeStartMs: 1234,
			PlanId:          "plan1",
			RepoId:          "repo1",
			Op:              &v1.Operation_OperationBackup{},
		}); err != nil {
			t.Fatalf("error adding operation: %s", err)
		}
	}

	ops, err := log.GetByPlan("plan1", indexutil.CollectAll())
	if err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if len(ops) != count {
		t.Errorf("want %v operations, got %d", count, len(ops))
	}

	ops, err = log.GetByRepo("repo1", indexutil.CollectAll())
	if err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if len(ops) != count {
		t.Errorf("want %v operations, got %d", count, len(ops))
	}
}

func TestIndexSnapshot(t *testing.T) {
	t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	op := &v1.Operation{
		UnixTimeStartMs: 1234,
		PlanId:          "plan1",
		RepoId:          "repo1",
		SnapshotId:      "abcdefgh",
		Op:              &v1.Operation_OperationIndexSnapshot{},
	}
	if err := log.Add(op); err != nil {
		t.Fatalf("error adding operation: %s", err)
	}

	ops, err := log.GetBySnapshotId("abcdefgh", indexutil.CollectAll())
	if err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	}
	if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}
	if ops[0].Id != op.Id {
		t.Errorf("want operation ID %d, got %d", op.Id, ops[0].Id)
	}
}

func TestUpdateOperation(t *testing.T) {
	t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	// Insert initial operation
	op := &v1.Operation{
		UnixTimeStartMs: 1234,
		PlanId:          "oldplan",
		RepoId:          "oldrepo",
		SnapshotId:      "12345678",
	}
	if err := log.Add(op); err != nil {
		t.Fatalf("error adding operation: %s", err)
	}
	opId := op.Id

	// Validate initial values are indexed
	if ops, err := log.GetByPlan("oldplan", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for plan: %s", err)
	} else if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}
	if ops, err := log.GetByRepo("oldrepo", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for repo: %s", err)
	} else if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}
	if ops, err := log.GetBySnapshotId("12345678", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	} else if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}

	// Update indexed values
	op.SnapshotId = "abcdefgh"
	op.PlanId = "myplan"
	op.RepoId = "myrepo"
	if err := log.Update(op); err != nil {
		t.Fatalf("error updating operation: %s", err)
	}

	// Validate updated values are indexed
	if opId != op.Id {
		t.Errorf("want operation ID %d, got %d", opId, op.Id)
	}
	if ops, err := log.GetBySnapshotId("abcdefgh", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	} else if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}

	if ops, err := log.GetByPlan("myplan", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for plan: %s", err)
	} else if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}

	if ops, err := log.GetByRepo("myrepo", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for repo: %s", err)
	} else if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}

	// Validate prior values are gone
	if ops, err := log.GetByPlan("oldplan", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for plan: %s", err)
	} else if len(ops) != 0 {
		t.Fatalf("want 0 operations, got %d", len(ops))
	}
	if ops, err := log.GetByRepo("oldrepo", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for repo: %s", err)
	} else if len(ops) != 0 {
		t.Fatalf("want 0 operations, got %d", len(ops))
	}
	if ops, err := log.GetBySnapshotId("12345678", indexutil.CollectAll()); err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	} else if len(ops) != 0 {
		t.Fatalf("want 0 operations, got %d", len(ops))
	}
}

func collectMessages(ops []*v1.Operation) []string {
	var messages []string
	for _, op := range ops {
		messages = append(messages, op.DisplayMessage)
	}
	return messages
}
