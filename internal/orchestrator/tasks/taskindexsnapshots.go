package tasks

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
)

func NewOneoffIndexSnapshotsTask(repo *v1.Repo, at time.Time) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType: "index_snapshots",
				TaskName: fmt.Sprintf("index snapshots for repo %q", repo.Id),
				TaskRepo: repo,
			},
			RunAt:   at,
			ProtoOp: nil,
		},
		Do: func(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
			return NotifyError(ctx, taskRunner, st.Task.Name(), indexSnapshotsHelper(ctx, st, taskRunner))
		},
	}
}

// indexSnapshotsHelper indexes all snapshots for a plan.
//   - If the snapshot is already indexed, it is skipped.
//   - If the snapshot is not indexed, an index snapshot operation with it's metadata is added.
//   - If an index snapshot operation is found for a snapshot that is not returned by the repo, it is marked as forgotten.
func indexSnapshotsHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
	t := st.Task
	l := taskRunner.Logger(ctx)

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
	currentIds, err := indexCurrentSnapshotIdsForRepo(taskRunner, t.Repo().GetGuid())
	if err != nil {
		return fmt.Errorf("get known snapshot IDs for repo %q: %w", t.RepoID(), err)
	}

	foundIds := make(map[string]struct{})

	// Index newly found operations
	var indexOps []*v1.Operation
	for _, snapshot := range snapshots {
		if _, ok := currentIds[snapshot.Id]; ok {
			foundIds[snapshot.Id] = struct{}{}
			continue
		}

		snapshotProto := protoutil.SnapshotToProto(snapshot)
		flowID, err := FlowIDForSnapshotID(taskRunner, t.Repo().GetGuid(), snapshot.Id)
		if err != nil {
			return fmt.Errorf("get flow ID for snapshot %q: %w", snapshot.Id, err)
		}
		planId := planForSnapshot(snapshotProto)
		instanceID := instanceIDForSnapshot(snapshotProto)
		indexOps = append(indexOps, &v1.Operation{
			RepoId:          t.Repo().Id,
			RepoGuid:        t.Repo().Guid,
			PlanId:          planId,
			FlowId:          flowID,
			InstanceId:      instanceID,
			UnixTimeStartMs: snapshotProto.UnixTimeMs,
			UnixTimeEndMs:   snapshotProto.UnixTimeMs + snapshot.SnapshotSummary.DurationMs(),
			Status:          v1.OperationStatus_STATUS_SUCCESS,
			SnapshotId:      snapshotProto.Id,
			Op: &v1.Operation_OperationIndexSnapshot{
				OperationIndexSnapshot: &v1.OperationIndexSnapshot{
					Snapshot: snapshotProto,
				},
			},
		})
	}

	l.Sugar().Debugf("adding %v new snapshots to the oplog", len(indexOps))
	l.Sugar().Debugf("found %v snapshots already indexed", len(foundIds))

	if err := taskRunner.CreateOperation(indexOps...); err != nil {
		return fmt.Errorf("Create snapshot operations: %w", err)
	}

	// Mark missing operations as newly forgotten.
	for id, opId := range currentIds {
		if _, ok := foundIds[id]; ok {
			// skip snapshots that were found.
			continue
		}

		// mark snapshot forgotten.
		op, err := taskRunner.GetOperation(opId)
		if err != nil {
			// should only be possible in the case of a data race (e.g. operation was somehow deleted).
			return fmt.Errorf("get operation %v: %w", opId, err)
		}

		snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot)
		if !ok {
			return fmt.Errorf("operation %v is not an index snapshot operation", opId)
		}
		snapshotOp.OperationIndexSnapshot.Forgot = true

		if err := taskRunner.UpdateOperation(op); err != nil {
			return fmt.Errorf("mark index snapshot operation %v as forgotten: %w", opId, err)
		}
	}

	l.Sugar().Debugf("marked %v snapshots as forgotten", len(currentIds)-len(foundIds))

	return err
}

// returns a map of current (e.g. not forgotten) snapshot IDs for the plan.
func indexCurrentSnapshotIdsForRepo(taskRunner TaskRunner, repoGUID string) (map[string]int64, error) {
	knownIds := make(map[string]int64)

	startTime := time.Now()
	if err := taskRunner.QueryOperations(oplog.Query{}.
		SetRepoGUID(repoGUID),
		func(op *v1.Operation) error {
			if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
				if !snapshotOp.OperationIndexSnapshot.Forgot {
					knownIds[snapshotOp.OperationIndexSnapshot.Snapshot.Id] = op.Id
				}
			}
			return nil
		}); err != nil {
		return nil, err
	}
	zap.S().Debugf("found %v known snapshot IDs for repo %v in %v", len(knownIds), repoGUID, time.Since(startTime))
	return knownIds, nil
}

func planForSnapshot(snapshot *v1.ResticSnapshot) string {
	p := repo.PlanFromTags(snapshot.Tags)
	if p != "" {
		return p
	}
	return PlanForUnassociatedOperations
}

func instanceIDForSnapshot(snapshot *v1.ResticSnapshot) string {
	id := repo.InstanceIDFromTags(snapshot.Tags)
	if id != "" {
		return id
	}
	return InstanceIDForUnassociatedOperations
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
