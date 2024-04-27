package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
)

const (
	gcStartupDelay = 5 * time.Second
	gcInterval     = 24 * time.Hour
	// keep operations that are eligible for gc for 30 days OR up to a limit of 100 for any one plan.
	// an operation is eligible for gc if:
	// - it has no snapshot associated with it
	// - it has a forgotten snapshot associated with it
	gcHistoryAge      = 30 * 24 * time.Hour
	gcHistoryMaxCount = 1000
	// keep stats operations for 1 year (they're small and useful for long term trends)
	gcHistoryStatsAge = 365 * 24 * time.Hour
)

type CollectGarbageTask struct {
	BaseTask
	firstRun bool
}

var _ Task = &CollectGarbageTask{}

func (t *CollectGarbageTask) Next(now time.Time, runner TaskRunner) ScheduledTask {
	if !t.firstRun {
		t.firstRun = true
		runAt := now.Add(gcStartupDelay)
		return ScheduledTask{
			Task:  t,
			RunAt: runAt,
		}
	}

	runAt := now.Add(gcInterval)
	return ScheduledTask{
		Task:  t,
		RunAt: runAt,
	}
}

func (t *CollectGarbageTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	oplog := runner.OpLog()

	if err := t.gcOperations(oplog); err != nil {
		return fmt.Errorf("collecting garbage: %w", err)
	}

	return nil
}

func (t *CollectGarbageTask) gcOperations(oplog *oplog.OpLog) error {
	// snapshotForgottenForFlow returns whether the snapshot associated with the flow is forgotten
	snapshotForgottenForFlow := make(map[int64]bool)
	if err := oplog.ForAll(func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			snapshotForgottenForFlow[op.FlowId] = snapshotOp.OperationIndexSnapshot.Forgot
		}
		return nil
	}); err != nil {
		return fmt.Errorf("identifying forgotten snapshots: %w", err)
	}

	forgetIDs := []int64{}
	curTime := curTimeMillis()
	if err := oplog.ForAll(func(op *v1.Operation) error {
		forgot, ok := snapshotForgottenForFlow[op.FlowId]
		if !ok {
			// no snapshot associated with this flow; check if it's old enough to be gc'd
			maxAgeForType := gcHistoryAge.Milliseconds()
			if _, isStats := op.Op.(*v1.Operation_OperationStats); isStats {
				maxAgeForType = gcHistoryStatsAge.Milliseconds()
			}
			if curTime-op.UnixTimeStartMs > maxAgeForType {
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

	if err := oplog.Delete(forgetIDs...); err != nil {
		return fmt.Errorf("removing gc eligible operations: %w", err)
	}

	zap.L().Info("collecting garbage",
		zap.Any("operations_removed", len(forgetIDs)))
	return nil
}

func (t *CollectGarbageTask) Cancel(withStatus v1.OperationStatus) error {
	return nil
}

func (t *CollectGarbageTask) OperationId() int64 {
	return 0
}

type gcOpInfo struct {
	id        int64 // operation ID
	timestamp int64 // unix time milliseconds
	isStats   bool  // true if this is a stats operation
}
