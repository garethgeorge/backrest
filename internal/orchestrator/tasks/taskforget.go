package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

func NewOneoffForgetTask(repo *v1.Repo, planID string, flowID int64, at time.Time) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType:   "forget",
				TaskName:   fmt.Sprintf("forget for plan %q in repo %q", repo.Id, planID),
				TaskRepo:   repo,
				TaskPlanID: planID,
			},
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

			return forgetHelper(ctx, st, taskRunner)
		},
	}
}

func forgetHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
	t := st.Task
	l := taskRunner.Logger(ctx)

	// Helper to notify of errors
	notifyError := func(err error) error {
		return NotifyError(ctx, taskRunner, t.Name(), err, v1.Hook_CONDITION_FORGET_ERROR)
	}

	r, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return notifyError(fmt.Errorf("get repo %q: %w", t.RepoID(), err))
	}

	err = r.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return notifyError(fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err))
	}

	plan, err := taskRunner.GetPlan(t.PlanID())
	if err != nil {
		return notifyError(fmt.Errorf("get plan %q: %w", t.PlanID(), err))
	}

	// execute hooks
	if err := taskRunner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_FORGET_START,
	}, HookVars{Plan: plan}); err != nil {
		return notifyError(fmt.Errorf("forget start hook: %w", err))
	}

	tags := []string{repo.TagForPlan(t.PlanID())}
	if compat, err := useLegacyCompatMode(l, taskRunner, t.Repo().GetGuid(), t.PlanID()); err != nil {
		return notifyError(fmt.Errorf("check legacy compat mode: %w", err))
	} else if !compat {
		tags = append(tags, repo.TagForInstance(taskRunner.Config().Instance))
	} else {
		l.Warn("forgetting snapshots without instance ID, using legacy behavior (e.g. --tags not including instance ID)")
		l.Sugar().Warnf("to avoid this warning, tag all snapshots with the instance ID e.g. by running: \r\n"+
			"restic tag --set '%s' --set '%s' --tag '%s'", repo.TagForPlan(t.PlanID()), repo.TagForInstance(taskRunner.Config().Instance), repo.TagForPlan(t.PlanID()))
	}

	// check if any other instance IDs exist in the repo (unassociated don't count)
	forgot, err := r.Forget(ctx, plan, tags)

	forgetOp := &v1.Operation_OperationForget{
		OperationForget: &v1.OperationForget{},
	}
	st.Op.Op = forgetOp

	forgetOp.OperationForget.Forget = append(forgetOp.OperationForget.Forget, forgot...)
	forgetOp.OperationForget.Policy = plan.Retention

	var ops []*v1.Operation
	for _, forgot := range forgot {
		if e := taskRunner.QueryOperations(oplog.Query{}.
			SetRepoGUID(t.Repo().GetGuid()).
			SetSnapshotID(forgot.Id), func(op *v1.Operation) error {
			ops = append(ops, op)
			return nil
		}); e != nil {
			err = multierror.Append(err, fmt.Errorf("cleanup snapshot %v: %w", forgot.Id, e))
		}
	}

	l.Sugar().Debugf("found %v snapshots were forgotten, marking this in oplog", len(ops))

	for _, op := range ops {
		if indexOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			indexOp.OperationIndexSnapshot.Forgot = true
			if e := taskRunner.UpdateOperation(op); err != nil {
				err = multierror.Append(err, fmt.Errorf("mark index snapshot %v as forgotten: %w", op.Id, e))
				continue
			}
		}
	}

	if err != nil {
		return notifyError(fmt.Errorf("forget: %w", err))
	} else if e := taskRunner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_FORGET_SUCCESS,
	}, HookVars{}); e != nil {
		return fmt.Errorf("forget end hook: %w", e)
	}

	return err
}

// useLegacyCompatMode checks if there are any snapshots that were created without a `created-by` tag still exist in the repo.
// The property is overridden if mixed `created-by` tag values are found.
func useLegacyCompatMode(l *zap.Logger, taskRunner TaskRunner, repoGUID, planID string) (bool, error) {
	instanceIDs := make(map[string]struct{})
	if err := taskRunner.QueryOperations(oplog.Query{}.SetRepoGUID(repoGUID).SetPlanID(planID).SetReversed(true), func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok && !snapshotOp.OperationIndexSnapshot.GetForgot() {
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
		l.Sugar().Warn("found mixed instance IDs in indexed snapshots, overriding legacy forget behavior to include instance ID tags. This may result in unexpected behavior -- please inspect the tags on your snapshots.")
		return false, nil
	}
	l.Sugar().Warn("found legacy snapshots without instance ID, recommending legacy forget behavior.")
	return true, nil
}
