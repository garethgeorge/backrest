package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// ForgetTask tracks a forget operation.
type ForgetTask struct {
	name string
	TaskWithOperation
	plan         *v1.Plan
	linkSnapshot string // snapshot to link the task to.
	at           *time.Time
}

var _ Task = &ForgetTask{}

func NewOneoffForgetTask(orchestrator *Orchestrator, plan *v1.Plan, linkSnapshot string, at time.Time) *ForgetTask {
	return &ForgetTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		plan:         plan,
		at:           &at,
		linkSnapshot: linkSnapshot,
	}
}

func (t *ForgetTask) Name() string {
	return fmt.Sprintf("forget for plan %q", t.plan.Id)
}

func (t *ForgetTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.plan.Id,
			RepoId:          t.plan.Repo,
			SnapshotId:      t.linkSnapshot,
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

func (t *ForgetTask) Run(ctx context.Context) error {
	if err := t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		forgetOp := &v1.Operation_OperationForget{
			OperationForget: &v1.OperationForget{},
		}
		op.Op = forgetOp

		var err error
		repo, err := t.orch.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("get repo %q: %w", t.plan.Repo, err)
		}

		err = repo.UnlockIfAutoEnabled(ctx)
		if err != nil {
			return fmt.Errorf("auto unlock repo %q: %w", t.plan.Repo, err)
		}

		forgot, err := repo.Forget(ctx, t.plan)
		if err != nil {
			return fmt.Errorf("forget: %w", err)
		}

		forgetOp.OperationForget.Forget = append(forgetOp.OperationForget.Forget, forgot...)
		forgetOp.OperationForget.Policy = t.plan.Retention

		var ops []*v1.Operation
		for _, forgot := range forgot {
			if e := t.orch.OpLog.ForEachBySnapshotId(forgot.Id, indexutil.CollectAll(), func(op *v1.Operation) error {
				ops = append(ops, op)
				return nil
			}); e != nil {
				err = multierror.Append(err, fmt.Errorf("cleanup snapshot %v: %w", forgot.Id, e))
			}
		}
		for _, op := range ops {
			if indexOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
				indexOp.OperationIndexSnapshot.Forgot = true
				if e := t.orch.OpLog.Update(op); err != nil {
					err = multierror.Append(err, fmt.Errorf("mark index snapshot %v as forgotten: %w", op.Id, e))
					continue
				}
			}
		}

		if len(forgot) > 0 {
			t.orch.ScheduleTask(NewOneoffPruneTask(t.orch, t.plan, time.Now(), false), TaskPriorityPrune)
		}

		return err
	}); err != nil {
		repo, _ := t.orch.GetRepo(t.plan.Repo)
		_ = t.orch.hookExecutor.ExecuteHooks(repo.Config(), t.plan, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:       t.Name(),
			Error:      err.Error(),
			SnapshotId: t.linkSnapshot,
		})
	}
	return nil
}
