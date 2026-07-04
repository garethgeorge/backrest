package tasks

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/sqlitestore"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSnapshotID = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

func newTestConfig(repo *v1.Repo, plans ...*v1.Plan) *v1.Config {
	return &v1.Config{
		Instance: "test-instance",
		Repos:    []*v1.Repo{repo},
		Plans:    plans,
	}
}

func setupTestRunner(t *testing.T, cfg *v1.Config, fake *fakeRepoOrchestrator) *testTaskRunner {
	t.Helper()
	opstore, err := sqlitestore.NewMemorySqliteStore(t)
	require.NoError(t, err)
	ol, err := oplog.NewOpLog(opstore)
	require.NoError(t, err)
	runner := newTestTaskRunner(t, cfg, ol)
	runner.orchestrator = fake
	return runner
}

func nextAndCreate(t *testing.T, task Task, runner *testTaskRunner) ScheduledTask {
	t.Helper()
	st, err := task.Next(time.Now(), runner)
	require.NoError(t, err)
	st.Task = task
	if st.Op != nil {
		// Populate fields the orchestrator normally sets before storing.
		if st.Op.RepoId == "" && task.Repo() != nil {
			st.Op.RepoId = task.Repo().Id
		}
		if st.Op.RepoGuid == "" && task.Repo() != nil {
			st.Op.RepoGuid = task.Repo().Guid
		}
		if st.Op.PlanId == "" {
			st.Op.PlanId = task.PlanID()
		}
		if st.Op.InstanceId == "" {
			st.Op.InstanceId = runner.InstanceID()
		}
		if st.Op.FlowId == 0 {
			st.Op.FlowId = 1
		}
		if st.Op.UnixTimeStartMs == 0 {
			st.Op.UnixTimeStartMs = time.Now().UnixMilli()
		}
		require.NoError(t, runner.CreateOperation(st.Op))
	}
	return st
}

func hookContains(calls []hookCall, cond v1.Hook_Condition) bool {
	for _, c := range calls {
		for _, e := range c.Events {
			if e == cond {
				return true
			}
		}
	}
	return false
}

// --- PruneTask tests ---

func TestPruneTaskRun(t *testing.T) {
	tests := []struct {
		name          string
		fake          *fakeRepoOrchestrator
		wantErr       bool
		wantHooks     []v1.Hook_Condition
		wantScheduled int
		scheduledType string
	}{
		{
			name:          "success",
			fake:          &fakeRepoOrchestrator{},
			wantHooks:     []v1.Hook_Condition{v1.Hook_CONDITION_PRUNE_START, v1.Hook_CONDITION_PRUNE_SUCCESS},
			wantScheduled: 1,
			scheduledType: "stats",
		},
		{
			name:      "prune error",
			fake:      &fakeRepoOrchestrator{pruneErr: fmt.Errorf("prune failed")},
			wantErr:   true,
			wantHooks: []v1.Hook_Condition{v1.Hook_CONDITION_PRUNE_START, v1.Hook_CONDITION_PRUNE_ERROR, v1.Hook_CONDITION_ANY_ERROR},
		},
		{
			name:      "unlock error",
			fake:      &fakeRepoOrchestrator{unlockErr: fmt.Errorf("unlock failed")},
			wantErr:   true,
			wantHooks: []v1.Hook_Condition{v1.Hook_CONDITION_PRUNE_ERROR, v1.Hook_CONDITION_ANY_ERROR},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewPruneTask(repo, PlanForSystemTasks, true)
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for _, cond := range tc.wantHooks {
				assert.True(t, hookContains(runner.hookCalls, cond), "expected hook %v", cond)
			}

			assert.Len(t, runner.scheduledTasks, tc.wantScheduled)
			if tc.scheduledType != "" && len(runner.scheduledTasks) > 0 {
				assert.Equal(t, tc.scheduledType, runner.scheduledTasks[0].Task.Type())
			}
		})
	}
}

// --- CheckTask tests ---

func TestCheckTaskRun(t *testing.T) {
	tests := []struct {
		name      string
		fake      *fakeRepoOrchestrator
		wantErr   bool
		wantHooks []v1.Hook_Condition
	}{
		{
			name:      "success",
			fake:      &fakeRepoOrchestrator{},
			wantHooks: []v1.Hook_Condition{v1.Hook_CONDITION_CHECK_START, v1.Hook_CONDITION_CHECK_SUCCESS},
		},
		{
			name:      "check error",
			fake:      &fakeRepoOrchestrator{checkErr: fmt.Errorf("check failed")},
			wantErr:   true,
			wantHooks: []v1.Hook_Condition{v1.Hook_CONDITION_CHECK_START, v1.Hook_CONDITION_CHECK_ERROR, v1.Hook_CONDITION_ANY_ERROR},
		},
		{
			name:      "unlock error",
			fake:      &fakeRepoOrchestrator{unlockErr: fmt.Errorf("unlock failed")},
			wantErr:   true,
			wantHooks: []v1.Hook_Condition{v1.Hook_CONDITION_CHECK_ERROR, v1.Hook_CONDITION_ANY_ERROR},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewCheckTask(repo, PlanForSystemTasks, true)
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for _, cond := range tc.wantHooks {
				assert.True(t, hookContains(runner.hookCalls, cond), "expected hook %v", cond)
			}
		})
	}
}

// --- StatsTask tests ---

func TestStatsTaskRun(t *testing.T) {
	tests := []struct {
		name    string
		fake    *fakeRepoOrchestrator
		wantErr bool
	}{
		{
			name: "success",
			fake: &fakeRepoOrchestrator{
				statsResult: &v1.RepoStats{
					TotalSize: 1000,
				},
			},
		},
		{
			name:    "stats error",
			fake:    &fakeRepoOrchestrator{statsErr: fmt.Errorf("stats failed")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewStatsTask(repo, PlanForSystemTasks, true)
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				statsOp := st.Op.GetOperationStats()
				require.NotNil(t, statsOp)
				assert.Equal(t, tc.fake.statsResult.TotalSize, statsOp.Stats.TotalSize)
			}
		})
	}
}

// --- BackupTask tests ---

func TestBackupTaskRun(t *testing.T) {
	tests := []struct {
		name          string
		dryRun        bool
		fake          *fakeRepoOrchestrator
		repo          *v1.Repo
		plan          *v1.Plan
		wantErr       bool
		wantHooks     []v1.Hook_Condition
		wantNotHooks  []v1.Hook_Condition
		wantScheduled []string // expected scheduled task types
	}{
		{
			name: "successful backup with retention",
			fake: &fakeRepoOrchestrator{
				backupResult: &restic.BackupProgressEntry{
					MessageType:         "summary",
					SnapshotId:          testSnapshotID,
					TotalBytesProcessed: 1000,
				},
			},
			plan: &v1.Plan{
				Id:   "plan1",
				Repo: "repo1",
				Retention: &v1.RetentionPolicy{
					Policy: &v1.RetentionPolicy_PolicyKeepLastN{PolicyKeepLastN: 5},
				},
			},
			wantHooks:     []v1.Hook_Condition{v1.Hook_CONDITION_SNAPSHOT_START, v1.Hook_CONDITION_SNAPSHOT_SUCCESS, v1.Hook_CONDITION_SNAPSHOT_END},
			wantScheduled: []string{"forget", "index_snapshots"},
		},
		{
			name: "successful backup no retention",
			fake: &fakeRepoOrchestrator{
				backupResult: &restic.BackupProgressEntry{
					MessageType:         "summary",
					SnapshotId:          testSnapshotID,
					TotalBytesProcessed: 1000,
				},
			},
			plan: &v1.Plan{
				Id:   "plan1",
				Repo: "repo1",
			},
			wantHooks:     []v1.Hook_Condition{v1.Hook_CONDITION_SNAPSHOT_START, v1.Hook_CONDITION_SNAPSHOT_SUCCESS, v1.Hook_CONDITION_SNAPSHOT_END},
			wantScheduled: []string{"index_snapshots"},
		},
		{
			name: "successful backup with repo-level scheduled forget skips per-plan forget",
			repo: &v1.Repo{
				Id: "repo1", Guid: "guid1",
				ForgetPolicy: &v1.ForgetPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyDays{MaxFrequencyDays: 1},
					},
				},
			},
			fake: &fakeRepoOrchestrator{
				backupResult: &restic.BackupProgressEntry{
					MessageType: "summary",
					SnapshotId:  testSnapshotID,
				},
			},
			plan: &v1.Plan{
				Id:   "plan1",
				Repo: "repo1",
				Retention: &v1.RetentionPolicy{
					Policy: &v1.RetentionPolicy_PolicyKeepLastN{PolicyKeepLastN: 5},
				},
			},
			wantScheduled: []string{"index_snapshots"}, // no forget
		},
		{
			name:    "backup error",
			fake:    &fakeRepoOrchestrator{backupErr: fmt.Errorf("backup failed")},
			plan:    &v1.Plan{Id: "plan1", Repo: "repo1"},
			wantErr: true,
			wantHooks: []v1.Hook_Condition{
				v1.Hook_CONDITION_SNAPSHOT_START,
				v1.Hook_CONDITION_SNAPSHOT_ERROR,
				v1.Hook_CONDITION_ANY_ERROR,
				v1.Hook_CONDITION_SNAPSHOT_END,
			},
		},
		{
			name:    "unlock error",
			fake:    &fakeRepoOrchestrator{unlockErr: fmt.Errorf("unlock failed")},
			plan:    &v1.Plan{Id: "plan1", Repo: "repo1"},
			wantErr: true,
			wantHooks: []v1.Hook_Condition{
				v1.Hook_CONDITION_SNAPSHOT_ERROR,
				v1.Hook_CONDITION_ANY_ERROR,
			},
		},
		{
			name:   "dry run backup",
			dryRun: true,
			fake: &fakeRepoOrchestrator{
				backupResult: &restic.BackupProgressEntry{
					MessageType: "summary",
					SnapshotId:  testSnapshotID,
				},
			},
			plan:          &v1.Plan{Id: "plan1", Repo: "repo1"},
			wantScheduled: nil,
		},
		{
			name: "skip if unchanged",
			fake: &fakeRepoOrchestrator{
				backupResult: &restic.BackupProgressEntry{
					MessageType: "summary",
					SnapshotId:  "", // empty = no changes
				},
			},
			plan:          &v1.Plan{Id: "plan1", Repo: "repo1"},
			wantHooks:     []v1.Hook_Condition{v1.Hook_CONDITION_SNAPSHOT_START, v1.Hook_CONDITION_SNAPSHOT_SKIPPED, v1.Hook_CONDITION_SNAPSHOT_END},
			wantScheduled: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.repo
			if repo == nil {
				repo = &v1.Repo{Id: "repo1", Guid: "guid1"}
			}
			cfg := newTestConfig(repo, tc.plan)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewOneoffBackupTask(repo, tc.plan, time.Now(), tc.dryRun)
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for _, cond := range tc.wantHooks {
				assert.True(t, hookContains(runner.hookCalls, cond), "expected hook %v", cond)
			}
			for _, cond := range tc.wantNotHooks {
				assert.False(t, hookContains(runner.hookCalls, cond), "unexpected hook %v", cond)
			}

			var scheduledTypes []string
			for _, s := range runner.scheduledTasks {
				scheduledTypes = append(scheduledTypes, s.Task.Type())
			}
			if tc.wantScheduled != nil {
				assert.Equal(t, tc.wantScheduled, scheduledTypes)
			}
		})
	}
}

// --- ForgetSnapshot task tests ---

func TestForgetSnapshotTaskRun(t *testing.T) {
	tests := []struct {
		name    string
		fake    *fakeRepoOrchestrator
		wantErr bool
	}{
		{
			name: "success",
			fake: &fakeRepoOrchestrator{},
		},
		{
			name:    "forget snapshot error",
			fake:    &fakeRepoOrchestrator{forgetSnapshotErr: fmt.Errorf("forget failed")},
			wantErr: true,
		},
		{
			name:    "unlock error",
			fake:    &fakeRepoOrchestrator{unlockErr: fmt.Errorf("unlock failed")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewOneoffForgetSnapshotTask(repo, "plan1", 1, time.Now(), testSnapshotID)
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// On success, the task schedules an index snapshots task
				require.Len(t, runner.scheduledTasks, 1)
				assert.Equal(t, "index_snapshots", runner.scheduledTasks[0].Task.Type())
			}
		})
	}
}

// --- ScheduledForgetTask tests ---

func TestScheduledForgetTaskRun(t *testing.T) {
	tests := []struct {
		name          string
		fake          *fakeRepoOrchestrator
		wantErr       bool
		wantHooks     []v1.Hook_Condition
		wantScheduled int
	}{
		{
			name:          "success",
			fake:          &fakeRepoOrchestrator{},
			wantHooks:     []v1.Hook_Condition{v1.Hook_CONDITION_FORGET_START, v1.Hook_CONDITION_FORGET_SUCCESS},
			wantScheduled: 1, // stats task
		},
		{
			name:    "forget error",
			fake:    &fakeRepoOrchestrator{forgetErr: fmt.Errorf("forget failed")},
			wantErr: true,
			wantHooks: []v1.Hook_Condition{
				v1.Hook_CONDITION_FORGET_START,
				v1.Hook_CONDITION_FORGET_ERROR,
				v1.Hook_CONDITION_ANY_ERROR,
			},
		},
		{
			name:    "unlock error",
			fake:    &fakeRepoOrchestrator{unlockErr: fmt.Errorf("unlock failed")},
			wantErr: true,
			wantHooks: []v1.Hook_Condition{
				v1.Hook_CONDITION_FORGET_ERROR,
				v1.Hook_CONDITION_ANY_ERROR,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{
				Id: "repo1", Guid: "guid1",
				ForgetPolicy: &v1.ForgetPolicy{
					Retention: &v1.RetentionPolicy{
						Policy: &v1.RetentionPolicy_PolicyKeepLastN{PolicyKeepLastN: 5},
					},
				},
			}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewScheduledForgetTask(repo, PlanForSystemTasks, true)
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for _, cond := range tc.wantHooks {
				assert.True(t, hookContains(runner.hookCalls, cond), "expected hook %v", cond)
			}

			assert.Len(t, runner.scheduledTasks, tc.wantScheduled)
		})
	}
}

// --- RestoreTask tests ---

func TestRestoreTaskRun(t *testing.T) {
	tests := []struct {
		name    string
		fake    *fakeRepoOrchestrator
		wantErr bool
	}{
		{
			name: "success",
			fake: &fakeRepoOrchestrator{
				restoreResult: &v1.RestoreProgressEntry{
					MessageType:   "summary",
					TotalFiles:    10,
					TotalBytes:    5000,
					FilesRestored: 10,
					BytesRestored: 5000,
				},
			},
		},
		{
			name:    "restore error",
			fake:    &fakeRepoOrchestrator{restoreErr: fmt.Errorf("restore failed")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewOneoffRestoreTask(repo, "plan1", 1, time.Now(), testSnapshotID, "/data", "/tmp/restore")
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				restoreOp := st.Op.GetOperationRestore()
				require.NotNil(t, restoreOp)
				assert.NotNil(t, restoreOp.LastStatus)
			}
		})
	}
}

// --- RunCommand tests ---

func TestRunCommandTaskRun(t *testing.T) {
	tests := []struct {
		name    string
		fake    *fakeRepoOrchestrator
		wantErr bool
	}{
		{
			name: "success",
			fake: &fakeRepoOrchestrator{},
		},
		{
			name:    "command error",
			fake:    &fakeRepoOrchestrator{runCommandErr: fmt.Errorf("command failed")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			cfg := newTestConfig(repo)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewOneoffRunCommandTask(repo, "plan1", 1, time.Now(), "echo hello")
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// --- IndexSnapshots tests ---

func TestIndexSnapshotsTaskRun(t *testing.T) {
	tests := []struct {
		name    string
		fake    *fakeRepoOrchestrator
		wantErr bool
	}{
		{
			name: "no snapshots",
			fake: &fakeRepoOrchestrator{
				snapshots: []*restic.Snapshot{},
			},
		},
		{
			name: "indexes new snapshots",
			fake: &fakeRepoOrchestrator{
				snapshots: []*restic.Snapshot{
					{
						Id:              testSnapshotID,
						Time:            time.Now().Format(time.RFC3339Nano),
						Tags:            []string{"plan:plan1", "created-by:test-instance"},
						SnapshotSummary: restic.SnapshotSummary{},
					},
				},
			},
		},
		{
			name:    "snapshots error",
			fake:    &fakeRepoOrchestrator{snapshotsErr: fmt.Errorf("snapshots failed")},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &v1.Repo{Id: "repo1", Guid: "guid1"}
			plan := &v1.Plan{Id: "plan1", Repo: "repo1"}
			cfg := newTestConfig(repo, plan)
			runner := setupTestRunner(t, cfg, tc.fake)

			task := NewOneoffIndexSnapshotsTask(repo, time.Now())
			st := nextAndCreate(t, task, runner)

			err := task.Run(context.Background(), st, runner)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
