package tasks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/metric"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
)

var maxBackupErrorHistoryLength = 20 // arbitrary limit on the number of file read errors recorded in a backup operation to prevent it from growing too large.

// BackupTask is a scheduled backup operation.
type BackupTask struct {
	BaseTask
	force  bool
	didRun bool
}

var _ Task = &BackupTask{}

func NewScheduledBackupTask(repo *v1.Repo, plan *v1.Plan) *BackupTask {
	return &BackupTask{
		BaseTask: BaseTask{
			TaskType:   "backup",
			TaskName:   fmt.Sprintf("backup for plan %q", plan.Id),
			TaskRepo:   repo,
			TaskPlanID: plan.Id,
		},
	}
}

func NewOneoffBackupTask(repo *v1.Repo, plan *v1.Plan, at time.Time) *BackupTask {
	return &BackupTask{
		BaseTask: BaseTask{
			TaskType:   "backup",
			TaskName:   fmt.Sprintf("backup for plan %q", plan.Id),
			TaskRepo:   repo,
			TaskPlanID: plan.Id,
		},
		force: true,
	}
}

func (t *BackupTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.force {
		if t.didRun {
			return NeverScheduledTask, nil
		}
		t.didRun = true
		return ScheduledTask{
			Task:  t,
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationBackup{},
			},
		}, nil
	}

	plan, err := runner.GetPlan(t.PlanID())
	if err != nil {
		return NeverScheduledTask, err
	}

	if plan.Schedule == nil {
		return NeverScheduledTask, nil
	}

	var lastRan time.Time
	if err := runner.QueryOperations(oplog.Query{}.
		SetInstanceID(runner.InstanceID()).
		SetRepoGUID(t.Repo().GetGuid()).
		SetPlanID(t.PlanID()).
		SetReversed(true), func(op *v1.Operation) error {
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED {
			return nil
		}
		if _, ok := op.Op.(*v1.Operation_OperationBackup); ok && op.UnixTimeEndMs != 0 {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last backup run time: %w", err)
	} else if lastRan.IsZero() {
		lastRan = time.Now()
	}

	nextRun, err := protoutil.ResolveSchedule(plan.Schedule, lastRan, now)
	if errors.Is(err, protoutil.ErrScheduleDisabled) {
		return NeverScheduledTask, nil
	} else if err != nil {
		return NeverScheduledTask, fmt.Errorf("resolving schedule: %w", err)
	}

	return ScheduledTask{
		Task:  t,
		RunAt: nextRun,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationBackup{},
		},
	}, nil
}

func (t *BackupTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	l := runner.Logger(ctx)

	startTime := time.Now()
	op := st.Op
	backupOp := &v1.Operation_OperationBackup{
		OperationBackup: &v1.OperationBackup{},
	}
	op.Op = backupOp

	// Helper to notify of errors during the setup phase
	notifyError := func(err error) error {
		return NotifyError(ctx, runner, t.Name(), err, v1.Hook_CONDITION_SNAPSHOT_ERROR)
	}

	repo, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return notifyError(err)
	}

	if err := repo.UnlockIfAutoEnabled(ctx); err != nil {
		return notifyError(fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err))
	}

	plan, err := runner.GetPlan(t.PlanID())
	if err != nil {
		return notifyError(err)
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_SNAPSHOT_START,
	}, HookVars{}); err != nil {
		return notifyError(fmt.Errorf("snapshot start hook: %w", err))
	}

	var sendWg sync.WaitGroup
	lastSent := time.Now() // debounce progress updates, these can endup being very frequent.
	var lastFiles []string
	fileErrorCount := 0
	summary, err := repo.Backup(ctx, plan, func(entry *restic.BackupProgressEntry) {
		sendWg.Wait()
		if entry.MessageType == "status" {
			// prevents flickering output when a status entry omits the CurrentFiles property. Largely cosmetic.
			if len(entry.CurrentFiles) == 0 {
				entry.CurrentFiles = lastFiles
			} else {
				lastFiles = entry.CurrentFiles
			}

			backupOp.OperationBackup.LastStatus = protoutil.BackupProgressEntryToProto(entry)
		} else if entry.MessageType == "error" {
			l.Warn("error processing item", zap.String("item", entry.Item), zap.Any("error", entry.Error))
			fileErrorCount++
			backupError, err := protoutil.BackupProgressEntryToBackupError(entry)
			if err != nil {
				l.Error("failed to convert backup progress entry to backup error", zap.Error(err))
				return
			}
			if len(backupOp.OperationBackup.Errors) > maxBackupErrorHistoryLength ||
				slices.ContainsFunc(backupOp.OperationBackup.Errors, func(i *v1.BackupProgressError) bool {
					return i.Item == backupError.Item
				}) {
				return
			}
			backupOp.OperationBackup.Errors = append(backupOp.OperationBackup.Errors, backupError)
		} else if entry.MessageType != "summary" && entry.MessageType != "verbose_status" {
			zap.S().Warnf("unexpected message type %q in backup progress entry", entry.MessageType)
		}

		if time.Since(lastSent) <= 1000*time.Millisecond {
			return
		}
		lastSent = time.Now()

		sendWg.Add(1)
		go func() {
			if err := runner.UpdateOperation(op); err != nil {
				l.Sugar().Errorf("failed to update oplog with progress for backup: %v", err)
			}
			sendWg.Done()
		}()
	})
	sendWg.Wait()

	if summary == nil {
		summary = &restic.BackupProgressEntry{}
	}

	metric.GetRegistry().RecordBackupSummary(t.RepoID(), t.PlanID(), summary.TotalBytesProcessed, summary.DataAdded, int64(fileErrorCount))

	vars := HookVars{
		Task:          t.Name(),
		SnapshotStats: summary,
		SnapshotId:    summary.SnapshotId,
	}

	var conditions []v1.Hook_Condition

	if err != nil {
		vars.Error = err.Error()
		if !errors.Is(err, restic.ErrPartialBackup) {
			runner.ExecuteHooks(ctx, []v1.Hook_Condition{
				v1.Hook_CONDITION_SNAPSHOT_ERROR,
				v1.Hook_CONDITION_ANY_ERROR,
				v1.Hook_CONDITION_SNAPSHOT_END,
			}, vars)
			return err
		} else {
			vars.Error = fmt.Sprintf("partial backup, %d files may not have been read completely.", len(backupOp.OperationBackup.Errors))
		}
		op.Status = v1.OperationStatus_STATUS_WARNING
		op.DisplayMessage = "Partial backup, some files may not have been read completely."
	}

	op.SnapshotId = summary.SnapshotId
	backupOp.OperationBackup.LastStatus = protoutil.BackupProgressEntryToProto(summary)
	if backupOp.OperationBackup.LastStatus == nil {
		return fmt.Errorf("expected a final backup progress entry, got nil")
	}

	l.Info("backup complete", zap.String("plan", plan.Id), zap.Duration("duration", time.Since(startTime)), zap.Any("summary", backupOp.OperationBackup.LastStatus))

	if summary.SnapshotId == "" { // support --skip-if-unchanged which returns an operation with an empty snapshot ID
		op.DisplayMessage = "No snapshot added, possibly due to no changes in the source data."

		conditions = append(conditions, v1.Hook_CONDITION_SNAPSHOT_SKIPPED)
	} else {
		// schedule followup tasks if a snapshot was added
		at := time.Now()
		if _, ok := plan.Retention.GetPolicy().(*v1.RetentionPolicy_PolicyKeepAll); plan.Retention != nil && !ok {
			if err := runner.ScheduleTask(NewOneoffForgetTask(t.Repo(), t.PlanID(), op.FlowId, at), TaskPriorityForget); err != nil {
				return fmt.Errorf("failed to schedule forget task: %w", err)
			}
		}
		if err := runner.ScheduleTask(NewOneoffIndexSnapshotsTask(t.Repo(), at), TaskPriorityIndexSnapshots); err != nil {
			return fmt.Errorf("failed to schedule index snapshots task: %w", err)
		}
	}

	if err == nil {
		conditions = append(conditions, v1.Hook_CONDITION_SNAPSHOT_SUCCESS)
	} else {
		conditions = append(conditions, v1.Hook_CONDITION_SNAPSHOT_WARNING)
	}
	conditions = append(conditions, v1.Hook_CONDITION_SNAPSHOT_END)
	if err := runner.ExecuteHooks(ctx, conditions, vars); err != nil {
		return fmt.Errorf("snapshot end hook: %w", err)
	}

	return nil
}
