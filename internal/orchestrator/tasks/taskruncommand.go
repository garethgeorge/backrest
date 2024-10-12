package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func NewOneoffRunCommandTask(repoID string, planID string, flowID int64, at time.Time, command string) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType:   "forget_snapshot",
				TaskName:   fmt.Sprintf("run command in repo %q", repoID),
				TaskRepoID: repoID,
				TaskPlanID: planID,
			},
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationRunCommand{},
			},
		},
		Do: func(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
			op := st.Op
			rc := op.GetOperationRunCommand()
			if rc == nil {
				panic("forget task with non-forget operation")
			}

			return runCommandHelper(ctx, st, taskRunner, command)
		},
	}
}

func runCommandHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner, command string) error {
	t := st.Task

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	id, writer, err := taskRunner.LogrefWriter()
	if err != nil {
		return fmt.Errorf("get logref writer: %w", err)
	}
	st.Op.GetOperationRunCommand().OutputLogref = id
	if err := taskRunner.UpdateOperation(st.Op); err != nil {
		return fmt.Errorf("update operation: %w", err)
	}

	if err := repo.RunCommand(ctx, command, writer); err != nil {
		return fmt.Errorf("command %q: %w", command, err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close logref writer: %w", err)
	}

	return err
}
