package tasks

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
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

	config := taskRunner.Config()

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

	// check if any migrations are required
	if migrated, err := tryMigrate(ctx, repo, config, snapshots); err != nil {
		return fmt.Errorf("migrate snapshots for repo %q: %w", t.RepoID(), err)
	} else if migrated {
		// Delete snapshot operations
		if err := oplog.Delete(maps.Values(currentIds)...); err != nil {
			return fmt.Errorf("delete prior indexed operations: %w", err)
		}

		snapshots, err = repo.Snapshots(ctx)
		if err != nil {
			return fmt.Errorf("get snapshots for repo %q: %w", t.RepoID(), err)
		}
		currentIds = nil
	}

	foundIds := make(map[string]struct{})

	// Index newly found operations
	startTime := time.Now()
	var indexOps []*v1.Operation
	for _, snapshot := range snapshots {
		if _, ok := currentIds[snapshot.Id]; ok {
			foundIds[snapshot.Id] = struct{}{}
			continue
		}

		snapshotProto := protoutil.SnapshotToProto(snapshot)
		flowID, err := FlowIDForSnapshotID(taskRunner.OpLog(), snapshot.Id)
		if err != nil {
			return fmt.Errorf("get flow ID for snapshot %q: %w", snapshot.Id, err)
		}
		planId := planForSnapshot(snapshotProto)
		instanceID := hostForSnapshot(snapshotProto)
		indexOps = append(indexOps, &v1.Operation{
			RepoId:          t.RepoID(),
			PlanId:          planId,
			FlowId:          flowID,
			InstanceId:      instanceID,
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
			return tag[len("plan:"):]
		}
	}
	return PlanForUnassociatedOperations
}

func hostForSnapshot(snapshot *v1.ResticSnapshot) string {
	for _, tag := range snapshot.Tags {
		if strings.HasPrefix(tag, "created-by:") {
			return tag[len("created-by:"):]
		}
	}
	return ""
}

// tryMigrate checks if the snapshots use the latest backrest tag set and migrates them if necessary.
func tryMigrate(ctx context.Context, repo *repo.RepoOrchestrator, config *v1.Config, snapshots []*restic.Snapshot) (bool, error) {
	if config.Instance == "" {
		zap.S().Warnf("Instance ID not set. Skipping migration.")
		return false, nil
	}

	planIDs := make(map[string]struct{})
	for _, plan := range config.Plans {
		planIDs[plan.Id] = struct{}{}
	}

	needsCreatedBy := []string{}
	for _, snapshot := range snapshots {
		// Check if snapshot is already tagged with `created-by:``
		if idx := slices.IndexFunc(snapshot.Tags, func(tag string) bool {
			return strings.HasPrefix(tag, "created-by:")
		}); idx != -1 {
			continue
		}
		// Check that snapshot is included in a plan for this instance. Backrest will not take ownership of snapshots belonging to it isn't aware of.
		if _, ok := planIDs[planForSnapshot(protoutil.SnapshotToProto(snapshot))]; !ok {
			continue
		}
		needsCreatedBy = append(needsCreatedBy, snapshot.Id)
	}

	if len(needsCreatedBy) == 0 {
		return false, nil
	}

	zap.S().Warnf("Found %v snapshots without created-by tag but included in a plan for this instance. Taking ownership and adding created-by tag.", len(needsCreatedBy))

	if err := repo.AddTags(ctx, needsCreatedBy, []string{fmt.Sprintf("created-by:%v", config.Instance)}); err != nil {
		return false, fmt.Errorf("add created-by tag to snapshots: %w", err)
	}

	return true, nil
}
