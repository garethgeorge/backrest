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

	snapshotsMu sync.Mutex // enable very fast snapshot access IF no update is required.
	snapshotsAge time.Time
	snapshots []*restic.Snapshot
}

func newRepoOrchestrator(repoConfig *v1.Repo, repo *restic.Repo) *RepoOrchestrator {
	return &RepoOrchestrator{
		repoConfig: repoConfig,
		repo: repo,
	}
}

func (r *RepoOrchestrator) updateSnapshotsIfNeeded(ctx context.Context) error {
	r.snapshotsMu.Lock()
	defer r.snapshotsMu.Unlock()
	if time.Since(r.snapshotsAge) > 10 * time.Minute {
		r.snapshots = nil
	}

	if r.snapshots != nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshots, err := r.repo.Snapshots(ctx, restic.WithPropagatedEnvVars(restic.EnvToPropagate...))
	if err != nil {
		return fmt.Errorf("failed to update snapshots: %w", err)
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].Time < snapshots[j].Time
	})
	r.snapshots = snapshots

	return nil
}

func (r *RepoOrchestrator) Snapshots(ctx context.Context) ([]*restic.Snapshot, error) {
	if err := r.updateSnapshotsIfNeeded(ctx); err != nil {
		return nil, err
	}

	r.snapshotsMu.Lock()
	defer r.snapshotsMu.Unlock()
	return r.snapshots, nil
}

func (r *RepoOrchestrator) SnapshotsForPlan(ctx context.Context, plan *v1.Plan) ([]*restic.Snapshot, error) {
	if err := r.updateSnapshotsIfNeeded(ctx); err != nil {
		return nil, err 
	}

	r.snapshotsMu.Lock()
	defer r.snapshotsMu.Unlock()
	return filterSnapshotsForPlan(r.snapshots, plan), nil
}

func (r *RepoOrchestrator) Backup(ctx context.Context, plan *v1.Plan, progressCallback func(event *restic.BackupProgressEntry)) (*restic.BackupProgressEntry, error) {
	snapshots, err := r.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots for plan: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var opts []restic.BackupOption
	opts = append(opts, restic.WithBackupPaths(plan.Paths...))
	opts = append(opts, restic.WithBackupExcludes(plan.Excludes...))
	opts = append(opts, restic.WithBackupTags(tagForPlan(plan)))

	if len(snapshots) > 0 {
		// TODO: design a test strategy to verify that the backup parent is used correctly.
		opts = append(opts, restic.WithBackupParent(snapshots[len(snapshots) - 1].Id))
	}

	summary, err := r.repo.Backup(ctx, progressCallback, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to backup: %w", err)
	}
	return summary, nil
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
