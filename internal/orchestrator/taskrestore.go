package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"go.uber.org/zap"
)

type RestoreTaskOpts struct {
	planId     string
	repoId     string
	snapshotId string
	path       string
	target     string
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
	return fmt.Sprintf("restore snapshot %v in repo %v", t.restoreOpts.snapshotId, t.restoreOpts.repoId)
}

func (t *RestoreTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.restoreOpts.planId,
			RepoId:          t.restoreOpts.repoId,
			SnapshotId:      t.restoreOpts.snapshotId,
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
				Path:   t.restoreOpts.path,
				Target: t.restoreOpts.target,
			},
		}
		op.Op = forgetOp
		op.UnixTimeStartMs = curTimeMillis()

		repo, err := t.orch.GetRepo(t.restoreOpts.repoId)
		if err != nil {
			return fmt.Errorf("couldn't get repo %q: %w", t.restoreOpts.repoId, err)
		}

		summary, err := repo.Restore(ctx, t.restoreOpts.snapshotId, t.restoreOpts.path, t.restoreOpts.target, func(entry *v1.RestoreProgressEntry) {
			zap.S().Infof("restore progress: %v", entry)
			forgetOp.OperationRestore.Status = entry
		})
		if err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}
		forgetOp.OperationRestore.Status = summary

		return nil
	})
}
