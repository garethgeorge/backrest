package tasks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

func NewOneoffRestoreTask(repo *v1.Repo, planID string, flowID int64, at time.Time, snapshotID, path, target string) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType:   "restore",
				TaskName:   fmt.Sprintf("restore snapshot %q in repo %q", snapshotID, repo.Id),
				TaskRepo:   repo,
				TaskPlanID: planID,
			},
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				SnapshotId: snapshotID,
				Op: &v1.Operation_OperationRestore{
					OperationRestore: &v1.OperationRestore{
						Path:   path,
						Target: target,
					},
				},
			},
		},
		Do: func(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
			return NotifyError(ctx, taskRunner, st.Task.Name(), restoreHelper(ctx, st, taskRunner, snapshotID, path, target))
		},
	}
}

func restoreHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner, snapshotID, path, target string) error {
	t := st.Task
	op := st.Op

	if snapshotID == "" || path == "" || target == "" {
		return errors.New("snapshotID, path, and target are required")
	}

	restoreOp := st.Op.GetOperationRestore()
	if restoreOp == nil {
		return errors.New("operation is not a restore operation")
	}

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", t.RepoID(), err)
	}

	var sendWg sync.WaitGroup
	lastSent := time.Now() // debounce progress updates, these can endup being very frequent.
	summary, err := repo.Restore(ctx, snapshotID, path, target, func(entry *v1.RestoreProgressEntry) {
		sendWg.Wait()
		if time.Since(lastSent) < 1*time.Second {
			return
		}
		lastSent = time.Now()

		restoreOp.LastStatus = entry

		sendWg.Add(1)
		go func() {
			if err := taskRunner.UpdateOperation(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for restore: %v", err)
			}
			sendWg.Done()
		}()
	})

	if err != nil {
		return err
	}
	restoreOp.LastStatus = summary

	return nil
}
