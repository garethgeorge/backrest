package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
)

const (
	gcStartupDelay = 60 * time.Second
	gcInterval     = 24 * time.Hour
)

// gcAgeForOperation returns the age at which an operation is eligible for garbage collection.
func gcAgeForOperation(op *v1.Operation) time.Duration {
	switch op.Op.(type) {
	// stats, check, and prune operations are kept for a year
	case *v1.Operation_OperationStats, *v1.Operation_OperationCheck, *v1.Operation_OperationPrune:
		return 365 * 24 * time.Hour
	// all other operations are kept for 30 days
	default:
		return 30 * 24 * time.Hour
	}
}

type CollectGarbageTask struct {
	BaseTask
	firstRun bool
	logstore *logstore.LogStore
}

func NewCollectGarbageTask(logstore *logstore.LogStore) *CollectGarbageTask {
	return &CollectGarbageTask{
		BaseTask: BaseTask{
			TaskType: "collect_garbage",
			TaskName: "collect garbage",
		},
		logstore: logstore,
	}
}

var _ Task = &CollectGarbageTask{}

func (t *CollectGarbageTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if !t.firstRun {
		t.firstRun = true
		runAt := now.Add(gcStartupDelay)
		return ScheduledTask{
			Task:  t,
			RunAt: runAt,
		}, nil
	}

	runAt := now.Add(gcInterval)
	return ScheduledTask{
		Task:  t,
		RunAt: runAt,
	}, nil
}

func (t *CollectGarbageTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	oplog := runner.OpLog()

	if err := t.gcOperations(oplog); err != nil {
		return fmt.Errorf("collecting garbage: %w", err)
	}

	return nil
}

func (t *CollectGarbageTask) gcOperations(log *oplog.OpLog) error {
	// snapshotForgottenForFlow returns whether the snapshot associated with the flow is forgotten
	snapshotForgottenForFlow := make(map[int64]bool)
	if err := log.Query(oplog.SelectAll, func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			snapshotForgottenForFlow[op.FlowId] = snapshotOp.OperationIndexSnapshot.Forgot
		}
		return nil
	}); err != nil {
		return fmt.Errorf("identifying forgotten snapshots: %w", err)
	}

	validIDs := make(map[int64]struct{})
	forgetIDs := []int64{}
	curTime := curTimeMillis()
	if err := log.Query(oplog.SelectAll, func(op *v1.Operation) error {
		validIDs[op.Id] = struct{}{}

		forgot, ok := snapshotForgottenForFlow[op.FlowId]
		if !ok {
			// no snapshot associated with this flow; check if it's old enough to be gc'd
			maxAgeForOperation := gcAgeForOperation(op)
			if curTime-op.UnixTimeStartMs > maxAgeForOperation.Milliseconds() {
				forgetIDs = append(forgetIDs, op.Id)
			}
		} else if forgot {
			// snapshot is forgotten; this operation is eligible for gc
			forgetIDs = append(forgetIDs, op.Id)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("identifying gc eligible operations: %w", err)
	}

	if err := log.Delete(forgetIDs...); err != nil {
		return fmt.Errorf("removing gc eligible operations: %w", err)
	} else if len(forgetIDs) > 0 {
		for _, id := range forgetIDs {
			delete(validIDs, id)
		}
	}

	zap.L().Info("collecting garbage",
		zap.Any("operations_removed", len(forgetIDs)))

	// cleaning up logstore
	toDelete := []string{}
	if err := t.logstore.SelectAll(func(id string, parentID int64) {
		if parentID == 0 {
			return
		}
		if _, ok := validIDs[parentID]; !ok {
			toDelete = append(toDelete, id)
		}
	}); err != nil {
		return fmt.Errorf("selecting all logstore entries: %w", err)
	}
	for _, id := range toDelete {
		if err := t.logstore.Delete(id); err != nil {
			zap.L().Error("deleting logstore entry", zap.String("id", id), zap.Error(err))
		}
	}
	zap.L().Info("collecting garbage logs", zap.Any("logs_removed", len(toDelete)))

	return nil
}
