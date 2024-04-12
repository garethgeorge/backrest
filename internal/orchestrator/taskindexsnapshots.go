package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.uber.org/zap"
)

// IndexSnapshotsTask tracks a forget operation.
type IndexSnapshotsTask struct {
	orchestrator *Orchestrator // owning orchestrator
	repoId       string
	at           *time.Time
}

var _ Task = &IndexSnapshotsTask{}

func NewOneoffIndexSnapshotsTask(orchestrator *Orchestrator, repoId string, at time.Time) *IndexSnapshotsTask {
	return &IndexSnapshotsTask{
		orchestrator: orchestrator,
		repoId:       repoId,
		at:           &at,
	}
}

func (t *IndexSnapshotsTask) Name() string {
	return fmt.Sprintf("index snapshots for plan %q", t.repoId)
}

func (t *IndexSnapshotsTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
	}
	return ret
}

func (t *IndexSnapshotsTask) Run(ctx context.Context) error {
	if err := indexSnapshotsHelper(ctx, t.orchestrator, t.repoId); err != nil {
		repo, _ := t.orchestrator.GetRepo(t.repoId)
		t.orchestrator.hookExecutor.ExecuteHooks(repo.Config(), nil, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:  t.Name(),
			Error: err.Error(),
		})
		return err
	}
	return nil
}

func (t *IndexSnapshotsTask) Cancel(withStatus v1.OperationStatus) error {
	return nil
}

func (t *IndexSnapshotsTask) OperationId() int64 {
	return 0
}

// indexSnapshotsHelper indexes all snapshots for a plan.
//   - If the snapshot is already indexed, it is skipped.
//   - If the snapshot is not indexed, an index snapshot operation with it's metadata is added.
//   - If an index snapshot operation is found for a snapshot that is not returned by the repo, it is marked as forgotten.
func indexSnapshotsHelper(ctx context.Context, orchestrator *Orchestrator, repoId string) error {
	repo, err := orchestrator.GetRepo(repoId)
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", repoId, err)
	}

	// collect all tracked snapshots for the plan.
	snapshots, err := repo.Snapshots(ctx)
	if err != nil {
		return fmt.Errorf("get snapshots for repo %q: %w", repoId, err)
	}

	// collect all current snapshot IDs.
	currentIds, err := indexCurrentSnapshotIdsForRepo(orchestrator.OpLog, repoId)
	if err != nil {
		return fmt.Errorf("get known snapshot IDs for repo %q: %w", repoId, err)
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
		planId := planForSnapshot(snapshotProto)
		indexOps = append(indexOps, &v1.Operation{
			RepoId:          repoId,
			PlanId:          planId,
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
	for id, opId := range currentIds {
		if _, ok := foundIds[id]; ok {
			// skip snapshots that were found.
			continue
		}

		// mark snapshot forgotten.
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
	zap.L().Debug("indexed snapshots",
		zap.String("repo", repoId),
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("alreadyIndexed", len(foundIds)),
		zap.Int("newlyAdded", len(indexOps)),
		zap.Int("markedForgotten", len(currentIds)-len(foundIds)),
	)

	return err
}

// returns a map of current (e.g. not forgotten) snapshot IDs for the plan.
func indexCurrentSnapshotIdsForRepo(log *oplog.OpLog, repoId string) (map[string]int64, error) {
	knownIds := make(map[string]int64)

	startTime := time.Now()
	if err := log.ForEachByRepo(repoId, indexutil.CollectAll(), func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			if snapshotOp.OperationIndexSnapshot == nil {
				return fmt.Errorf("operation %q has nil OperationIndexSnapshot, this shouldn't be possible", op.Id)
			}
			if !snapshotOp.OperationIndexSnapshot.Forgot {
				knownIds[snapshotOp.OperationIndexSnapshot.Snapshot.Id] = op.Id
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	zap.S().Debugf("indexed known snapshot IDs for repo %v in %v", repoId, time.Since(startTime))
	return knownIds, nil
}

func planForSnapshot(snapshot *v1.ResticSnapshot) string {
	for _, tag := range snapshot.Tags {
		if strings.HasPrefix(tag, "plan:") {
			return tag[5:]
		}
	}
	return PlanForUnassociatedOperations
}
