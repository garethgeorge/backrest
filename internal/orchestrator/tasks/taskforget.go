package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/hashicorp/go-multierror"
)

func NewOneoffForgetTask(repoID, planID string, flowID int64, at time.Time) Task {
	return &GenericOneoffTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("forget for plan %q in repo %q", planID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		OneoffTask: OneoffTask{
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationForget{},
			},
		},
		Do: func(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
			op := st.Op
			forgetOp := op.GetOperationForget()
			if forgetOp == nil {
				panic("forget task with non-forget operation")
			}

			if err := forgetHelper(ctx, st, taskRunner); err != nil {
				taskRunner.ExecuteHooks([]v1.Hook_Condition{
					v1.Hook_CONDITION_ANY_ERROR,
				}, hook.HookVars{
					Error: err.Error(),
				})
				return err
			}

			return nil
		},
	}
}

func forgetHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
	t := st.Task

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	err = repo.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err)
	}

	plan, err := taskRunner.GetPlan(t.PlanID())
	if err != nil {
		return fmt.Errorf("get plan %q: %w", t.PlanID(), err)
	}

	forgot, err := repo.Forget(ctx, plan)
	if err != nil {
		return fmt.Errorf("forget: %w", err)
	}

	forgetOp := &v1.Operation_OperationForget{}
	st.Op.Op = forgetOp

	forgetOp.OperationForget.Forget = append(forgetOp.OperationForget.Forget, forgot...)
	forgetOp.OperationForget.Policy = plan.Retention

	var ops []*v1.Operation
	for _, forgot := range forgot {
		if e := taskRunner.OpLog().ForEachBySnapshotId(forgot.Id, indexutil.CollectAll(), func(op *v1.Operation) error {
			ops = append(ops, op)
			return nil
		}); e != nil {
			err = multierror.Append(err, fmt.Errorf("cleanup snapshot %v: %w", forgot.Id, e))
		}
	}

	for _, op := range ops {
		if indexOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			indexOp.OperationIndexSnapshot.Forgot = true
			if e := taskRunner.UpdateOperation(op); err != nil {
				err = multierror.Append(err, fmt.Errorf("mark index snapshot %v as forgotten: %w", op.Id, e))
				continue
			}
		}
	}

	if len(forgot) > 0 {
		if err := taskRunner.ScheduleTask(NewOneoffPruneTask(t.RepoID(), t.PlanID(), st.Op.FlowId, time.Now(), false), TaskPriorityPrune); err != nil {
			return fmt.Errorf("schedule prune task: %w", err)
		}
	}

	return err
}
