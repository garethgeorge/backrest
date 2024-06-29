package tasks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
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

func NewScheduledBackupTask(plan *v1.Plan) (*BackupTask, error) {
	return &BackupTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("backup for plan %q", plan.Id),
			TaskRepoID: plan.Repo,
			TaskPlanID: plan.Id,
		},
	}, nil
}

func NewOneoffBackupTask(plan *v1.Plan, at time.Time) *BackupTask {
	return &BackupTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("backup for plan %q", plan.Id),
			TaskRepoID: plan.Repo,
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
	nextRun, err := protoutil.ResolveSchedule(plan.Schedule, now)
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

	repo, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return err
	}

	plan, err := runner.GetPlan(t.PlanID())
	if err != nil {
		return err
	}

	if err := runner.ExecuteHooks([]v1.Hook_Condition{
		v1.Hook_CONDITION_SNAPSHOT_START,
	}, HookVars{}); err != nil {
		return fmt.Errorf("snapshot start hook: %w", err)
	}

	var sendWg sync.WaitGroup
	lastSent := time.Now() // debounce progress updates, these can endup being very frequent.
	var lastFiles []string
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
			l.Sugar().Warnf("an unknown error was encountered in processing item: %v", entry.Item)
			backupError, err := protoutil.BackupProgressEntryToBackupError(entry)
			if err != nil {
				l.Sugar().Errorf("failed to convert backup progress entry to backup error: %v", err)
				return
			}
			if len(backupOp.OperationBackup.Errors) > maxBackupErrorHistoryLength ||
				slices.ContainsFunc(backupOp.OperationBackup.Errors, func(i *v1.BackupProgressError) bool {
					return i.Item == backupError.Item
				}) {
				return
			}
			backupOp.OperationBackup.Errors = append(backupOp.OperationBackup.Errors, backupError)
		} else if entry.MessageType != "summary" {
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

	vars := HookVars{
		Task:          t.Name(),
		SnapshotStats: summary,
		SnapshotId:    summary.SnapshotId,
	}

	if err != nil {
		vars.Error = err.Error()
		if !errors.Is(err, restic.ErrPartialBackup) {
			runner.ExecuteHooks([]v1.Hook_Condition{
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

		runner.ExecuteHooks([]v1.Hook_Condition{
			v1.Hook_CONDITION_SNAPSHOT_WARNING,
			v1.Hook_CONDITION_SNAPSHOT_END,
		}, vars)
	} else {
		runner.ExecuteHooks([]v1.Hook_Condition{
			v1.Hook_CONDITION_SNAPSHOT_SUCCESS,
			v1.Hook_CONDITION_SNAPSHOT_END,
		}, vars)
	}

	op.SnapshotId = summary.SnapshotId
	backupOp.OperationBackup.LastStatus = protoutil.BackupProgressEntryToProto(summary)
	if backupOp.OperationBackup.LastStatus == nil {
		return fmt.Errorf("expected a final backup progress entry, got nil")
	}

	l.Info("backup complete", zap.String("plan", plan.Id), zap.Duration("duration", time.Since(startTime)), zap.Any("summary", backupOp.OperationBackup.LastStatus))

	// schedule followup tasks
	at := time.Now()
	if _, ok := plan.Retention.GetPolicy().(*v1.RetentionPolicy_PolicyKeepAll); plan.Retention != nil && !ok {
		if err := runner.ScheduleTask(NewOneoffForgetTask(t.RepoID(), t.PlanID(), op.FlowId, at), TaskPriorityForget); err != nil {
			return fmt.Errorf("failed to schedule forget task: %w", err)
		}
	}
	if err := runner.ScheduleTask(NewOneoffIndexSnapshotsTask(t.RepoID(), at), TaskPriorityIndexSnapshots); err != nil {
		return fmt.Errorf("failed to schedule index snapshots task: %w", err)
	}

	return nil
}
