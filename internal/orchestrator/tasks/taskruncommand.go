package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/ioutil"
)

var DefaultCommandOutputSizeLimit uint64 = 2_000_000 // 2MB

func NewOneoffRunCommandTask(repo *v1.Repo, planID string, flowID int64, at time.Time, command string) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType:   "run_command",
				TaskName:   fmt.Sprintf("run command in repo %q", repo.Id),
				TaskRepo:   repo,
				TaskPlanID: planID,
			},
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationRunCommand{
					OperationRunCommand: &v1.OperationRunCommand{
						Command: command,
					},
				},
			},
		},
		Do: func(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
			op := st.Op
			rc := op.GetOperationRunCommand()
			if rc == nil {
				panic("run command task with non-forget operation")
			}

			return NotifyError(ctx, taskRunner, st.Task.Name(), runCommandHelper(ctx, st, taskRunner, command))
		},
	}
}

func runCommandHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner, command string) error {
	t := st.Task
	runCmdOp := st.Op.GetOperationRunCommand()

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	id, writer, err := taskRunner.LogrefWriter()
	if err != nil {
		return fmt.Errorf("get logref writer: %w", err)
	}
	defer writer.Close()
	sizeWriter := &ioutil.SizeLimitedWriter{
		SizeTrackingWriter: ioutil.SizeTrackingWriter{Writer: writer},
		Limit:              DefaultCommandOutputSizeLimit, // 2 MB max output size
	}
	defer func() {
		size := sizeWriter.Size()
		runCmdOp.OutputSizeBytes = int64(size)
	}()

	runCmdOp.OutputLogref = id
	if err := taskRunner.UpdateOperation(st.Op); err != nil {
		return fmt.Errorf("update operation: %w", err)
	}

	if err := repo.RunCommand(ctx, command, sizeWriter); err != nil {
		return fmt.Errorf("command %q: %w", command, err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close logref writer: %w", err)
	}

	return err
}
