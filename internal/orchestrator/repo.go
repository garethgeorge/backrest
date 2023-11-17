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
	"go.uber.org/zap"
)

// RepoOrchestrator is responsible for managing a single repo.
type RepoOrchestrator struct {
	mu sync.Mutex

	repoConfig *v1.Repo
	repo *restic.Repo
}

func newRepoOrchestrator(repoConfig *v1.Repo, repo *restic.Repo) *RepoOrchestrator {
	return &RepoOrchestrator{
		repoConfig: repoConfig,
		repo: repo,
	}
}

func (r *RepoOrchestrator) Snapshots(ctx context.Context) ([]*restic.Snapshot, error) {
	snapshots, err := r.repo.Snapshots(ctx, restic.WithPropagatedEnvVars(restic.EnvToPropagate...), restic.WithFlags("--latest", "1000"))
	if err != nil {
		return nil, fmt.Errorf("restic.Snapshots: %w", err)
	}

	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].UnixTimeMs() < snapshots[j].UnixTimeMs()
	})

	return snapshots, nil
}

func (r *RepoOrchestrator) SnapshotsForPlan(ctx context.Context, plan *v1.Plan) ([]*restic.Snapshot, error) {
	snapshots, err := r.Snapshots(ctx)
	if err != nil {
		return nil, err 
	}
	
	return filterSnapshotsForPlan(snapshots, plan), nil
}

func (r *RepoOrchestrator) Backup(ctx context.Context, plan *v1.Plan, progressCallback func(event *restic.BackupProgressEntry)) (*restic.BackupProgressEntry, error) {
	zap.L().Debug("repo orchestrator starting backup", zap.String("repo", r.repoConfig.Id))
	snapshots, err := r.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots for plan: %w", err)
	}

	zap.L().Debug("got snapshots for plan", zap.String("repo", r.repoConfig.Id), zap.Int("count", len(snapshots)), zap.String("plan", plan.Id), zap.String("tag", tagForPlan(plan)))

	r.mu.Lock()
	defer r.mu.Unlock()

	startTime := time.Now()

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

	zap.L().Debug("Backup completed", zap.String("repo", r.repoConfig.Id), zap.Duration("duration", time.Since(startTime)))
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
