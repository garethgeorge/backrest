package tasks

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

func NewOneoffIndexSnapshotsTask(repoID string, at time.Time) Task {
	return &GenericOneoffTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("index snapshots for repo %q", repoID),
			TaskRepoID: repoID,
		},
		OneoffTask: OneoffTask{
			RunAt:   at,
			ProtoOp: nil,
		},
		Do: func(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
			if err := indexSnapshotsHelper(ctx, st, taskRunner); err != nil {
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

// indexSnapshotsHelper indexes all snapshots for a plan.
//   - If the snapshot is already indexed, it is skipped.
//   - If the snapshot is not indexed, an index snapshot operation with it's metadata is added.
//   - If an index snapshot operation is found for a snapshot that is not returned by the repo, it is marked as forgotten.
func indexSnapshotsHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
	t := st.Task
	oplog := taskRunner.OpLog()

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", t.RepoID(), err)
	}

	// collect all tracked snapshots for the plan.
	snapshots, err := repo.Snapshots(ctx)
	if err != nil {
		return fmt.Errorf("get snapshots for repo %q: %w", t.RepoID(), err)
	}

	// collect all current snapshot IDs.
	currentIds, err := indexCurrentSnapshotIdsForRepo(taskRunner.OpLog(), t.RepoID())
	if err != nil {
		return fmt.Errorf("get known snapshot IDs for repo %q: %w", t.RepoID(), err)
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
		flowID, err := FlowIDForSnapshotID(taskRunner.OpLog(), snapshot.Id)
		if err != nil {
			return fmt.Errorf("get flow ID for snapshot %q: %w", snapshot.Id, err)
		}
		planId := planForSnapshot(snapshotProto)
		indexOps = append(indexOps, &v1.Operation{
			RepoId:          t.RepoID(),
			PlanId:          planId,
			FlowId:          flowID,
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

	if err := taskRunner.OpLog().BulkAdd(indexOps); err != nil {
		return fmt.Errorf("BulkAdd snapshot operations: %w", err)
	}

	// Mark missing operations as newly forgotten.
	for id, opId := range currentIds {
		if _, ok := foundIds[id]; ok {
			// skip snapshots that were found.
			continue
		}

		// mark snapshot forgotten.
		op, err := oplog.Get(opId)
		if err != nil {
			// should only be possible in the case of a data race (e.g. operation was somehow deleted).
			return fmt.Errorf("get operation %v: %w", opId, err)
		}

		snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot)
		if !ok {
			return fmt.Errorf("operation %v is not an index snapshot operation", opId)
		}
		snapshotOp.OperationIndexSnapshot.Forgot = true

		if err := oplog.Update(op); err != nil {
			return fmt.Errorf("mark index snapshot operation %v as forgotten: %w", opId, err)
		}
	}

	// Print stats at the end of indexing.
	zap.L().Debug("indexed snapshots",
		zap.String("repo", t.RepoID()),
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
