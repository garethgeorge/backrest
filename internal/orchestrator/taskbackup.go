package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/gitploy-io/cronexpr"
	"go.uber.org/zap"
)

var maxBackupErrorHistoryLength = 20 // arbitrary limit on the number of file read errors recorded in a backup operation to prevent it from growing too large.

// BackupTask is a scheduled backup operation.
type BackupTask struct {
	name      string
	plan      *v1.Plan
	scheduler func(curTime time.Time) *time.Time
}

var _ Task = &BackupTask{}

func NewScheduledBackupTask(orchestrator *Orchestrator, plan *v1.Plan) (*BackupTask, error) {
	sched, err := cronexpr.ParseInLocation(plan.Cron, time.Now().Location().String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse schedule %q: %w", plan.Cron, err)
	}

	return &BackupTask{
		name: fmt.Sprintf("backup for plan %q", plan.Id),
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		plan: plan,
		scheduler: func(curTime time.Time) *time.Time {
			next := sched.Next(curTime)
			return &next
		},
	}, nil
}

func NewOneoffBackupTask(orchestrator *Orchestrator, plan *v1.Plan, at time.Time) *BackupTask {
	didOnce := false
	return &BackupTask{
		name: fmt.Sprintf("onetime backup for plan %q", plan.Id),
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		plan: plan,
		scheduler: func(curTime time.Time) *time.Time {
			if didOnce {
				return nil
			}
			didOnce = true
			return &at
		},
	}
}

func (t *BackupTask) Name() string {
	return t.name
}

func (t *BackupTask) Next(now time.Time) *time.Time {
	next := t.scheduler(now)
	if next == nil {
		return nil
	}

	if err := t.setOperation(&v1.Operation{
		PlanId:          t.plan.Id,
		RepoId:          t.plan.Repo,
		UnixTimeStartMs: timeToUnixMillis(*next),
		Status:          v1.OperationStatus_STATUS_PENDING,
		Op:              &v1.Operation_OperationBackup{},
	}); err != nil {
		zap.S().Errorf("task %v failed to add operation to oplog: %v", t.Name(), err)
	}

	return next
}

func (t *BackupTask) Run(ctx context.Context) error {
	return t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		return backupHelper(ctx, t, t.orch, t.plan, op)
	})
}

// backupHelper does a backup.
func backupHelper(ctx context.Context, t Task, orchestrator *Orchestrator, plan *v1.Plan, op *v1.Operation) error {
	startTime := time.Now()
	backupOp := &v1.Operation_OperationBackup{
		OperationBackup: &v1.OperationBackup{},
	}
	op.Op = backupOp

	zap.L().Info("Starting backup", zap.String("plan", plan.Id), zap.Int64("opId", op.Id))
	repo, err := orchestrator.GetRepo(plan.Repo)
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", plan.Repo, err)
	}

	// Run start hooks e.g. preflight checks and backup start notifications.
	if err := orchestrator.hookExecutor.ExecuteHooks(repo.Config(), plan, []v1.Hook_Condition{
		v1.Hook_CONDITION_SNAPSHOT_START,
	}, hook.HookVars{
		Task: t.Name(),
	}); err != nil {
		var cancelErr *hook.HookErrorRequestCancel
		if errors.As(err, &cancelErr) {
			op.Status = v1.OperationStatus_STATUS_USER_CANCELLED // user visible cancelled status
			op.DisplayMessage = err.Error()
			return nil
		}

		// If the snapshot start hook fails we trigger error notification hooks.
		retErr := fmt.Errorf("hook failed: %w", err)
		_ = orchestrator.hookExecutor.ExecuteHooks(repo.Config(), plan, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:  t.Name(),
			Error: retErr.Error(),
		})

		return retErr
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
			zap.S().Warnf("an unknown error was encountered in processing item: %v", entry.Item)
			backupError, err := protoutil.BackupProgressEntryToBackupError(entry)
			if err != nil {
				zap.S().Errorf("failed to convert backup progress entry to backup error: %v", err)
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

		if time.Since(lastSent) < 1*time.Second {
			return
		}
		lastSent = time.Now()

		sendWg.Add(1)
		go func() {
			if err := orchestrator.OpLog.Update(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for backup: %v", err)
			}
			sendWg.Done()
		}()
	})

	sendWg.Wait()

	if summary == nil {
		summary = &restic.BackupProgressEntry{}
	}

	vars := hook.HookVars{
		Task:          t.Name(),
		SnapshotStats: summary,
		SnapshotId:    summary.SnapshotId,
	}
	if err != nil {
		vars.Error = err.Error()
		if !errors.Is(err, restic.ErrPartialBackup) {
			_ = orchestrator.hookExecutor.ExecuteHooks(repo.Config(), plan, []v1.Hook_Condition{
				v1.Hook_CONDITION_SNAPSHOT_ERROR, v1.Hook_CONDITION_ANY_ERROR,
			}, vars)
			return fmt.Errorf("repo.Backup for repo %q: %w", plan.Repo, err)
		}
		op.Status = v1.OperationStatus_STATUS_WARNING
		op.DisplayMessage = "Partial backup, some files may not have been read completely."
	}

	orchestrator.hookExecutor.ExecuteHooks(repo.Config(), plan, []v1.Hook_Condition{
		v1.Hook_CONDITION_SNAPSHOT_END,
	}, vars)

	op.SnapshotId = summary.SnapshotId
	backupOp.OperationBackup.LastStatus = protoutil.BackupProgressEntryToProto(summary)
	if backupOp.OperationBackup.LastStatus == nil {
		return fmt.Errorf("expected a final backup progress entry, got nil")
	}

	zap.L().Info("Backup complete", zap.String("plan", plan.Id), zap.Duration("duration", time.Since(startTime)), zap.Any("summary", backupOp.OperationBackup.LastStatus))

	// schedule followup tasks
	at := time.Now()
	if _, ok := plan.Retention.GetPolicy().(*v1.RetentionPolicy_PolicyKeepAll); plan.Retention != nil && !ok {
		orchestrator.ScheduleTask(NewOneoffForgetTask(orchestrator, plan, op.SnapshotId, at), TaskPriorityForget)
	}
	orchestrator.ScheduleTask(NewOneoffIndexSnapshotsTask(orchestrator, plan.Repo, at), TaskPriorityIndexSnapshots)

	return nil
}
