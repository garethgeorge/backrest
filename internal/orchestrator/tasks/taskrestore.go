package tasks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"go.uber.org/zap"
)

func NewOneoffRestoreTask(repoID, planID string, flowID int64, at time.Time, snapshotID, path, target string) Task {
	return &GenericOneoffTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("restore snapshot %q in repo %q", snapshotID, repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		OneoffTask: OneoffTask{
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
			if err := restoreHelper(ctx, st, taskRunner, snapshotID, path, target); err != nil {
				taskRunner.ExecuteHooks([]v1.Hook_Condition{
					v1.Hook_CONDITION_ANY_ERROR,
				}, hook.HookVars{
					Task:  st.Task.Name(),
					Error: err.Error(),
				})
				return err
			}
			return nil
		},
	}
}

func restoreHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner, snapshotID, path, target string) error {
	t := st.Task
	oplog := taskRunner.OpLog()
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

		zap.S().Infof("restore progress: %v", entry)

		restoreOp.Status = entry

		sendWg.Add(1)
		go func() {
			if err := oplog.Update(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for restore: %v", err)
			}
			sendWg.Done()
		}()
	})

	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	restoreOp.Status = summary

	return nil
}
