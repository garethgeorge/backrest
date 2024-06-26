package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func NewOneoffForgetSnapshotTask(repoID, planID string, flowID int64, at time.Time, snapshotID string) Task {
	return &GenericOneoffTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("forget snapshot %q for plan %q in repo %q", snapshotID, planID, repoID),
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

			if err := forgetSnapshotHelper(ctx, st, taskRunner, snapshotID); err != nil {
				taskRunner.ExecuteHooks([]v1.Hook_Condition{
					v1.Hook_CONDITION_ANY_ERROR,
				}, HookVars{
					Error: err.Error(),
				})
				return err
			}
			return nil
		},
	}
}

func forgetSnapshotHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner, snapshotID string) error {
	t := st.Task

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
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

	taskRunner.ScheduleTask(NewOneoffIndexSnapshotsTask(t.RepoID(), time.Now()), TaskPriorityIndexSnapshots)
	taskRunner.OpLog().Delete(st.Op.Id)
	st.Op = nil
	return err
}
