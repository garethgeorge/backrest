package orchestrator

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	v1 "github.com/garethgeorge/restora/gen/go/v1"
	"github.com/garethgeorge/restora/internal/oplog/indexutil"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

var (
	garbageCollectionInterval = time.Hour * 24 * 7 // every 7 days.
	garbageCollectionAge      = time.Hour * 24 * 7 // tasks remain in log 7 days after the oldest snapshot for the plan they belong to.
)

// CollectGarbageTask prunes old entries in the operation log.
type CollectGarbageTask struct {
	next         *time.Time
	orchestrator *Orchestrator
}

var _ Task = &CollectGarbageTask{}

func (t *CollectGarbageTask) Name() string {
	return "collect garbage"
}

// Next will schedule the task once at startup and then once every 7 days.
func (t *CollectGarbageTask) Next(now time.Time) *time.Time {
	ret := t.next
	next := now.Add(garbageCollectionInterval)
	t.next = &next
	if ret == nil {
		return &now
	}
	return ret
}

func (t *CollectGarbageTask) Run(ctx context.Context) error {
	var multierr error

	// collect garbage
	oldestSnapshotByPlan := make(map[string]int64)
	forgottenSnapshots := make(map[string]bool)

	if err := t.orchestrator.OpLog.ForAll(func(op *v1.Operation) error {
		snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot)
		if !ok {
			return nil
		}
		if snapshotOp.OperationIndexSnapshot.Forgot {
			forgottenSnapshots[snapshotOp.OperationIndexSnapshot.Snapshot.Id] = true
		}
		existingOldest := oldestSnapshotByPlan[op.PlanId]
		if existingOldest == 0 || op.UnixTimeStartMs < existingOldest {
			oldestSnapshotByPlan[op.PlanId] = op.UnixTimeStartMs
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to collect oldest snapshots: %w", err)
	}

	// collect all snapshots that are marked as forgotten
	idsToRemove := []int64{}
	for snapshotId := range forgottenSnapshots {
		if err := t.orchestrator.OpLog.ForEachBySnapshotId(snapshotId, indexutil.CollectAll(), func(op *v1.Operation) error {
			idsToRemove = append(idsToRemove, op.Id)
			return nil
		}); err != nil {
			multierr = multierror.Append(multierr, fmt.Errorf("failed to collect operations for snapshot %q: %w", snapshotId, err))
		}
	}

	// collect all snapshots that are older than the oldest snapshot for that plan
	if err := t.orchestrator.OpLog.ForAll(func(op *v1.Operation) error {
		oldestForPlan, ok := oldestSnapshotByPlan[op.PlanId]
		if !ok {
			return nil
		}
		if op.UnixTimeStartMs < oldestForPlan-garbageCollectionAge.Milliseconds() {
			idsToRemove = append(idsToRemove, op.Id)
		}
		return nil
	}); err != nil {
		multierr = multierror.Append(multierr, fmt.Errorf("failed to collect operations for garbage collection: %w", err))
	}

	// remove duplicates
	sort.Slice(idsToRemove, func(i, j int) bool {
		return idsToRemove[i] < idsToRemove[j]
	})
	slices.Compact(idsToRemove)

	if err := t.orchestrator.OpLog.DeleteAll(idsToRemove); err != nil {
		multierr = multierror.Append(multierr, fmt.Errorf("failed to delete operations for garbage collection: %w", err))
	}

	zap.L().Info("garbage collection complete", zap.Int("deleted", len(idsToRemove)))

	return multierr
}

func (t *CollectGarbageTask) Cancel(withStatus v1.OperationStatus) error {
	// not cancellable.
	return nil
}

func (t *CollectGarbageTask) OperationId() int64 {
	return 0
}
