package orchestrator

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/pkg/restic"
)

// RepoOrchestrator is responsible for managing a single repo.
type RepoOrchestrator struct {
	mu sync.Mutex

	repoConfig *v1.Repo
	repo *restic.Repo

	snapshotsAge time.Time
	snapshots []*restic.Snapshot
}

func (r *RepoOrchestrator) updateSnapshotsIfNeeded(ctx context.Context) error {
	if time.Since(r.snapshotsAge) > 10 * time.Minute {
		r.snapshots = nil
	}

	if r.snapshots != nil {
		return nil
	}

	snapshots, err := r.repo.Snapshots(ctx, restic.WithPropagatedEnvVars(restic.EnvToPropagate...))
	if err != nil {
		return fmt.Errorf("failed to update snapshots: %w", err)
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].Time > snapshots[j].Time
	})

	r.snapshots = snapshots

	return nil
}

func (r *RepoOrchestrator) Snapshots(ctx context.Context) ([]*restic.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.updateSnapshotsIfNeeded(ctx); err != nil {
		return nil, err
	}

	return r.snapshots, nil
}

func (r *RepoOrchestrator) SnapshotsForPlan(ctx context.Context, plan *v1.Plan) ([]*restic.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.updateSnapshotsIfNeeded(ctx); err != nil {
		return nil, err 
	}

	return filterSnapshotsForPlan(r.snapshots, plan), nil
}

func (r *RepoOrchestrator) Backup(ctx context.Context, plan *v1.Plan, progressCallback func(event *restic.BackupProgressEntry)) error {
	snapshots, err := r.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return err
	}

	var opts []restic.BackupOption
	opts = append(opts, restic.WithBackupPaths(plan.Paths...))
	opts = append(opts, restic.WithBackupExcludes(plan.Excludes...))
	opts = append(opts, restic.WithBackupTags(tagForPlan(plan)))

	if len(snapshots) > 0 {
		// TODO: design a test strategy to verify that the backup parent is used correctly.
		opts = append(opts, restic.WithBackupParent(snapshots[len(snapshots) - 1].Id))
	}

	panic("not yet implemented")
}

func filterSnapshotsForPlan(snapshots []*restic.Snapshot, plan *v1.Plan) []*restic.Snapshot {
	wantTag := tagForPlan(plan)
	var filtered []*restic.Snapshot
	for _, snapshot := range snapshots {
		if snapshot.Tags == nil {
			continue
		}

		if slices.Contains(snapshot.Tags, wantTag) {
			filtered = append(filtered, snapshot)
		}
	}

	return filtered
}

func tagForPlan(plan *v1.Plan) string {
	return fmt.Sprintf("plan:%s", plan.Id)
}
