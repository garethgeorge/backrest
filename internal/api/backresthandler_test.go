package api

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

func TestUpdateConfig(t *testing.T) {
	t.Parallel()

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno: 1234,
		},
	})

	tests := []struct {
		name    string
		req     *v1.Config
		wantErr bool
		res     *v1.Config
	}{
		{
			name: "bad modno",
			req: &v1.Config{
				Modno: 4321,
			},
			wantErr: true,
		},
		{
			name: "good modno",
			req: &v1.Config{
				Modno:    1234,
				Instance: "test",
			},
			wantErr: false,
			res: &v1.Config{
				Modno:    1235,
				Instance: "test",
			},
		},
		{
			name: "reject when validation fails",
			req: &v1.Config{
				Modno:    1235,
				Instance: "test",
				Repos: []*v1.Repo{
					{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := sut.handler.SetConfig(context.Background(), connect.NewRequest(tt.req))
			if (err != nil) != tt.wantErr {
				t.Errorf("SetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && !proto.Equal(res.Msg, tt.res) {
				t.Errorf("SetConfig() got = %v, want %v", res, tt.res)
			}
		})
	}
}

func TestBackup(t *testing.T) {
	t.Parallel()

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno:    1234,
			Instance: "test",
			Repos: []*v1.Repo{
				{
					Id:       "local",
					Uri:      t.TempDir(),
					Password: "test",
					Flags:    []string{"--no-cache"},
				},
			},
			Plans: []*v1.Plan{
				{
					Id:   "test",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Retention: &v1.RetentionPolicy{
						Policy: &v1.RetentionPolicy_PolicyKeepAll{PolicyKeepAll: true},
					},
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sut.orch.Run(ctx)
	}()

	_, err := sut.handler.Backup(context.Background(), connect.NewRequest(&types.StringValue{Value: "test"}))
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	// Check that there is a successful backup recorded in the log.
	if slices.IndexFunc(getOperations(t, sut.oplog), func(op *v1.Operation) bool {
		_, ok := op.GetOp().(*v1.Operation_OperationBackup)
		return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok
	}) == -1 {
		t.Fatalf("Expected a backup operation")
	}

	// Wait for the index snapshot operation to appear in the oplog.
	var snapshotOp *v1.Operation
	if err := retry(t, 10, 1*time.Second, func() error {
		operations := getOperations(t, sut.oplog)
		if index := slices.IndexFunc(operations, func(op *v1.Operation) bool {
			_, ok := op.GetOp().(*v1.Operation_OperationIndexSnapshot)
			return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok
		}); index != -1 {
			snapshotOp = operations[index]
			return nil
		}
		return errors.New("snapshot not indexed")
	}); err != nil {
		t.Fatalf("Couldn't find snapshot in oplog")
	}

	if snapshotOp.SnapshotId == "" {
		t.Fatalf("snapshotId must be set")
	}

	// Wait for a forget operation to appear in the oplog.
	if err := retry(t, 10, 1*time.Second, func() error {
		operations := getOperations(t, sut.oplog)
		if index := slices.IndexFunc(operations, func(op *v1.Operation) bool {
			_, ok := op.GetOp().(*v1.Operation_OperationForget)
			return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok
		}); index != -1 {
			op := operations[index]
			if op.FlowId != snapshotOp.FlowId {
				t.Fatalf("Flow ID mismatch on forget operation")
			}
			return nil
		}
		return errors.New("forget not indexed")
	}); err != nil {
		t.Fatalf("Couldn't find forget in oplog")
	}
}

func TestMultipleBackup(t *testing.T) {
	t.Parallel()

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno:    1234,
			Instance: "test",
			Repos: []*v1.Repo{
				{
					Id:       "local",
					Uri:      t.TempDir(),
					Password: "test",
					Flags:    []string{"--no-cache"},
				},
			},
			Plans: []*v1.Plan{
				{
					Id:   "test",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Retention: &v1.RetentionPolicy{
						Policy: &v1.RetentionPolicy_PolicyKeepLastN{
							PolicyKeepLastN: 1,
						},
					},
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sut.orch.Run(ctx)
	}()

	for i := 0; i < 3; i++ {
		_, err := sut.handler.Backup(context.Background(), connect.NewRequest(&types.StringValue{Value: "test"}))
		if err != nil {
			t.Fatalf("Backup() error = %v", err)
		}
	}

	// Wait for a forget that removed 1 snapshot to appear in the oplog
	if err := retry(t, 10, 1*time.Second, func() error {
		operations := getOperations(t, sut.oplog)
		if index := slices.IndexFunc(operations, func(op *v1.Operation) bool {
			forget, ok := op.GetOp().(*v1.Operation_OperationForget)
			return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok && len(forget.OperationForget.Forget) > 0
		}); index != -1 {
			return nil
		}
		return errors.New("forget not indexed")
	}); err != nil {
		t.Fatalf("Couldn't find forget with 1 item removed in the operation log")
	}
}

func TestHookExecution(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}

	dir := t.TempDir()

	hookOutputBefore := path.Join(dir, "before.txt")
	hookOutputAfter := path.Join(dir, "after.txt")

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno:    1234,
			Instance: "test",
			Repos: []*v1.Repo{
				{
					Id:       "local",
					Uri:      t.TempDir(),
					Password: "test",
					Flags:    []string{"--no-cache"},
				},
			},
			Plans: []*v1.Plan{
				{
					Id:   "test",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Hooks: []*v1.Hook{
						{
							Conditions: []v1.Hook_Condition{
								v1.Hook_CONDITION_SNAPSHOT_START,
							},
							Action: &v1.Hook_ActionCommand{
								ActionCommand: &v1.Hook_Command{
									Command: "echo before > " + hookOutputBefore,
								},
							},
						},
						{
							Conditions: []v1.Hook_Condition{
								v1.Hook_CONDITION_SNAPSHOT_END,
							},
							Action: &v1.Hook_ActionCommand{
								ActionCommand: &v1.Hook_Command{
									Command: "echo after > " + hookOutputAfter,
								},
							},
						},
					},
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sut.orch.Run(ctx)
	}()

	_, err := sut.handler.Backup(context.Background(), connect.NewRequest(&types.StringValue{Value: "test"}))
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	// Wait for two hook operations to appear in the oplog
	if err := retry(t, 10, 1*time.Second, func() error {
		hookOps := slices.DeleteFunc(getOperations(t, sut.oplog), func(op *v1.Operation) bool {
			_, ok := op.GetOp().(*v1.Operation_OperationRunHook)
			return !ok
		})

		if len(hookOps) == 2 {
			return nil
		}
		return fmt.Errorf("expected 2 hook operations, got %d", len(hookOps))
	}); err != nil {
		t.Fatalf("Couldn't find hooks in oplog: %v", err)
	}

	// expect the hook output files to exist
	if _, err := os.Stat(hookOutputBefore); err != nil {
		t.Fatalf("hook output file before not found")
	}

	if _, err := os.Stat(hookOutputAfter); err != nil {
		t.Fatalf("hook output file after not found")
	}
}

func TestHookOnErrorHandling(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno:    1234,
			Instance: "test",
			Repos: []*v1.Repo{
				{
					Id:       "local",
					Uri:      t.TempDir(),
					Password: "test",
					Flags:    []string{"--no-cache"},
				},
			},
			Plans: []*v1.Plan{
				{
					Id:   "test-cancel",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Hooks: []*v1.Hook{
						{
							Conditions: []v1.Hook_Condition{
								v1.Hook_CONDITION_SNAPSHOT_START,
							},
							Action: &v1.Hook_ActionCommand{
								ActionCommand: &v1.Hook_Command{
									Command: "exit 123",
								},
							},
							OnError: v1.Hook_ON_ERROR_CANCEL,
						},
					},
				},
				{
					Id:   "test-error",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Hooks: []*v1.Hook{
						{
							Conditions: []v1.Hook_Condition{
								v1.Hook_CONDITION_SNAPSHOT_START,
							},
							Action: &v1.Hook_ActionCommand{
								ActionCommand: &v1.Hook_Command{
									Command: "exit 123",
								},
							},
							OnError: v1.Hook_ON_ERROR_FATAL,
						},
					},
				},
				{
					Id:   "test-ignore",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Hooks: []*v1.Hook{
						{
							Conditions: []v1.Hook_Condition{
								v1.Hook_CONDITION_SNAPSHOT_START,
							},
							Action: &v1.Hook_ActionCommand{
								ActionCommand: &v1.Hook_Command{
									Command: "exit 123",
								},
							},
							OnError: v1.Hook_ON_ERROR_IGNORE,
						},
					},
				},
				{
					Id:   "test-retry",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Hooks: []*v1.Hook{
						{
							Conditions: []v1.Hook_Condition{
								v1.Hook_CONDITION_SNAPSHOT_START,
							},
							Action: &v1.Hook_ActionCommand{
								ActionCommand: &v1.Hook_Command{
									Command: "exit 123",
								},
							},
							OnError: v1.Hook_ON_ERROR_RETRY_10MINUTES,
						},
					},
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sut.orch.Run(ctx)
	}()

	tests := []struct {
		name             string
		plan             string
		wantHookStatus   v1.OperationStatus
		wantBackupStatus v1.OperationStatus
		wantBackupError  bool
		noWaitForBackup  bool
	}{
		{
			name:             "cancel",
			plan:             "test-cancel",
			wantHookStatus:   v1.OperationStatus_STATUS_ERROR,
			wantBackupStatus: v1.OperationStatus_STATUS_USER_CANCELLED,
			wantBackupError:  true,
		},
		{
			name:             "error",
			plan:             "test-error",
			wantHookStatus:   v1.OperationStatus_STATUS_ERROR,
			wantBackupStatus: v1.OperationStatus_STATUS_ERROR,
			wantBackupError:  true,
		},
		{
			name:             "ignore",
			plan:             "test-ignore",
			wantHookStatus:   v1.OperationStatus_STATUS_ERROR,
			wantBackupStatus: v1.OperationStatus_STATUS_SUCCESS,
			wantBackupError:  false,
		},
		{
			name:             "retry",
			plan:             "test-retry",
			wantHookStatus:   v1.OperationStatus_STATUS_ERROR,
			wantBackupStatus: v1.OperationStatus_STATUS_PENDING,
			wantBackupError:  false,
			noWaitForBackup:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sut.opstore.ResetForTest(t)

			var errgroup errgroup.Group

			errgroup.Go(func() error {
				_, err := sut.handler.Backup(context.Background(), connect.NewRequest(&types.StringValue{Value: tc.plan}))
				if (err != nil) != tc.wantBackupError {
					return fmt.Errorf("Backup() error = %v, wantErr %v", err, tc.wantBackupError)
				}
				return nil
			})

			if !tc.noWaitForBackup {
				if err := errgroup.Wait(); err != nil {
					t.Fatalf(err.Error())
				}
			}

			// Wait for hook operation to be attempted in the oplog
			if err := retry(t, 10, 1*time.Second, func() error {
				hookOps := slices.DeleteFunc(getOperations(t, sut.oplog), func(op *v1.Operation) bool {
					_, ok := op.GetOp().(*v1.Operation_OperationRunHook)
					return !ok
				})
				if len(hookOps) != 1 {
					return fmt.Errorf("expected 1 hook operations, got %d", len(hookOps))
				}
				if hookOps[0].Status != tc.wantHookStatus {
					return fmt.Errorf("expected hook operation error status, got %v", hookOps[0].Status)
				}
				return nil
			}); err != nil {
				t.Fatalf("Couldn't find hook operation in oplog: %v", err)
			}

			backupOps := slices.DeleteFunc(getOperations(t, sut.oplog), func(op *v1.Operation) bool {
				_, ok := op.GetOp().(*v1.Operation_OperationBackup)
				return !ok
			})
			if len(backupOps) != 1 {
				t.Errorf("expected 1 backup operation, got %d", len(backupOps))
			}
			if backupOps[0].Status != tc.wantBackupStatus {
				t.Errorf("expected backup operation cancelled status, got %v", backupOps[0].Status)
			}
		})
	}
}

func TestCancelBackup(t *testing.T) {
	t.Parallel()

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno:    1234,
			Instance: "test",
			Repos: []*v1.Repo{
				{
					Id:       "local",
					Uri:      t.TempDir(),
					Password: "test",
					Flags:    []string{"--no-cache"},
				},
			},
			Plans: []*v1.Plan{
				{
					Id:   "test",
					Repo: "local",
					Paths: []string{
						t.TempDir(),
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Retention: &v1.RetentionPolicy{
						Policy: &v1.RetentionPolicy_PolicyKeepLastN{
							PolicyKeepLastN: 1,
						},
					},
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sut.orch.Run(ctx)
	}()

	// Start a backup
	var errgroup errgroup.Group
	errgroup.Go(func() error {
		backupReq := connect.NewRequest(&types.StringValue{Value: "test"})
		sut.handler.Backup(context.Background(), backupReq)
		return nil
	})

	// Find the backup operation ID in the oplog
	var backupOpId int64
	if err := retry(t, 100, 10*time.Millisecond, func() error {
		operations := getOperations(t, sut.oplog)
		for _, op := range operations {
			_, ok := op.GetOp().(*v1.Operation_OperationBackup)
			if ok {
				backupOpId = op.Id
				return nil
			}
		}
		return errors.New("backup operation not found")
	}); err != nil {
		t.Fatalf("Couldn't find backup operation in oplog")
	}

	errgroup.Go(func() error {
		if _, err := sut.handler.Cancel(context.Background(), connect.NewRequest(&types.Int64Value{Value: backupOpId})); err != nil {
			return fmt.Errorf("Cancel() error = %v, wantErr nil", err)
		}
		return nil
	})

	if err := errgroup.Wait(); err != nil {
		t.Fatalf(err.Error())
	}

	// Assert that the backup operation was cancelled
	if slices.IndexFunc(getOperations(t, sut.oplog), func(op *v1.Operation) bool {
		_, ok := op.GetOp().(*v1.Operation_OperationBackup)
		return op.Status == v1.OperationStatus_STATUS_ERROR && ok
	}) == -1 {
		t.Fatalf("Expected a failed backup operation in the log")
	}
}

func TestRestore(t *testing.T) {
	t.Parallel()

	backupDataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(backupDataDir, "findme.txt"), []byte("test data"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sut := createSystemUnderTest(t, &config.MemoryStore{
		Config: &v1.Config{
			Modno:    1234,
			Instance: "test",
			Repos: []*v1.Repo{
				{
					Id:       "local",
					Uri:      t.TempDir(),
					Password: "test",
					Flags:    []string{"--no-cache"},
				},
			},
			Plans: []*v1.Plan{
				{
					Id:   "test",
					Repo: "local",
					Paths: []string{
						backupDataDir,
					},
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Disabled{Disabled: true},
					},
					Retention: &v1.RetentionPolicy{
						Policy: &v1.RetentionPolicy_PolicyKeepAll{PolicyKeepAll: true},
					},
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sut.orch.Run(ctx)
	}()

	_, err := sut.handler.Backup(context.Background(), connect.NewRequest(&types.StringValue{Value: "test"}))
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	// Check that there is a successful backup recorded in the log.
	if slices.IndexFunc(getOperations(t, sut.oplog), func(op *v1.Operation) bool {
		_, ok := op.GetOp().(*v1.Operation_OperationBackup)
		return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok
	}) == -1 {
		t.Fatalf("Expected a backup operation")
	}

	// Wait for the index snapshot operation to appear in the oplog.
	var snapshotOp *v1.Operation
	if err := retry(t, 10, 2*time.Second, func() error {
		operations := getOperations(t, sut.oplog)
		if index := slices.IndexFunc(operations, func(op *v1.Operation) bool {
			_, ok := op.GetOp().(*v1.Operation_OperationIndexSnapshot)
			return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok
		}); index != -1 {
			snapshotOp = operations[index]
			return nil
		}
		return errors.New("snapshot not indexed")
	}); err != nil {
		t.Fatalf("Couldn't find snapshot in oplog")
	}

	if snapshotOp.SnapshotId == "" {
		t.Fatalf("snapshotId must be set")
	}

	restoreTarget := t.TempDir() + "/restore"

	_, err = sut.handler.Restore(context.Background(), connect.NewRequest(&v1.RestoreSnapshotRequest{
		SnapshotId: snapshotOp.SnapshotId,
		PlanId:     "test",
		RepoId:     "local",
		Target:     restoreTarget,
	}))
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	// Wait for a restore operation to appear in the oplog.
	if err := retry(t, 10, 2*time.Second, func() error {
		operations := getOperations(t, sut.oplog)
		if index := slices.IndexFunc(operations, func(op *v1.Operation) bool {
			_, ok := op.GetOp().(*v1.Operation_OperationRestore)
			return op.Status == v1.OperationStatus_STATUS_SUCCESS && ok
		}); index != -1 {
			op := operations[index]
			if op.SnapshotId != snapshotOp.SnapshotId {
				t.Errorf("Snapshot ID mismatch on restore operation")
			}
			return nil
		}
		return errors.New("restore not indexed")
	}); err != nil {
		t.Fatalf("Couldn't find restore in oplog")
	}

	// Check that the restore target contains the expected file.
	var files []string
	filepath.Walk(restoreTarget, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			t.Errorf("Walk() error = %v", err)
		}
		files = append(files, path)
		return nil
	})
	t.Logf("files: %v", files)
	if !slices.ContainsFunc(files, func(s string) bool {
		return strings.HasSuffix(s, "findme.txt")
	}) {
		t.Fatalf("Expected file not found in restore target")
	}
}

type systemUnderTest struct {
	handler  *BackrestHandler
	oplog    *oplog.OpLog
	opstore  *bboltstore.BboltStore
	orch     *orchestrator.Orchestrator
	logStore *rotatinglog.RotatingLog
	config   *v1.Config
}

func createSystemUnderTest(t *testing.T, config config.ConfigStore) systemUnderTest {
	dir := t.TempDir()

	cfg, err := config.Get()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	resticBin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		t.Fatalf("Failed to find or install restic binary: %v", err)
	}
	opstore, err := bboltstore.NewBboltStore(dir + "/oplog.boltdb")
	if err != nil {
		t.Fatalf("Failed to create oplog store: %v", err)
	}
	t.Cleanup(func() { opstore.Close() })
	oplog := oplog.NewOpLog(opstore)
	logStore := rotatinglog.NewRotatingLog(dir+"/log", 10)
	orch, err := orchestrator.NewOrchestrator(
		resticBin, cfg, oplog, logStore,
	)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	for _, repo := range cfg.Repos {
		rorch, err := orch.GetRepoOrchestrator(repo.Id)
		if err != nil {
			t.Fatalf("Failed to get repo %s: %v", repo.Id, err)
		}

		if err := rorch.Init(context.Background()); err != nil {
			t.Fatalf("Failed to init repo %s: %v", repo.Id, err)
		}
	}

	h := NewBackrestHandler(config, orch, oplog, logStore)

	return systemUnderTest{
		handler:  h,
		oplog:    oplog,
		opstore:  opstore,
		orch:     orch,
		logStore: logStore,
		config:   cfg,
	}
}

func retry(t *testing.T, times int, backoff time.Duration, f func() error) error {
	t.Helper()
	var err error
	for i := 0; i < times; i++ {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(backoff)
	}
	return err
}

func getOperations(t *testing.T, log *oplog.OpLog) []*v1.Operation {
	t.Logf("Reading oplog at time %v", time.Now())
	operations := []*v1.Operation{}
	if err := log.Query(oplog.SelectAll, func(op *v1.Operation) error {
		operations = append(operations, op)
		t.Logf("operation %t status %s", op.GetOp(), op.Status)
		return nil
	}); err != nil {
		t.Fatalf("Failed to read oplog: %v", err)
	}
	return operations
}
