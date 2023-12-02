package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/oplog"
	"github.com/garethgeorge/resticui/internal/oplog/indexutil"
	"github.com/garethgeorge/resticui/internal/protoutil"
	"go.uber.org/zap"
)

// ForgetTask tracks a forget operation.
type IndexSnapshotsTask struct {
	orchestrator *Orchestrator // owning orchestrator
	plan         *v1.Plan
	at           *time.Time
}

var _ Task = &ForgetTask{}

func NewOneofIndexSnapshotsTask(orchestrator *Orchestrator, plan *v1.Plan, at time.Time) *IndexSnapshotsTask {
	return &IndexSnapshotsTask{
		orchestrator: orchestrator,
		plan:         plan,
		at:           &at,
	}
}

func (t *IndexSnapshotsTask) Name() string {
	return fmt.Sprintf("index snapshots for plan %q", t.plan.Id)
}

func (t *IndexSnapshotsTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
	}
	return ret
}

func (t *IndexSnapshotsTask) Run(ctx context.Context) error {
	return indexSnapshotsHelper(ctx, t.orchestrator, t.plan)
}

func (t *IndexSnapshotsTask) Cancel(withStatus v1.OperationStatus) error {
	return nil
}

// indexSnapshotsHelper indexes all snapshots for a plan.
//   - If the snapshot is already indexed, it is skipped.
//   - If the snapshot is not indexed, an index snapshot operation with it's metadata is added.
//   - If an index snapshot operation is found for a snapshot that is not returned by the repo, it is marked as forgotten.
func indexSnapshotsHelper(ctx context.Context, orchestrator *Orchestrator, plan *v1.Plan) error {
	repo, err := orchestrator.GetRepo(plan.Repo)
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", plan.Repo, err)
	}

	// collect all tracked snapshots for the plan.
	snapshots, err := repo.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("get snapshots for plan %q: %w", plan.Id, err)
	}

	// collect all current snapshot IDs.
	currentIds, err := indexCurrentSnapshotIdsForPlan(orchestrator.OpLog, plan.Id)
	if err != nil {
		return fmt.Errorf("get known snapshot IDs for plan %q: %w", plan.Id, err)
	}

	foundIds := make(map[string]bool)

	// Index newly found operations
	startTime := time.Now()
	var indexOps []*v1.Operation
	for _, snapshot := range snapshots {
		if _, ok := currentIds[snapshot.Id]; ok {
			foundIds[snapshot.Id] = true
			continue
		}

		snapshotProto := protoutil.SnapshotToProto(snapshot)
		indexOps = append(indexOps, &v1.Operation{
			RepoId:          plan.Repo,
			PlanId:          plan.Id,
			UnixTimeStartMs: snapshotProto.UnixTimeMs,
			UnixTimeEndMs:   snapshotProto.UnixTimeMs,
			Status:          v1.OperationStatus_STATUS_SUCCESS,
			SnapshotId:      snapshotProto.Id,
			Op: &v1.Operation_OperationIndexSnapshot{
				OperationIndexSnapshot: &v1.OperationIndexSnapshot{
					Snapshot: snapshotProto,
				},
			},
		})
	}

	if err := orchestrator.OpLog.BulkAdd(indexOps); err != nil {
		return fmt.Errorf("BulkAdd snapshot operations: %w", err)
	}

	// Mark missing operations as newly forgotten.
	var forgetIds []int64
	for id, opId := range currentIds {
		if _, ok := foundIds[id]; !ok {
			forgetIds = append(forgetIds, opId)
		}
	}

	for _, opId := range forgetIds {
		op, err := orchestrator.OpLog.Get(opId)
		if err != nil {
			// should only be possible in the case of a data race (e.g. operation was somehow deleted).
			return fmt.Errorf("get operation %v: %w", opId, err)
		}

		snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot)
		if !ok {
			return fmt.Errorf("operation %v is not an index snapshot operation", opId)
		}
		snapshotOp.OperationIndexSnapshot.Forgot = true

		if err := orchestrator.OpLog.Update(op); err != nil {
			return fmt.Errorf("mark index snapshot operation %v as forgotten: %w", opId, err)
		}
	}

	// Print stats at the end of indexing.
	zap.L().Debug("Indexed snapshots",
		zap.String("plan", plan.Id),
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("alreadyIndexed", len(foundIds)),
		zap.Int("newlyAdded", len(indexOps)),
		zap.Int("markedForgotten", len(currentIds)-len(foundIds)),
	)

	return err
}

// returns a map of current (e.g. not forgotten) snapshot IDs for the plan.
func indexCurrentSnapshotIdsForPlan(log *oplog.OpLog, planId string) (map[string]int64, error) {
	knownIds := make(map[string]int64)

	startTime := time.Now()
	if err := log.ForEachByPlan(planId, indexutil.CollectAll(), func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			if snapshotOp.OperationIndexSnapshot == nil {
				return fmt.Errorf("operation %q has nil OperationIndexSnapshot, this shouldn't be possible.", op.Id)
			}
			if !snapshotOp.OperationIndexSnapshot.Forgot {
				knownIds[snapshotOp.OperationIndexSnapshot.Snapshot.Id] = op.Id
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	zap.S().Debugf("Indexed known (and not forgotten) snapshot IDs for plan %v in %v", planId, time.Since(startTime))
	return knownIds, nil
}
