package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

func NewOneoffForgetTask(repoID, planID string, flowID int64, at time.Time) Task {
	return &GenericOneoffTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("forget for plan %q in repo %q", repoID, planID),
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
	oplog := taskRunner.OpLog()

	r, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	err = r.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err)
	}

	plan, err := taskRunner.GetPlan(t.PlanID())
	if err != nil {
		return fmt.Errorf("get plan %q: %w", t.PlanID(), err)
	}

	tags := []string{repo.TagForPlan(t.PlanID())}
	if compat, err := useLegacyCompatMode(oplog, t.PlanID()); err != nil {
		return fmt.Errorf("check legacy compat mode: %w", err)
	} else if !compat {
		tags = append(tags, repo.TagForInstance(taskRunner.Config().Instance))
	} else {
		zap.L().Warn("forgetting snapshots without instance ID, using legacy behavior (e.g. --tags not including instance ID)")
		zap.S().Warnf("to avoid this warning, tag all snapshots with the instance ID e.g. by running: \r\n"+
			"restic tag --set '%s' --set '%s' --tag '%s'", repo.TagForPlan(t.PlanID()), repo.TagForInstance(taskRunner.Config().Instance), repo.TagForPlan(t.PlanID()))
	}

	// check if any other instance IDs exist in the repo (unassociated don't count)
	forgot, err := r.Forget(ctx, plan, tags)
	if err != nil {
		return fmt.Errorf("forget: %w", err)
	}

	forgetOp := &v1.Operation_OperationForget{
		OperationForget: &v1.OperationForget{},
	}
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

	zap.S().Debugf("found %v snapshots were forgotten, marking this in oplog", len(ops))

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

// useLegacyCompatMode checks if there are any snapshots that were created without a `created-by` tag still exist in the repo.
// The property is overridden if mixed `created-by` tag values are found.
func useLegacyCompatMode(oplog *oplog.OpLog, planID string) (bool, error) {
	instanceIDs := make(map[string]struct{})
	if err := oplog.ForEachByPlan(planID, indexutil.CollectAll(), func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			tags := snapshotOp.OperationIndexSnapshot.GetSnapshot().GetTags()
			instanceIDs[repo.InstanceIDFromTags(tags)] = struct{}{}
		}
		return nil
	}); err != nil {
		return false, err
	}
	if _, ok := instanceIDs[""]; !ok {
		return false, nil
	}
	delete(instanceIDs, "")
	if len(instanceIDs) > 1 {
		zap.L().Warn("found mixed instance IDs in indexed snapshots, forcing forget to use new behavior (e.g. --tags including instance ID) despite the presence of legacy (e.g. untagged) snapshots.")
		return false, nil
	}
	zap.L().Warn("found legacy snapshots without instance ID, forget will use legacy behavior e.g. --tags not including instance ID")
	return true, nil
}
