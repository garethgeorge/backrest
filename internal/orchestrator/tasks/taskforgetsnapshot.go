package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func NewOneoffForgetSnapshotTask(repo *v1.Repo, planID string, flowID int64, at time.Time, snapshotID string) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType:   "forget_snapshot",
				TaskName:   fmt.Sprintf("forget snapshot %q for plan %q in repo %q", snapshotID, planID, repo.Id),
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

			return NotifyError(ctx, taskRunner, st.Task.Name(), forgetSnapshotHelper(ctx, st, taskRunner, snapshotID))
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

	taskRunner.ScheduleTask(NewOneoffIndexSnapshotsTask(t.Repo(), time.Now()), TaskPriorityIndexSnapshots)
	taskRunner.DeleteOperation(st.Op.Id)
	st.Op = nil
	return err
}
