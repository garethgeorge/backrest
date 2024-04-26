package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/orchestrator"
)

func NewOneoffForgetSnapshotTask(repoID, planID string, flowID int64, at time.Time, snapshotID string) orchestrator.Task {
	return &orchestrator.GenericOneoffTask{
		BaseTask: orchestrator.BaseTask{
			TaskName:   fmt.Sprintf("forget snapshot %q for plan %q in repo %q", snapshotID, planID, repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		OneoffTask: orchestrator.OneoffTask{
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationForget{},
			},
		},
		Do: func(ctx context.Context, st orchestrator.ScheduledTask, taskRunner orchestrator.TaskRunner) error {
			op := st.Op
			forgetOp := op.GetOperationForget()
			if forgetOp == nil {
				panic("forget task with non-forget operation")
			}

			if err := forgetSnapshotHelper(ctx, st, taskRunner, snapshotID); err != nil {
				taskRunner.ExecuteHooks([]v1.Hook_Condition{
					v1.Hook_CONDITION_ANY_ERROR,
				}, hook.HookVars{
					Error: err.Error(),
				})
			}
		},
	}
}

func forgetSnapshotHelper(ctx context.Context, st orchestrator.ScheduledTask, taskRunner orchestrator.TaskRunner, snapshotID string) error {
	t := st.Task

	repo, err := taskRunner.Orchestrator().GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	err = repo.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err)
	}

	if err := repo.ForgetSnapshot(ctx, snapshotID); err != nil {
		return fmt.Errorf("forget %q: %w", snapshotID, err)
	}

	taskRunner.Orchestrator().ScheduleTask(NewOneoffIndexSnapshotsTask(t.RepoID(), time.Now()), orchestrator.TaskPriorityIndexSnapshots)
	taskRunner.OpLog().Delete(st.Op.Id)
	st.Op = nil
	return err
}
