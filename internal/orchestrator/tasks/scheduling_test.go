package tasks

import (
	"os"
	"runtime"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/sqlitestore"
)

func TestScheduling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}

	os.Setenv("TZ", "America/Los_Angeles")
	defer os.Unsetenv("TZ")

	cfg := &v1.Config{
		Instance: "instance1",
		Repos: []*v1.Repo{
			{
				Id:   "repo1",
				Guid: cryptoutil.MustRandomID(cryptoutil.DefaultIDBits),
			},
			{
				Id:   "repo-absolute",
				Guid: cryptoutil.MustRandomID(cryptoutil.DefaultIDBits),
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyHours{
							MaxFrequencyHours: 1,
						},
					},
				},
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyHours{
							MaxFrequencyHours: 1,
						},
					},
				},
			},
			{
				Id:   "repo-relative",
				Guid: cryptoutil.MustRandomID(cryptoutil.DefaultIDBits),
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyHours{
							MaxFrequencyHours: 1,
						},
						Clock: v1.Schedule_CLOCK_LAST_RUN_TIME,
					},
				},
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyHours{
							MaxFrequencyHours: 1,
						},
						Clock: v1.Schedule_CLOCK_LAST_RUN_TIME,
					},
				},
			},
		},
		Plans: []*v1.Plan{
			{
				Id: "plan-cron",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_Cron{
						Cron: "0 0 * * *", // every day at midnight
					},
					Clock: v1.Schedule_CLOCK_LOCAL,
				},
			},
			{
				Id: "plan-cron-utc",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_Cron{
						Cron: "0 0 * * *", // every day at midnight
					},
					Clock: v1.Schedule_CLOCK_UTC,
				},
			},
			{
				Id: "plan-cron-since-last-run",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_Cron{
						Cron: "0 0 * * *", // every day at midnight
					},
					Clock: v1.Schedule_CLOCK_LAST_RUN_TIME,
				},
			},
			{
				Id: "plan-max-frequency-days",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_MaxFrequencyDays{
						MaxFrequencyDays: 1,
					},
					Clock: v1.Schedule_CLOCK_LOCAL,
				},
			},
			{
				Id: "plan-min-days-since-last-run",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_MaxFrequencyDays{
						MaxFrequencyDays: 1,
					},
					Clock: v1.Schedule_CLOCK_LAST_RUN_TIME,
				},
			},
			{
				Id: "plan-max-frequency-hours",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_MaxFrequencyHours{
						MaxFrequencyHours: 1,
					},
				},
			},
			{
				Id: "plan-min-hours-since-last-run",
				Schedule: &v1.Schedule{
					Schedule: &v1.Schedule_MaxFrequencyHours{
						MaxFrequencyHours: 1,
					},
					Clock: v1.Schedule_CLOCK_LAST_RUN_TIME,
				},
			},
		},
	}

	repoAbsolute := config.FindRepo(cfg, "repo-absolute")
	repoRelative := config.FindRepo(cfg, "repo-relative")
	if repoAbsolute == nil || repoRelative == nil {
		t.Fatalf("test config declaration error")
	}

	now := time.Unix(100000, 0) // 1000 seconds after the epoch as an arbitrary time for the test
	farFuture := time.Unix(999999, 0)

	tests := []struct {
		name     string
		task     Task
		ops      []*v1.Operation // operations in the log
		wantTime time.Time       // time to run the next task
	}{
		{
			name: "backup schedule max frequency days",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-max-frequency-days")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-max-frequency-days",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: now.Add(time.Hour * 24),
		},
		{
			name: "backup schedule min days since last run",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-min-days-since-last-run")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-min-days-since-last-run",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: farFuture.Add(time.Hour * 24),
		},
		{
			name: "backup schedule max frequency hours",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-max-frequency-hours")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-max-frequency-hours",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: now.Add(time.Hour),
		},
		{
			name: "backup schedule min hours since last run",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-min-hours-since-last-run")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-min-hours-since-last-run",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: farFuture.Add(time.Hour),
		},
		{
			name: "backup schedule cron",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-cron")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-cron",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: mustParseTime(t, "1970-01-02T00:00:00-08:00"),
		},
		{
			name: "backup schedule cron utc",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-cron-utc")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-cron-utc",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: mustParseTime(t, "1970-01-02T08:00:00Z"),
		},
		{
			name: "backup schedule cron since last run",
			task: NewScheduledBackupTask(config.FindRepo(cfg, "repo1"), config.FindPlan(cfg, "plan-cron-since-last-run")),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo1",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "plan-cron-since-last-run",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: mustParseTime(t, "1970-01-13T00:00:00-08:00"),
		},
		{
			name: "check schedule absolute",
			task: NewCheckTask(repoAbsolute, "_system_", false),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo-absolute",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationCheck{
						OperationCheck: &v1.OperationCheck{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: now.Add(time.Hour),
		},
		{
			name: "check schedule relative no backup yet",
			task: NewCheckTask(repoRelative, "_system_", false),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo-relative",
					RepoGuid:   repoRelative.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationCheck{
						OperationCheck: &v1.OperationCheck{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: now.Add(time.Hour),
		},
		{
			name: "check schedule relative",
			task: NewCheckTask(repoRelative, "_system_", false),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo-relative",
					RepoGuid:   repoRelative.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationCheck{
						OperationCheck: &v1.OperationCheck{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
				{
					InstanceId: "instance1",
					RepoId:     "repo-relative",
					RepoGuid:   repoRelative.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: farFuture.Add(time.Hour),
		},
		{
			name: "prune schedule absolute",
			task: NewPruneTask(repoAbsolute, "_system_", false),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo-absolute",
					RepoGuid:   repoAbsolute.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationPrune{
						OperationPrune: &v1.OperationPrune{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: now.Add(time.Hour),
		},
		{
			name: "prune schedule relative no backup yet",
			task: NewPruneTask(repoRelative, "_system_", false),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo-relative",
					RepoGuid:   repoRelative.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationPrune{
						OperationPrune: &v1.OperationPrune{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: now.Add(time.Hour),
		},
		{
			name: "prune schedule relative",
			task: NewPruneTask(repoRelative, "_system_", false),
			ops: []*v1.Operation{
				{
					InstanceId: "instance1",
					RepoId:     "repo-relative",
					RepoGuid:   repoRelative.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationPrune{
						OperationPrune: &v1.OperationPrune{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
				{
					InstanceId: "instance1",
					RepoId:     "repo-relative",
					RepoGuid:   repoRelative.Guid,
					PlanId:     "_system_",
					Op: &v1.Operation_OperationBackup{
						OperationBackup: &v1.OperationBackup{},
					},
					UnixTimeStartMs: 1000,
					UnixTimeEndMs:   farFuture.UnixMilli(),
				},
			},
			wantTime: farFuture.Add(time.Hour),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opstore, err := sqlitestore.NewMemorySqliteStore()
			if err != nil {
				t.Fatalf("failed to create opstore: %v", err)
			}
			for _, op := range tc.ops {
				if err := opstore.Add(op); err != nil {
					t.Fatalf("failed to add operation to opstore: %v", err)
				}
			}

			log, err := oplog.NewOpLog(opstore)
			if err != nil {
				t.Fatalf("failed to create oplog: %v", err)
			}

			runner := newTestTaskRunner(t, cfg, log)

			st, err := tc.task.Next(now, runner)
			if err != nil {
				t.Fatalf("failed to get next task: %v", err)
			}

			if !st.RunAt.Equal(tc.wantTime) {
				t.Errorf("got run at %v, want %v", st.RunAt.Format(time.RFC3339), tc.wantTime.Format(time.RFC3339))
			}
		})
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return tm
}
