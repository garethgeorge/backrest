package tasks

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
)

type gcSettingsForType struct {
	maxAge  time.Duration
	keepMin int
	keepMax int
}

type groupByKey struct {
	RepoID     string
	RepoGUID   string
	PlanID     string
	InstanceID string
	Type       reflect.Type
}

const (
	gcStartupDelay = 1 * time.Second
	gcInterval     = 24 * time.Hour
)

var gcSettings = map[reflect.Type]gcSettingsForType{
	reflect.TypeOf(&v1.Operation_OperationStats{}): {
		maxAge:  365 * 24 * time.Hour,
		keepMin: 1,
		keepMax: 100,
	},
	reflect.TypeOf(&v1.Operation_OperationCheck{}): {
		maxAge:  365 * 24 * time.Hour,
		keepMin: 1,
		keepMax: 12,
	},
	reflect.TypeOf(&v1.Operation_OperationPrune{}): {
		maxAge:  365 * 24 * time.Hour,
		keepMin: 1,
		keepMax: 12,
	},
}

var defaultGcSettings = gcSettingsForType{
	maxAge:  30 * 24 * time.Hour,
	keepMin: 1,
	keepMax: 100,
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
	if err := t.gcOperations(runner); err != nil {
		return fmt.Errorf("collecting garbage: %w", err)
	}

	return nil
}

func (t *CollectGarbageTask) gcOperations(runner TaskRunner) error {
	// snapshotForgottenForFlow returns whether the snapshot associated with the flow is forgotten
	snapshotForgottenForFlow := make(map[int64]bool)
	if err := runner.QueryOperations(oplog.SelectAll, func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			snapshotForgottenForFlow[op.FlowId] = snapshotOp.OperationIndexSnapshot.Forgot
		}
		return nil
	}); err != nil {
		return fmt.Errorf("identifying forgotten snapshots: %w", err)
	}

	// cache known peer key ids, operations from unknown peers will be purged from the history as they can always be resync'd if the peer is readded.
	knownPeerKeyids := make(map[string]struct{})
	for _, peer := range runner.Config().GetMultihost().GetAuthorizedClients() {
		peerKeyid := peer.GetKeyid()
		if peerKeyid != "" {
			knownPeerKeyids[peerKeyid] = struct{}{}
		}
	}

	// keep track of IDs that are still valid and of the IDs that are being forgotten
	validIDs := make(map[int64]struct{})
	forgetIDs := []int64{}
	curTime := curTimeMillis()

	var deletedByMaxAge, deletedByMaxCount, deletedByForgottenSnapshot, deletedByUnknownPeerKeyid int
	deletedByType := make(map[string]int)
	stats := make(map[groupByKey]gcSettingsForType)

	if err := runner.QueryOperations(oplog.Query{}.SetReversed(true), func(op *v1.Operation) error {
		validIDs[op.Id] = struct{}{}

		// check if its a remote op, if it is forget it if its peer is forgotten. Elsewise use age.
		if op.OriginalInstanceKeyid != "" {
			_, ok := knownPeerKeyids[op.OriginalInstanceKeyid]
			if !ok {
				forgetIDs = append(forgetIDs, op.Id)
				deletedByUnknownPeerKeyid++
				deletedByType[reflect.TypeOf(op.Op).String()]++
				return nil
			}
		}

		forgot, ok := snapshotForgottenForFlow[op.FlowId]
		if ok {
			if forgot {
				// snapshot is forgotten; this operation is eligible for gc
				forgetIDs = append(forgetIDs, op.Id)
				deletedByForgottenSnapshot++
				deletedByType[reflect.TypeOf(op.Op).String()]++
			}
			return nil
		}

		key := groupByKey{
			RepoGUID:   op.RepoGuid,
			RepoID:     op.RepoId,
			PlanID:     op.PlanId,
			InstanceID: op.InstanceId,
			Type:       reflect.TypeOf(op.Op),
		}

		st, ok := stats[key]
		if !ok {
			gcSettings, ok := gcSettings[reflect.TypeOf(op.Op)]
			if !ok {
				st = defaultGcSettings
			} else {
				st = gcSettings
			}
		}

		st.keepMax--    // decrement the max retention, when this < 0 operation must be gc'd
		st.keepMin--    // decrement the min retention, when this < 0 we can start gc'ing
		stats[key] = st // update the stats

		if st.keepMin >= 0 {
			// can't delete if within min retention period
			return nil
		}
		if st.keepMax < 0 {
			// max retention reached; this operation must be gc'd.
			forgetIDs = append(forgetIDs, op.Id)
			deletedByMaxCount++
			deletedByType[key.Type.String()]++
		} else if curTime-op.UnixTimeStartMs > st.maxAge.Milliseconds() {
			// operation is old enough to be gc'd
			forgetIDs = append(forgetIDs, op.Id)
			deletedByMaxAge++
			deletedByType[key.Type.String()]++
		}

		return nil
	}); err != nil {
		return fmt.Errorf("identifying gc eligible operations: %w", err)
	}

	if err := runner.DeleteOperation(forgetIDs...); err != nil {
		return fmt.Errorf("removing gc eligible operations: %w", err)
	}
	for _, id := range forgetIDs { // update validIDs with respect to the just deleted operations
		delete(validIDs, id)
	}

	zap.L().Info("collecting garbage operations",
		zap.Int("operations_removed", len(forgetIDs)),
		zap.Int("removed_by_age", deletedByMaxAge),
		zap.Int("removed_by_limit", deletedByMaxCount),
		zap.Int("removed_by_snapshot_forgotten", deletedByForgottenSnapshot),
		zap.Any("removed_by_type", deletedByType),
		zap.Any("removed_by_unknown_peer_keyid", deletedByUnknownPeerKeyid))

	// cleaning up logstore
	toDelete := make(map[string]int64)
	if err := t.logstore.SelectAll(func(id string, parentID int64) {
		if parentID == 0 {
			return
		}
		if _, ok := validIDs[parentID]; !ok {
			toDelete[id] = parentID // this logstore entry is orphaned, mark it for deletion
		}
	}); err != nil {
		return fmt.Errorf("selecting all logstore entries: %w", err)
	}
	for id, parentID := range toDelete {
		// Confirm that the ID is invalid by trying to get it from the oplog
		if _, err := runner.GetOperation(parentID); !errors.Is(err, oplog.ErrNotExist) {
			if err != nil {
				zap.L().Error("getting operation for logstore entry", zap.String("id", id), zap.Int64("parent_id", parentID), zap.Error(err))
				continue
			}
			zap.L().Debug("logstore entry is still valid, skipping deletion", zap.String("id", id), zap.Error(err))
			continue
		}

		// The logstore entry is orphaned, delete it
		if err := t.logstore.Delete(id); err != nil {
			zap.L().Error("deleting logstore entry", zap.String("id", id), zap.Error(err))
		}
	}
	zap.L().Info("collecting garbage logs", zap.Any("logs_removed", len(toDelete)))

	return nil
}
