package conformance

import (
	"fmt"
	"slices"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"github.com/garethgeorge/backrest/internal/oplog/sqlitestore"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	snapshotId  = "1234567890123456789012345678901234567890123456789012345678901234"
	snapshotId2 = "abcdefgh01234567890123456789012345678901234567890123456789012345"
)

func StoresForTest(t testing.TB) map[string]oplog.OpStore {
	sqlitestoreinst, err := sqlitestore.NewSqliteStore(t.TempDir() + "/test.sqlite")
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}
	t.Cleanup(func() { sqlitestoreinst.Close() })

	sqlitememstore, err := sqlitestore.NewMemorySqliteStore(t)
	if err != nil {
		t.Fatalf("error creating sqlite store: %s", err)
	}
	t.Cleanup(func() { sqlitememstore.Close() })

	return map[string]oplog.OpStore{
		"memory":    memstore.NewMemStore(),
		"sqlite":    sqlitestoreinst,
		"sqlitemem": sqlitememstore,
	}
}

func TestCreate(t *testing.T) {
	// t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			_, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}
		})
	}
}

func TestListAll(t *testing.T) {
	// t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			store, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}

			opsToAdd := []*v1.Operation{
				{
					UnixTimeStartMs: 1234,
					PlanId:          "plan1",
					RepoId:          "repo1",
					RepoGuid:        "repo1",
					InstanceId:      "instance1",
					Op:              &v1.Operation_OperationBackup{},
				},
				{
					UnixTimeStartMs: 4567,
					PlanId:          "plan2",
					RepoId:          "repo2",
					RepoGuid:        "repo2",
					InstanceId:      "instance2",
					Op:              &v1.Operation_OperationBackup{},
				},
			}

			for _, op := range opsToAdd {
				if err := store.Add(op); err != nil {
					t.Fatalf("error adding operation: %s", err)
				}
			}

			var ops []*v1.Operation
			if err := store.Query(oplog.Query{}, func(op *v1.Operation) error {
				ops = append(ops, op)
				return nil
			}); err != nil {
				t.Fatalf("error querying operations: %s", err)
			}

			if len(ops) != len(opsToAdd) {
				t.Errorf("expected %d operations, got %d", len(opsToAdd), len(ops))
			}

			for i := 0; i < len(ops); i++ {
				if diff := cmp.Diff(ops[i], opsToAdd[i], protocmp.Transform()); diff != "" {
					t.Fatalf("unexpected diff ops[%d] != opsToAdd[%d]: %v", i, i, diff)
				}
			}
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
				RepoGuid:        "testrepo",
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
				RepoGuid:        "testrepo",
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
				RepoGuid:        "testrepo",
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
				RepoGuid:        "testrepo",
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
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					log, err := oplog.NewOpLog(store)
					if err != nil {
						t.Fatalf("error creating oplog: %v", err)
					}
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
			InstanceId:      "foo",
			PlanId:          "plan1",
			RepoId:          "repo1",
			RepoGuid:        "repo1",
			UnixTimeStartMs: 1234,
			DisplayMessage:  "op1",
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			InstanceId:      "bar",
			PlanId:          "plan1",
			RepoId:          "repo2",
			RepoGuid:        "repo2",
			UnixTimeStartMs: 1234,
			DisplayMessage:  "op2",
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			InstanceId:      "baz",
			PlanId:          "plan2",
			RepoId:          "repo2",
			RepoGuid:        "repo2",
			UnixTimeStartMs: 1234,
			DisplayMessage:  "op3",
			FlowId:          943,
			Op:              &v1.Operation_OperationBackup{},
		},
		{
			InstanceId:      "foo",
			PlanId:          "foo-plan",
			RepoId:          "foo-repo",
			RepoGuid:        "foo-repo-guid",
			UnixTimeStartMs: 1234,
			DisplayMessage:  "foo-op",
			Op:              &v1.Operation_OperationBackup{},
			OriginalId:      4567,
			OriginalFlowId:  789,
		},
	}

	tests := []struct {
		name     string
		query    oplog.Query
		expected []string
	}{
		{
			name:     "list plan1",
			query:    oplog.Query{}.SetPlanID("plan1"),
			expected: []string{"op1", "op2"},
		},
		{
			name:     "list plan1 with limit",
			query:    oplog.Query{}.SetPlanID("plan1").SetLimit(1),
			expected: []string{"op1"},
		},
		{
			name:     "list plan1 with offset",
			query:    oplog.Query{}.SetPlanID("plan1").SetOffset(1),
			expected: []string{"op2"},
		},
		{
			name:     "list plan1 reversed",
			query:    oplog.Query{}.SetPlanID("plan1").SetReversed(true),
			expected: []string{"op2", "op1"},
		},
		{
			name:     "list plan2",
			query:    oplog.Query{}.SetPlanID("plan2"),
			expected: []string{"op3"},
		},
		{
			name:     "list repo1",
			query:    oplog.Query{}.SetRepoGUID("repo1"),
			expected: []string{"op1"},
		},
		{
			name:     "list repo2",
			query:    oplog.Query{}.SetRepoGUID("repo2"),
			expected: []string{"op2", "op3"},
		},
		{
			name:  "list flow 943",
			query: oplog.Query{}.SetFlowID(943),
			expected: []string{
				"op3",
			},
		},
		{
			name:  "list original ID",
			query: oplog.Query{}.SetOriginalID(4567),
			expected: []string{
				"foo-op",
			},
		},
		{
			name:  "list original flow ID",
			query: oplog.Query{}.SetOriginalFlowID(789),
			expected: []string{
				"foo-op",
			},
		},
		{
			name: "a very compound query",
			query: oplog.Query{}.
				SetPlanID("foo-plan").
				SetRepoGUID("foo-repo-guid").
				SetInstanceID("foo").
				SetOriginalID(4567).
				SetOriginalFlowID(789),
			expected: []string{
				"foo-op",
			},
		},
		{
			name:     "list modno gte",
			query:    oplog.Query{}.SetModnoGte(3),
			expected: []string{"op3", "foo-op"},
		},
	}

	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}
			for _, op := range ops {
				if err := log.Add(proto.Clone(op).(*v1.Operation)); err != nil {
					t.Fatalf("error adding operation: %s", err)
				}
			}

			for _, tc := range tests {
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
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
		store := store
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}
			for i := 0; i < count; i++ {
				if err := log.Add(&v1.Operation{
					UnixTimeStartMs: 1234,
					PlanId:          "plan1",
					RepoId:          "repo1",
					RepoGuid:        "repo1",
					InstanceId:      "instance1",
					Op:              &v1.Operation_OperationBackup{},
				}); err != nil {
					t.Fatalf("error adding operation: %s", err)
				}
			}

			countByPlanHelper(t, log, "plan1", count)
			countByRepoGUIDHelper(t, log, "repo1", count)
		})
	}
}

func TestIndexSnapshot(t *testing.T) {
	t.Parallel()

	op := &v1.Operation{
		UnixTimeStartMs: 1234,
		PlanId:          "plan1",
		RepoId:          "repo1",
		RepoGuid:        "repo1",
		InstanceId:      "instance1",
		SnapshotId:      snapshotId,
		Op:              &v1.Operation_OperationIndexSnapshot{},
	}

	for name, store := range StoresForTest(t) {
		store := store
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}
			op := proto.Clone(op).(*v1.Operation)

			if err := log.Add(op); err != nil {
				t.Fatalf("error adding operation: %s", err)
			}

			var ops []*v1.Operation
			if err := log.Query(oplog.Query{}.SetSnapshotID(snapshotId), func(op *v1.Operation) error {
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
		RepoGuid:        "oldrepo",
		InstanceId:      "instance1",
		SnapshotId:      snapshotId,
	}

	for name, store := range StoresForTest(t) {
		store := store
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}
			op := proto.Clone(op).(*v1.Operation)

			if err := log.Add(op); err != nil {
				t.Fatalf("error adding operation: %s", err)
			}
			opId := op.Id

			// Validate initial values are indexed
			countByPlanHelper(t, log, "oldplan", 1)
			countByRepoGUIDHelper(t, log, "oldrepo", 1)
			countBySnapshotIdHelper(t, log, snapshotId, 1)

			// Update indexed values
			op.SnapshotId = snapshotId2
			op.PlanId = "myplan"
			op.RepoId = "myrepo"
			op.RepoGuid = "myrepo"
			if err := log.Update(op); err != nil {
				t.Fatalf("error updating operation: %s", err)
			}

			// Validate updated values are indexed
			if opId != op.Id {
				t.Errorf("want operation ID %d, got %d", opId, op.Id)
			}

			countByPlanHelper(t, log, "myplan", 1)
			countByRepoGUIDHelper(t, log, "myrepo", 1)
			countBySnapshotIdHelper(t, log, snapshotId2, 1)

			// Validate prior values are gone
			countByPlanHelper(t, log, "oldplan", 0)
			countByRepoGUIDHelper(t, log, "oldrepo", 0)
			countBySnapshotIdHelper(t, log, snapshotId, 0)
		})
	}
}

func TestTransform(t *testing.T) {
	ops := []*v1.Operation{
		{
			InstanceId:      "foo",
			PlanId:          "plan1",
			RepoId:          "repo1",
			RepoGuid:        "repo1",
			UnixTimeStartMs: 1234,
			UnixTimeEndMs:   5678,
		},
		{
			InstanceId:      "bar",
			PlanId:          "plan1",
			RepoId:          "repo1",
			RepoGuid:        "repo1",
			UnixTimeStartMs: 1234,
			UnixTimeEndMs:   5678,
		},
	}

	tcs := []struct {
		name  string
		f     func(*v1.Operation) (*v1.Operation, error)
		ops   []*v1.Operation
		want  []*v1.Operation
		query oplog.Query
	}{
		{
			name: "no change",
			f: func(op *v1.Operation) (*v1.Operation, error) {
				return nil, nil
			},
			ops:  ops,
			want: ops,
		},
		{
			name: "modno incremented by copy",
			f: func(op *v1.Operation) (*v1.Operation, error) {
				return proto.Clone(op).(*v1.Operation), nil
			},
			ops:  ops,
			want: ops,
		},
		{
			name: "change plan",
			f: func(op *v1.Operation) (*v1.Operation, error) {
				op.PlanId = "newplan"
				return op, nil
			},
			ops: []*v1.Operation{
				{
					InstanceId:      "foo",
					PlanId:          "oldplan",
					RepoId:          "repo1",
					RepoGuid:        "repo1",
					UnixTimeStartMs: 1234,
					UnixTimeEndMs:   5678,
				},
			},
			want: []*v1.Operation{
				{
					InstanceId:      "foo",
					PlanId:          "newplan",
					RepoId:          "repo1",
					RepoGuid:        "repo1",
					UnixTimeStartMs: 1234,
					UnixTimeEndMs:   5678,
				},
			},
		},
		{
			name: "change plan with query",
			f: func(op *v1.Operation) (*v1.Operation, error) {
				op.PlanId = "newplan"
				return op, nil
			},
			ops: ops,
			want: []*v1.Operation{
				{
					InstanceId:      "foo",
					PlanId:          "newplan",
					RepoId:          "repo1",
					RepoGuid:        "repo1",
					UnixTimeStartMs: 1234,
					UnixTimeEndMs:   5678,
				},
				ops[1],
			},
			query: oplog.Query{}.SetInstanceID("foo"),
		},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for name, store := range StoresForTest(t) {
				store := store
				t.Run(name, func(t *testing.T) {
					log, err := oplog.NewOpLog(store)
					if err != nil {
						t.Fatalf("error creating oplog: %v", err)
					}
					for _, op := range tc.ops {
						copy := proto.Clone(op).(*v1.Operation)
						if err := log.Add(copy); err != nil {
							t.Fatalf("error adding operation: %s", err)
						}
					}

					if err := log.Transform(tc.query, tc.f); err != nil {
						t.Fatalf("error transforming operations: %s", err)
					}

					var got []*v1.Operation
					if err := log.Query(oplog.Query{}, func(op *v1.Operation) error {
						op.Id = 0
						op.FlowId = 0
						got = append(got, op)
						return nil
					}); err != nil {
						t.Fatalf("error listing operations: %s", err)
					}

					for _, op := range got {
						op.Modno = 0
					}

					if diff := cmp.Diff(got, tc.want, protocmp.Transform()); diff != "" {
						t.Errorf("unexpected diff: %v", diff)
					}
				})
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}

			op := &v1.Operation{
				UnixTimeStartMs: 1234,
				PlanId:          "plan1",
				RepoId:          "repo1",
				RepoGuid:        "repo1",
				InstanceId:      "instance1",
				Op:              &v1.Operation_OperationBackup{},
			}

			if err := log.Add(op); err != nil {
				t.Fatalf("error adding operation: %s", err)
			}

			if err := log.Delete(op.Id); err != nil {
				t.Fatalf("error deleting operation: %s", err)
			}

			var ops []*v1.Operation
			if err := log.Query(oplog.Query{}, func(op *v1.Operation) error {
				ops = append(ops, op)
				return nil
			}); err != nil {
				t.Fatalf("error querying operations: %s", err)
			}

			if len(ops) != 0 {
				t.Errorf("expected 0 operations after deletion, got %d", len(ops))
			}
		})
	}
}

func TestBulkDelete(t *testing.T) {
	t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}

			// Add 2000 operations
			var ops []*v1.Operation
			for i := 0; i < 2000; i++ {
				op := &v1.Operation{
					UnixTimeStartMs: 1234,
					PlanId:          fmt.Sprintf("plan%d", i),
					RepoId:          fmt.Sprintf("repo%d", i),
					RepoGuid:        fmt.Sprintf("repo%d", i),
					InstanceId:      fmt.Sprintf("instance%d", i),
					Op:              &v1.Operation_OperationBackup{},
				}
				ops = append(ops, op)
			}

			var ids []int64
			if err := log.Add(ops...); err != nil {
				t.Fatalf("error adding operations: %s", err)
			}
			for _, op := range ops {
				ids = append(ids, op.Id)
			}

			// Delete all operations
			err = log.Delete(ids...)
			if err != nil {
				t.Fatalf("error deleting operations: %s", err)
			}
			if len(ids) != 2000 {
				t.Errorf("expected 2000 deleted operations, got %d", len(ids))
			}

			// Verify deletion
			var count int
			if err := log.Query(oplog.Query{}, func(op *v1.Operation) error {
				count++
				return nil
			}); err != nil {
				t.Fatalf("error querying operations: %s", err)
			}
			if count != 0 {
				t.Errorf("expected 0 operations after deletion, got %d", count)
			}
		})
	}
}

func TestQueryMetadata(t *testing.T) {
	t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}
			if err := log.Add(&v1.Operation{
				UnixTimeStartMs: 1234,
				PlanId:          "plan1",
				RepoId:          "repo1",
				RepoGuid:        "repo1-guid",
				InstanceId:      "instance1",
				Op:              &v1.Operation_OperationBackup{},
				FlowId:          5,
				OriginalId:      3,
				OriginalFlowId:  4,
				Status:          v1.OperationStatus_STATUS_INPROGRESS,
			}); err != nil {
				t.Fatalf("error adding operation: %s", err)
			}

			var metadata []oplog.OpMetadata
			if err := log.QueryMetadata(oplog.Query{}.SetPlanID("plan1"), func(op oplog.OpMetadata) error {
				metadata = append(metadata, op)
				return nil
			}); err != nil {
				t.Fatalf("error listing metadata: %s", err)
			}
			if len(metadata) != 1 {
				t.Fatalf("want 1 metadata, got %d", len(metadata))
			}

			if metadata[0].Modno == 0 {
				t.Errorf("modno should not be 0")
			}
			metadata[0].Modno = 0 // ignore for diff since it's random

			if diff := cmp.Diff(metadata[0], oplog.OpMetadata{
				ID:             metadata[0].ID,
				FlowID:         5,
				OriginalID:     3,
				OriginalFlowID: 4,
				Status:         v1.OperationStatus_STATUS_INPROGRESS,
			}); diff != "" {
				t.Errorf("unexpected diff: %v", diff)
			}
		})
	}
}

func TestGetHighestOpIDAndModno(t *testing.T) {
	t.Parallel()
	for name, store := range StoresForTest(t) {
		t.Run(name, func(t *testing.T) {
			log, err := oplog.NewOpLog(store)
			if err != nil {
				t.Fatalf("error creating oplog: %v", err)
			}

			t.Run("empty store", func(t *testing.T) {
				highestID, highestModno, err := store.GetHighestOpIDAndModno(oplog.Query{})
				if err != nil {
					t.Fatalf("error getting highest ID and modno: %v", err)
				}
				if highestID != 0 {
					t.Errorf("expected highest ID 0, got %d", highestID)
				}
				if highestModno != 0 {
					t.Errorf("expected highest modno 0, got %d", highestModno)
				}
			})

			// Add operations with different plans and repos
			ops := []*v1.Operation{
				{
					UnixTimeStartMs: 1000,
					PlanId:          "plan1",
					RepoId:          "repo1",
					RepoGuid:        "repo1-guid",
					InstanceId:      "instance1",
					Op:              &v1.Operation_OperationBackup{},
				},
				{
					UnixTimeStartMs: 2000,
					PlanId:          "plan1",
					RepoId:          "repo1",
					RepoGuid:        "repo1-guid",
					InstanceId:      "instance1",
					Op:              &v1.Operation_OperationBackup{},
				},
				{
					UnixTimeStartMs: 3000,
					PlanId:          "plan2",
					RepoId:          "repo2",
					RepoGuid:        "repo2-guid",
					InstanceId:      "instance2",
					Op:              &v1.Operation_OperationBackup{},
				},
			}

			for _, op := range ops {
				if err := log.Add(op); err != nil {
					t.Fatalf("error adding operation: %v", err)
				}
			}

			t.Run("all operations", func(t *testing.T) {
				highestID, highestModno, err := store.GetHighestOpIDAndModno(oplog.Query{})
				if err != nil {
					t.Fatalf("error getting highest ID and modno: %v", err)
				}
				if highestID != ops[2].Id {
					t.Errorf("expected highest ID %d, got %d", ops[2].Id, highestID)
				}
				if highestModno != ops[2].Modno {
					t.Errorf("expected highest modno %d, got %d", ops[2].Modno, highestModno)
				}
			})

			t.Run("filtered by plan", func(t *testing.T) {
				highestID, highestModno, err := store.GetHighestOpIDAndModno(oplog.Query{}.SetPlanID("plan1"))
				if err != nil {
					t.Fatalf("error getting highest ID and modno: %v", err)
				}
				if highestID != ops[1].Id {
					t.Errorf("expected highest ID %d, got %d", ops[1].Id, highestID)
				}
				if highestModno != ops[1].Modno {
					t.Errorf("expected highest modno %d, got %d", ops[1].Modno, highestModno)
				}
			})

			t.Run("filtered by repo", func(t *testing.T) {
				highestID, highestModno, err := store.GetHighestOpIDAndModno(oplog.Query{}.SetRepoGUID("repo2-guid"))
				if err != nil {
					t.Fatalf("error getting highest ID and modno: %v", err)
				}
				if highestID != ops[2].Id {
					t.Errorf("expected highest ID %d, got %d", ops[2].Id, highestID)
				}
				if highestModno != ops[2].Modno {
					t.Errorf("expected highest modno %d, got %d", ops[2].Modno, highestModno)
				}
			})

			t.Run("no matching operations", func(t *testing.T) {
				highestID, highestModno, err := store.GetHighestOpIDAndModno(oplog.Query{}.SetPlanID("nonexistent"))
				if err != nil {
					t.Fatalf("error getting highest ID and modno: %v", err)
				}
				if highestID != 0 {
					t.Errorf("expected highest ID 0, got %d", highestID)
				}
				if highestModno != 0 {
					t.Errorf("expected highest modno 0, got %d", highestModno)
				}
			})
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

func countByRepoGUIDHelper(t *testing.T, log *oplog.OpLog, repoGUID string, expected int) {
	t.Helper()
	count := 0
	if err := log.Query(oplog.Query{}.SetRepoGUID(repoGUID), func(op *v1.Operation) error {
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
	if err := log.Query(oplog.Query{}.SetPlanID(plan), func(op *v1.Operation) error {
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
	if err := log.Query(oplog.Query{}.SetSnapshotID(snapshotId), func(op *v1.Operation) error {
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
			log, err := oplog.NewOpLog(store)
			if err != nil {
				b.Fatalf("error creating oplog: %v", err)
			}
			for i := 0; i < b.N; i++ {
				_ = log.Add(&v1.Operation{
					UnixTimeStartMs: 1234,
					PlanId:          "plan1",
					RepoId:          "repo1",
					RepoGuid:        "repo1",
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
				log, err := oplog.NewOpLog(store)
				if err != nil {
					b.Fatalf("error creating oplog: %v", err)
				}
				for i := 0; i < count; i++ {
					_ = log.Add(&v1.Operation{
						UnixTimeStartMs: 1234,
						PlanId:          "plan1",
						RepoId:          "repo1",
						RepoGuid:        "repo1",
						InstanceId:      "instance1",
						Op:              &v1.Operation_OperationBackup{},
					})
				}

				b.Run(name, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						c := 0
						if err := log.Query(oplog.Query{}.SetPlanID("plan1"), func(op *v1.Operation) error {
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

func BenchmarkGetLastItem(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			for name, store := range StoresForTest(b) {
				log, err := oplog.NewOpLog(store)
				if err != nil {
					b.Fatalf("error creating oplog: %v", err)
				}
				for i := 0; i < count; i++ {
					_ = log.Add(&v1.Operation{
						UnixTimeStartMs: 1234,
						PlanId:          "plan1",
						RepoId:          "repo1",
						RepoGuid:        "repo1",
						InstanceId:      "instance1",
						Op:              &v1.Operation_OperationBackup{},
					})
				}

				b.Run(name, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						c := 0
						if err := log.Query(oplog.Query{}.SetPlanID("plan1").SetReversed(true), func(op *v1.Operation) error {
							c += 1
							return oplog.ErrStopIteration
						}); err != nil {
							b.Fatalf("error listing operations: %s", err)
						}
						if c != 1 {
							b.Fatalf("want 1 operation, got %d", c)
						}
					}
				})
			}
		})
	}
}
