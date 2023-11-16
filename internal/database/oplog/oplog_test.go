package oplog

import (
	"slices"
	"testing"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("error closing oplog: %s", err)
	}
}

func TestAddOperation(t *testing.T) {
	t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	defer log.Close()

	var tests = []struct {
		name string
		op *v1.Operation
		wantErr bool
	}{
		{
			name: "no operation",
			op: &v1.Operation{
				Id: 0,
			},
			wantErr: true,
		},
		{
			name: "basic backup operation",
			op: &v1.Operation{
				Id: 0,
				Op: &v1.Operation_OperationBackup{},
			},
			wantErr: false,
		},
		{
			name: "basic snapshot operation",
			op: &v1.Operation{
				Id: 0,
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
			name: "basic snapshot operation with no snapshot",
			op: &v1.Operation{
				Id: 0,
				Op: &v1.Operation_OperationIndexSnapshot{
					OperationIndexSnapshot: &v1.OperationIndexSnapshot{
					},
				},
			},
			wantErr: true,
		},
		{
			name: "operation with ID",
			op: &v1.Operation{
				Id: 1,
				Op: &v1.Operation_OperationBackup{},
			},
			wantErr: true,
		},
		{
			name: "operation with repo",
			op: &v1.Operation{
				Id: 0,
				RepoId: "testrepo",
				Op: &v1.Operation_OperationBackup{},
			},
		},
		{
			name: "operation with plan",
			op: &v1.Operation{
				Id: 0,
				PlanId: "testplan",
				Op: &v1.Operation_OperationBackup{},
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
	t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}

	// these should get assigned IDs 1-3 respectively by the oplog
	ops := []*v1.Operation{
		{
			PlanId: "plan1",
			RepoId: "repo1",
			DisplayMessage: "op1",
			Op: &v1.Operation_OperationBackup{},
		},
		{
			PlanId: "plan1",
			RepoId: "repo2",
			DisplayMessage: "op2",
			Op: &v1.Operation_OperationBackup{},
		},
		{
			PlanId: "plan2",
			RepoId: "repo2",
			DisplayMessage: "op3",
			Op: &v1.Operation_OperationBackup{},
		},
	}

	for _, op := range ops {
		if err := log.Add(op); err != nil {
			t.Fatalf("error adding operation: %s", err)
		}
	}

	tests := []struct {
		name string
		byPlan bool 
		byRepo bool 
		id string 
		expected []string
	}{
		{
			name: "list plan1",
			byPlan: true,
			id: "plan1",
			expected: []string{"op1", "op2"},
		},
		{
			name: "list plan2",
			byPlan: true,
			id: "plan2",
			expected: []string{"op3"},
		},
		{
			name: "list repo1",
			byRepo: true,
			id: "repo1",
			expected: []string{"op1"},
		},
		{
			name: "list repo2",
			byRepo: true,
			id: "repo2",
			expected: []string{"op2", "op3"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ops []*v1.Operation
			var err error
			if tc.byPlan {
				ops, err = log.GetByPlan(tc.id, FilterKeepAll())
			} else if tc.byRepo {
				ops, err = log.GetByRepo(tc.id, FilterKeepAll())
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
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}

	for i := 0; i < 1000; i++ {
		if err := log.Add(&v1.Operation{
			PlanId: "plan1",
			RepoId: "repo1",
			Op: &v1.Operation_OperationBackup{},
		}); err != nil {
			t.Fatalf("error adding operation: %s", err)
		}
	}

	ops, err := log.GetByPlan("plan1", FilterKeepAll())
	if err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if len(ops) != 1000 {
		t.Errorf("want 1000 operations, got %d", len(ops))
	}

	ops, err = log.GetByRepo("repo1", FilterKeepAll())
	if err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if len(ops) != 1000 {
		t.Errorf("want 1000 operations, got %d", len(ops))
	}
}

func TestIndexSnapshot(t *testing.T) {
	t.Parallel()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}

	op := &v1.Operation{
		PlanId: "plan1",
		RepoId: "repo1",
		Op: &v1.Operation_OperationIndexSnapshot{
			OperationIndexSnapshot: &v1.OperationIndexSnapshot{
				Snapshot: &v1.ResticSnapshot{
					Id: "abcdefghijklmnop",
				},
			},
		},
	}
	if err := log.Add(op); err != nil {
		t.Fatalf("error adding operation: %s", err)
	}

	id, err := log.HasIndexedSnapshot("abcdefgh")
	if err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	}
	if id != op.Id {
		t.Fatalf("want id %d, got %d", op.Id, id)
	}
	
	id, err = log.HasIndexedSnapshot("notfound")
	if err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	}
	if id != -1 {
		t.Fatalf("want id -1, got %d", id)
	}
}

func collectMessages(ops []*v1.Operation) []string {
	var messages []string
	for _, op := range ops {
		messages = append(messages, op.DisplayMessage)
	}
	return messages
}