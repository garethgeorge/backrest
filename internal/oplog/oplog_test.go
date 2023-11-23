package oplog

import (
	"slices"
	"testing"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
)

type MyStruct struct {
	ID    string `storm:"id"`
	Field string `storm:"index"`
}

func TestDatabase(t *testing.T) {
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}

	t.Run("can store struct", func(t *testing.T) {
		log.sdb.From("struct").Save(&MyStruct{ID: "1", Field: "test"})

		var s MyStruct
		if err := log.sdb.From("struct").One("Field", "test", &s); err != nil {
			t.Fatalf("error getting struct: %s", err)
		}
		t.Logf("Got struct: %+v", s)
		if s.ID != "1" {
			t.Errorf("want ID 1, got %s", s.ID)
		}
		if s.Field != "test" {
			t.Errorf("want field test, got %s", s.Field)
		}
	})

	t.Run("can store proto", func(t *testing.T) {
		log.sdb.From("proto").Save(&v1.Operation{Id: 1, PlanId: "foo", RepoId: "bar"})

		var op v1.Operation
		if err := log.sdb.From("proto").One("PlanId", "foo", &op); err != nil {
			t.Fatalf("error getting operation: %s", err)
		}
		t.Logf("Got operation: %+v", op)
		if op.Id != 1 {
			t.Errorf("want ID 1, got %d", op.Id)
		}

		var ops []v1.Operation
		if err := log.sdb.From("proto").Find("RepoId", "bar", &ops); err != nil {
			t.Fatalf("error getting operations: %s", err)
		}
		t.Logf("Got operations: %+v", ops)
		if len(ops) != 1 {
			t.Errorf("want 1 operation, got %d", len(ops))
		}
	})
}

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
			name: "basic backup operation",
			op: &v1.Operation{
				Id: 0,
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
			name: "operation with ID",
			op: &v1.Operation{
				Id: 1,
			},
			wantErr: true,
		},
		{
			name: "operation with repo",
			op: &v1.Operation{
				Id:     0,
				RepoId: "testrepo",
			},
		},
		{
			name: "operation with plan",
			op: &v1.Operation{
				Id:     0,
				PlanId: "testplan",
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

				got, err := log.Get(tc.op.Id)
				if err != nil {
					t.Errorf("Get() error = %v", err)
				}
				if got.Id != tc.op.Id {
					t.Errorf("Get() got = %+v, want %+v", got, tc.op)
				}
			}
		})
	}
}

func TestListOperation(t *testing.T) {
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	// these should get assigned IDs 1-3 respectively by the oplog
	ops := []*v1.Operation{
		{
			PlanId:         "plan1",
			RepoId:         "repo1",
			DisplayMessage: "op1",
		},
		{
			PlanId:         "plan1",
			RepoId:         "repo2",
			DisplayMessage: "op2",
		},
		{
			PlanId:         "plan2",
			RepoId:         "repo2",
			DisplayMessage: "op3",
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
	t.Skip()
	log, err := NewOpLog(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating oplog: %s", err)
	}
	t.Cleanup(func() { log.Close() })

	for i := 0; i < 100; i++ {
		if err := log.Add(&v1.Operation{
			PlanId: "plan1",
			RepoId: "repo1",
			Op:     &v1.Operation_OperationBackup{},
		}); err != nil {
			t.Fatalf("error adding operation: %s", err)
		}
	}

	ops, err := log.GetByPlan("plan1", FilterKeepAll())
	if err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if len(ops) != 100 {
		t.Errorf("want 100 operations, got %d", len(ops))
	}

	ops, err = log.GetByRepo("repo1", FilterKeepAll())
	if err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if len(ops) != 100 {
		t.Errorf("want 100 operations, got %d", len(ops))
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
		PlanId:     "plan1",
		RepoId:     "repo1",
		SnapshotId: NormalizeSnapshotId("abcdefghijklmnop"),
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

	ops, err := log.GetBySnapshotId("abcdefgh")
	if err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	}
	if len(ops) != 1 {
		t.Fatalf("want 1 operation, got %d", len(ops))
	}
	if ops[0].Id != op.Id {
		t.Fatalf("want id %d, got %d", op.Id, ops[0].Id)
	}

	ops, err = log.GetBySnapshotId("notfound")
	if err != nil {
		t.Fatalf("error checking for snapshot: %s", err)
	}
	if len(ops) != 0 {
		t.Fatalf("want 0 operations, got %d", len(ops))
	}
}

func collectMessages(ops []*v1.Operation) []string {
	var messages []string
	for idx, _ := range ops {
		messages = append(messages, ops[idx].DisplayMessage)
	}
	return messages
}
