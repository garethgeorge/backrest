package conformance

import (
	"fmt"
	"slices"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"google.golang.org/protobuf/proto"
)

const (
	snapshotId  = "1234567890123456789012345678901234567890123456789012345678901234"
	snapshotId2 = "abcdefgh01234567890123456789012345678901234567890123456789012345"
)

func StoresForTest(t testing.TB) map[string]oplog.OpStore {
	bboltstore, err := bboltstore.NewBboltStore(t.TempDir() + "/test.boltdb")
	if err != nil {
		t.Fatalf("error creating bbolt store: %s", err)
	}

	t.Cleanup(func() { bboltstore.Close() })

	return map[string]oplog.OpStore{
		"bbolt":  bboltstore,
		"memory": memstore.NewMemStore(),
	}
}

func TestCreate(t *testing.T) {
	// t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			_ = oplog.NewOpLog(store)
		})
	}
}

func TestAddOperation(t *testing.T) {
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
			wantErr: true,
		},
		{
			name: "basic backup operation",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				RepoId:          "testrepo",
				PlanId:          "testplan",
				InstanceId:      "testinstance",
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: false,
		},
		{
			name: "basic snapshot operation",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				RepoId:          "testrepo",
				PlanId:          "testplan",
				InstanceId:      "testinstance",
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
				RepoId:          "testrepo",
				PlanId:          "testplan",
				InstanceId:      "testinstance",
				UnixTimeStartMs: 1234,
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: true,
		},
		{
			name: "operation with repo only",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				RepoId:          "testrepo",
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: true,
		},
		{
			name: "operation with plan only",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				PlanId:          "testplan",
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: true,
		},
		{
			name: "operation with instance only",
			op: &v1.Operation{
				UnixTimeStartMs: 1234,
				InstanceId:      "testinstance",
				Op:              &v1.Operation_OperationBackup{},
			},
			wantErr: true,
		},
	}
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					log := oplog.NewOpLog(store)
					op := proto.Clone(tc.op).(*v1.Operation)
					if err := log.Add(op); (err != nil) != tc.wantErr {
						t.Errorf("Add() error = %v, wantErr %v", err, tc.wantErr)
					}
					if !tc.wantErr {
						if op.Id == 0 {
							t.Errorf("Add() did not set op ID")
						}
					}
				})
			}
		})
	}
}

func TestListOperation(t *testing.T) {
	// t.Parallel()

	// these should get assigned IDs 1-3 respectively by the oplog
	ops := []*v1.Operation{
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan1",
			RepoId:          "repo1",
			InstanceId:      "instance1",
			DisplayMessage:  "op1",
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan1",
			RepoId:          "repo2",
			InstanceId:      "instance2",
			DisplayMessage:  "op2",
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			UnixTimeStartMs: 1234,
			PlanId:          "plan2",
			RepoId:          "repo2",
			InstanceId:      "instance3",
			DisplayMessage:  "op3",
			FlowId:          943,
			Op:              &v1.Operation_OperationBackup{},
		},
	}

	tests := []struct {
		name     string
		query    oplog.Query
		expected []string
	}{
		{
			name:     "list plan1",
			query:    oplog.Query{PlanID: "plan1"},
			expected: []string{"op1", "op2"},
		},
		{
			name:     "list plan1 with limit",
			query:    oplog.Query{PlanID: "plan1", Limit: 1},
			expected: []string{"op1"},
		},
		{
			name:     "list plan1 with offset",
			query:    oplog.Query{PlanID: "plan1", Offset: 1},
			expected: []string{"op2"},
		},
		{
			name:     "list plan1 reversed",
			query:    oplog.Query{PlanID: "plan1", Reversed: true},
			expected: []string{"op2", "op1"},
		},
		{
			name:     "list plan2",
			query:    oplog.Query{PlanID: "plan2"},
			expected: []string{"op3"},
		},
		{
			name:     "list repo1",
			query:    oplog.Query{RepoID: "repo1"},
			expected: []string{"op1"},
		},
		{
			name:     "list repo2",
			query:    oplog.Query{RepoID: "repo2"},
			expected: []string{"op2", "op3"},
		},
		{
			name:  "list flow 943",
			query: oplog.Query{FlowID: 943},
			expected: []string{
				"op3",
			},
		},
	}

	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log := oplog.NewOpLog(store)
			for _, op := range ops {
				if err := log.Add(proto.Clone(op).(*v1.Operation)); err != nil {
					t.Fatalf("error adding operation: %s", err)
				}
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					var ops []*v1.Operation
					var err error
					collect := func(op *v1.Operation) error {
						ops = append(ops, op)
						return nil
					}
					err = log.Query(tc.query, collect)
					if err != nil {
						t.Fatalf("error listing operations: %s", err)
					}
					got := collectMessages(ops)
					if slices.Compare(got, tc.expected) != 0 {
						t.Errorf("want operations: %v, got unexpected operations: %v", tc.expected, got)
					}
				})
			}
		})
	}
}

func TestBigIO(t *testing.T) {
	t.Parallel()

	count := 10

	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log := oplog.NewOpLog(store)
			for i := 0; i < count; i++ {
				if err := log.Add(&v1.Operation{
					UnixTimeStartMs: 1234,
					PlanId:          "plan1",
					RepoId:          "repo1",
					InstanceId:      "instance1",
					Op:              &v1.Operation_OperationBackup{},
				}); err != nil {
					t.Fatalf("error adding operation: %s", err)
				}
			}

			countByPlanHelper(t, log, "plan1", count)
			countByRepoHelper(t, log, "repo1", count)
		})
	}
}

func TestIndexSnapshot(t *testing.T) {
	t.Parallel()

	op := &v1.Operation{
		UnixTimeStartMs: 1234,
		PlanId:          "plan1",
		RepoId:          "repo1",
		InstanceId:      "instance1",
		SnapshotId:      snapshotId,
		Op:              &v1.Operation_OperationIndexSnapshot{},
	}

	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log := oplog.NewOpLog(store)
			op := proto.Clone(op).(*v1.Operation)

			if err := log.Add(op); err != nil {
				t.Fatalf("error adding operation: %s", err)
			}

			var ops []*v1.Operation
			if err := log.Query(oplog.Query{SnapshotID: snapshotId}, func(op *v1.Operation) error {
				ops = append(ops, op)
				return nil
			}); err != nil {
				t.Fatalf("error listing operations: %s", err)
			}
			if len(ops) != 1 {
				t.Fatalf("want 1 operation, got %d", len(ops))
			}
			if ops[0].Id != op.Id {
				t.Errorf("want operation ID %d, got %d", op.Id, ops[0].Id)
			}
		})
	}
}

func TestUpdateOperation(t *testing.T) {
	t.Parallel()

	// Insert initial operation
	op := &v1.Operation{
		UnixTimeStartMs: 1234,
		PlanId:          "oldplan",
		RepoId:          "oldrepo",
		InstanceId:      "instance1",
		SnapshotId:      snapshotId,
	}

	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log := oplog.NewOpLog(store)
			op := proto.Clone(op).(*v1.Operation)

			if err := log.Add(op); err != nil {
				t.Fatalf("error adding operation: %s", err)
			}
			opId := op.Id

			// Validate initial values are indexed
			countByPlanHelper(t, log, "oldplan", 1)
			countByRepoHelper(t, log, "oldrepo", 1)
			countBySnapshotIdHelper(t, log, snapshotId, 1)

			// Update indexed values
			op.SnapshotId = snapshotId2
			op.PlanId = "myplan"
			op.RepoId = "myrepo"
			if err := log.Update(op); err != nil {
				t.Fatalf("error updating operation: %s", err)
			}

			// Validate updated values are indexed
			if opId != op.Id {
				t.Errorf("want operation ID %d, got %d", opId, op.Id)
			}

			countByPlanHelper(t, log, "myplan", 1)
			countByRepoHelper(t, log, "myrepo", 1)
			countBySnapshotIdHelper(t, log, snapshotId2, 1)

			// Validate prior values are gone
			countByPlanHelper(t, log, "oldplan", 0)
			countByRepoHelper(t, log, "oldrepo", 0)
			countBySnapshotIdHelper(t, log, snapshotId, 0)
		})
	}
}

func collectMessages(ops []*v1.Operation) []string {
	var messages []string
	for _, op := range ops {
		messages = append(messages, op.DisplayMessage)
	}
	return messages
}

func countByRepoHelper(t *testing.T, log *oplog.OpLog, repo string, expected int) {
	t.Helper()
	count := 0
	if err := log.Query(oplog.Query{RepoID: repo}, func(op *v1.Operation) error {
		count += 1
		return nil
	}); err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if count != expected {
		t.Errorf("want %d operations, got %d", expected, count)
	}
}

func countByPlanHelper(t *testing.T, log *oplog.OpLog, plan string, expected int) {
	t.Helper()
	count := 0
	if err := log.Query(oplog.Query{PlanID: plan}, func(op *v1.Operation) error {
		count += 1
		return nil
	}); err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if count != expected {
		t.Errorf("want %d operations, got %d", expected, count)
	}
}

func countBySnapshotIdHelper(t *testing.T, log *oplog.OpLog, snapshotId string, expected int) {
	t.Helper()
	count := 0
	if err := log.Query(oplog.Query{SnapshotID: snapshotId}, func(op *v1.Operation) error {
		count += 1
		return nil
	}); err != nil {
		t.Fatalf("error listing operations: %s", err)
	}
	if count != expected {
		t.Errorf("want %d operations, got %d", expected, count)
	}
}

func BenchmarkAdd(b *testing.B) {
	for name, store := range StoresForTest(b) {
		b.Run(name, func(b *testing.B) {
			log := oplog.NewOpLog(store)
			for i := 0; i < b.N; i++ {
				_ = log.Add(&v1.Operation{
					UnixTimeStartMs: 1234,
					PlanId:          "plan1",
					RepoId:          "repo1",
					InstanceId:      "instance1",
					Op:              &v1.Operation_OperationBackup{},
				})
			}
		})
	}
}

func BenchmarkList(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			for name, store := range StoresForTest(b) {
				log := oplog.NewOpLog(store)
				for i := 0; i < count; i++ {
					_ = log.Add(&v1.Operation{
						UnixTimeStartMs: 1234,
						PlanId:          "plan1",
						RepoId:          "repo1",
						InstanceId:      "instance1",
						Op:              &v1.Operation_OperationBackup{},
					})
				}

				b.Run(name, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						c := 0
						if err := log.Query(oplog.Query{PlanID: "plan1"}, func(op *v1.Operation) error {
							c += 1
							return nil
						}); err != nil {
							b.Fatalf("error listing operations: %s", err)
						}
						if c != count {
							b.Fatalf("want %d operations, got %d", count, c)
						}
					}
				})
			}
		})
	}
}
