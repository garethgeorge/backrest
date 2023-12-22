package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

type RestoreTaskOpts struct {
	PlanId     string
	RepoId     string
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

func NewOneofRestoreTask(orchestrator *Orchestrator, opts RestoreTaskOpts, at time.Time) *RestoreTask {
	return &RestoreTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		restoreOpts: opts,
		at:          &at,
	}
}

func (t *RestoreTask) Name() string {
	return fmt.Sprintf("restore snapshot %v in repo %v", t.restoreOpts.SnapshotId, t.restoreOpts.RepoId)
}

func (t *RestoreTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.restoreOpts.PlanId,
			RepoId:          t.restoreOpts.RepoId,
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
	return t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		forgetOp := &v1.Operation_OperationRestore{
			OperationRestore: &v1.OperationRestore{
				Path:   t.restoreOpts.Path,
				Target: t.restoreOpts.Target,
			},
		}
		op.Op = forgetOp
		op.UnixTimeStartMs = curTimeMillis()

		repo, err := t.orch.GetRepo(t.restoreOpts.RepoId)
		if err != nil {
			return fmt.Errorf("couldn't get repo %q: %w", t.restoreOpts.RepoId, err)
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
	})
}
