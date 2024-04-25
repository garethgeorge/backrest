package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
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
	orchestrator *Orchestrator // owning orchestrator
	firstRun     bool
}

var _ Task = &CollectGarbageTask{}

func (t *CollectGarbageTask) Name() string {
	return "collect garbage"
}

func (t *CollectGarbageTask) Next(now time.Time) *time.Time {
	if !t.firstRun {
		t.firstRun = true
		runAt := now.Add(gcStartupDelay)
		return &runAt
	}

	runAt := now.Add(gcInterval)
	return &runAt
}

func (t *CollectGarbageTask) Run(ctx context.Context) error {
	if err := t.gcOperations(); err != nil {
		return fmt.Errorf("collecting garbage: %w", err)
	}

	return nil
}

func (t *CollectGarbageTask) gcOperations() error {
	oplog := t.orchestrator.OpLog

	// pass 1: identify forgotten snapshots.
	snapshotIsForgotten := make(map[string]bool)
	if err := oplog.ForAll(func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			if snapshotOp.OperationIndexSnapshot.Forgot {
				snapshotIsForgotten[snapshotOp.OperationIndexSnapshot.Snapshot.Id] = true
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("identifying forgotten snapshots: %w", err)
	}

	// pass 2: identify operations that are gc eligible
	//  - any operation that has no snapshot associated with it
	//  - any operation that has a forgotten snapshot associated with it
	operationsByPlan := make(map[string][]gcOpInfo)
	if err := oplog.ForAll(func(op *v1.Operation) error {
		if op.SnapshotId == "" || snapshotIsForgotten[op.SnapshotId] {
			_, isStats := op.Op.(*v1.Operation_OperationStats)
			operationsByPlan[op.PlanId] = append(operationsByPlan[op.PlanId], gcOpInfo{
				id:        op.Id,
				timestamp: op.UnixTimeStartMs,
				isStats:   isStats,
			})
		}
		return nil
	}); err != nil {
		return fmt.Errorf("identifying gc eligible operations: %w", err)
	}

	var gcOps []int64
	curTime := curTimeMillis()
	for _, opInfos := range operationsByPlan {
		if len(opInfos) >= gcHistoryMaxCount {
			for _, opInfo := range opInfos[:len(opInfos)-gcHistoryMaxCount] {
				gcOps = append(gcOps, opInfo.id)
			}
			opInfos = opInfos[len(opInfos)-gcHistoryMaxCount:]
		}

		// check if each operation timestamp is old.
		for _, opInfo := range opInfos {
			maxAgeForType := gcHistoryAge.Milliseconds()
			if opInfo.isStats {
				maxAgeForType = gcHistoryStatsAge.Milliseconds()
			}
			if curTime-opInfo.timestamp > maxAgeForType {
				gcOps = append(gcOps, opInfo.id)
			}
		}
	}

	// pass 3: remove gc eligible operations
	if err := oplog.Delete(gcOps...); err != nil {
		return fmt.Errorf("removing gc eligible operations: %w", err)
	}

	zap.L().Info("collecting garbage",
		zap.Int("forgotten_snapshots", len(snapshotIsForgotten)),
		zap.Any("operations_removed", len(gcOps)))
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
