package orchestrator

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/protoutil"
	"github.com/garethgeorge/resticui/pkg/restic"
	"go.uber.org/zap"
)

// RepoOrchestrator is responsible for managing a single repo.
type RepoOrchestrator struct {
	mu sync.Mutex

	repoConfig *v1.Repo
	repo       *restic.Repo
}

// newRepoOrchestrator accepts a config and a repo that is configured with the properties of that config object.
func newRepoOrchestrator(repoConfig *v1.Repo, repo *restic.Repo) *RepoOrchestrator {
	return &RepoOrchestrator{
		repoConfig: repoConfig,
		repo:       repo,
	}
}

func (r *RepoOrchestrator) Snapshots(ctx context.Context) ([]*restic.Snapshot, error) {
	snapshots, err := r.repo.Snapshots(ctx)
	if err != nil {
		return nil, fmt.Errorf("get snapshots for repo %v: %w", r.repoConfig.Id, err)
	}
	sortSnapshotsByTime(snapshots)
	return snapshots, nil
}

func (r *RepoOrchestrator) SnapshotsForPlan(ctx context.Context, plan *v1.Plan) ([]*restic.Snapshot, error) {
	snapshots, err := r.repo.Snapshots(ctx, restic.WithFlags("--tag", tagForPlan(plan)))
	if err != nil {
		return nil, fmt.Errorf("get snapshots for plan %q: %w", plan.Id, err)
	}
	sortSnapshotsByTime(snapshots)
	return snapshots, nil
}

func (r *RepoOrchestrator) Backup(ctx context.Context, plan *v1.Plan, progressCallback func(event *restic.BackupProgressEntry)) (*restic.BackupProgressEntry, error) {
	zap.L().Debug("repo orchestrator starting backup", zap.String("repo", r.repoConfig.Id))

	if err := r.repo.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

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
		opts = append(opts, restic.WithBackupParent(snapshots[len(snapshots)-1].Id))
	}

	summary, err := r.repo.Backup(ctx, progressCallback, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to backup: %w", err)
	}

	zap.L().Debug("Backup completed", zap.String("repo", r.repoConfig.Id), zap.Duration("duration", time.Since(startTime)))
	return summary, nil
}

func (r *RepoOrchestrator) ListSnapshotFiles(ctx context.Context, snapshotId string, path string) ([]*v1.LsEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, entries, err := r.repo.ListDirectory(ctx, snapshotId, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot files: %w", err)
	}

	lsEnts := make([]*v1.LsEntry, 0, len(entries))
	for _, entry := range entries {
		lsEnts = append(lsEnts, entry.ToProto())
	}

	return lsEnts, nil
}

func (r *RepoOrchestrator) Forget(ctx context.Context, plan *v1.Plan) ([]*v1.ResticSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	policy := plan.Retention
	if policy == nil {
		return nil, fmt.Errorf("plan %q has no retention policy", plan.Id)
	}

	l := zap.L().With(zap.String("repo", r.repoConfig.Id), zap.String("plan", plan.Id))

	l.Debug("Forget snapshots", zap.Any("policy", policy))
	result, err := r.repo.Forget(
		ctx, protoutil.RetentionPolicyFromProto(plan.Retention),
		restic.WithFlags("--tag", tagForPlan(plan)), restic.WithFlags("--group-by", "tag"))
	if err != nil {
		return nil, fmt.Errorf("get snapshots for repo %v: %w", r.repoConfig.Id, err)
	}
	l.Debug("Forget result", zap.Any("result", result))

	var forgotten []*v1.ResticSnapshot
	for _, snapshot := range result.Remove {
		snapshotProto := protoutil.SnapshotToProto(&snapshot)
		if err := protoutil.ValidateSnapshot(snapshotProto); err != nil {
			return nil, fmt.Errorf("snapshot validation failed: %w", err)
		}
		forgotten = append(forgotten, snapshotProto)
	}

	return forgotten, nil
}

func (r *RepoOrchestrator) Prune(ctx context.Context, output io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	l := zap.L().With(zap.String("repo", r.repoConfig.Id))

	l.Debug("Prune snapshots")
	err := r.repo.Prune(ctx, output)
	if err != nil {
		return fmt.Errorf("prune snapshots for repo %v: %w", r.repoConfig.Id, err)
	}
	return nil
}

func tagForPlan(plan *v1.Plan) string {
	return fmt.Sprintf("plan:%s", plan.Id)
}

func sortSnapshotsByTime(snapshots []*restic.Snapshot) {
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].UnixTimeMs() < snapshots[j].UnixTimeMs()
	})
}
