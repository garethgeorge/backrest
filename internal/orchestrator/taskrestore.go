package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"go.uber.org/zap"
)

type RestoreTaskOpts struct {
	Plan       *v1.Plan
	SnapshotId string
	Path       string
	Target     string
}

// RestoreTask tracks a forget operation.
type RestoreTask struct {
	TaskWithOperation
	restoreOpts RestoreTaskOpts
	at          *time.Time
}

var _ Task = &RestoreTask{}

func NewOneoffRestoreTask(orchestrator *Orchestrator, opts RestoreTaskOpts, at time.Time) *RestoreTask {
	return &RestoreTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		restoreOpts: opts,
		at:          &at,
	}
}

func (t *RestoreTask) Name() string {
	return fmt.Sprintf("restore snapshot %v in repo %v", t.restoreOpts.SnapshotId, t.restoreOpts.Plan.Repo)
}

func (t *RestoreTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.restoreOpts.Plan.Id,
			RepoId:          t.restoreOpts.Plan.Repo,
			SnapshotId:      t.restoreOpts.SnapshotId,
			UnixTimeStartMs: timeToUnixMillis(*ret),
			Status:          v1.OperationStatus_STATUS_PENDING,
			Op:              &v1.Operation_OperationRestore{},
		}); err != nil {
			zap.S().Errorf("task %v failed to add operation to oplog: %v", t.Name(), err)
			return nil
		}
	}
	return ret
}

func (t *RestoreTask) Run(ctx context.Context) error {
	if err := t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		forgetOp := &v1.Operation_OperationRestore{
			OperationRestore: &v1.OperationRestore{
				Path:   t.restoreOpts.Path,
				Target: t.restoreOpts.Target,
			},
		}
		op.Op = forgetOp
		op.UnixTimeStartMs = curTimeMillis()

		repo, err := t.orch.GetRepo(t.restoreOpts.Plan.Repo)
		if err != nil {
			return fmt.Errorf("couldn't get repo %q: %w", t.restoreOpts.Plan.Repo, err)
		}

		lastSent := time.Now() // debounce progress updates, these can endup being very frequent.
		summary, err := repo.Restore(ctx, t.restoreOpts.SnapshotId, t.restoreOpts.Path, t.restoreOpts.Target, func(entry *v1.RestoreProgressEntry) {
			if time.Since(lastSent) < 250*time.Millisecond {
				return
			}
			lastSent = time.Now()

			zap.S().Infof("restore progress: %v", entry)
			forgetOp.OperationRestore.Status = entry
			if err := t.orch.OpLog.Update(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for restore: %v", err)
			}
		})
		if err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}
		forgetOp.OperationRestore.Status = summary

		return nil
	}); err != nil {
		repo, _ := t.orch.GetRepo(t.restoreOpts.Plan.Repo)
		hook.ExecuteHooks(t.orch.OpLog, repo.Config(), nil, t.restoreOpts.SnapshotId, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:  t.Name(),
			Error: err.Error(),
		})
		return err
	}
	return nil
}
