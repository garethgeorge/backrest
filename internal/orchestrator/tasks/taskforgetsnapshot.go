package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// ForgetTask tracks a forget operation.
type ForgetSnapshotTask struct {
	TaskWithOperation
	repoId         string
	planId         string
	forgetSnapshot string
	at             *time.Time
}

var _ Task = &ForgetSnapshotTask{}

func NewOneoffForgetSnapshotTask(orchestrator *Orchestrator, repoId, planId, forgetSnapshot string, at time.Time) *ForgetSnapshotTask {
	return &ForgetSnapshotTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		repoId:         repoId,
		planId:         planId,
		at:             &at,
		forgetSnapshot: forgetSnapshot,
	}
}

func (t *ForgetSnapshotTask) Name() string {
	return fmt.Sprintf("forget snapshot %q", t.forgetSnapshot)
}

func (t *ForgetSnapshotTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.planId,
			RepoId:          t.repoId,
			UnixTimeStartMs: timeToUnixMillis(*ret),
			Status:          v1.OperationStatus_STATUS_PENDING,
			Op:              &v1.Operation_OperationForget{},
		}); err != nil {
			zap.S().Errorf("task %v failed to add operation to oplog: %v", t.Name(), err)
			return nil
		}
	}
	return ret
}

func (t *ForgetSnapshotTask) Run(ctx context.Context) error {
	id := t.op.Id
	if err := t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		repo, err := t.orch.GetRepo(t.repoId)
		if err != nil {
			return fmt.Errorf("get repo %q: %w", t.repoId, err)
		}

		// Find snapshot to forget
		var ops []*v1.Operation
		t.orch.OpLog.ForEachBySnapshotId(t.forgetSnapshot, indexutil.CollectAll(), func(op *v1.Operation) error {
			ops = append(ops, op)
			return nil
		})

		for _, op := range ops {
			if indexOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
				err := repo.ForgetSnapshot(ctx, op.SnapshotId)
				if err != nil {
					return fmt.Errorf("forget %q: %w", op.SnapshotId, err)
				}
				indexOp.OperationIndexSnapshot.Forgot = true
				if e := t.orch.OpLog.Update(op); err != nil {
					err = multierror.Append(err, fmt.Errorf("mark index snapshot %v as forgotten: %w", op.Id, e))
					continue
				}
			}
		}

		return err
	}); err != nil {
		return err
	}

	return t.orch.OpLog.Delete(id)
}
