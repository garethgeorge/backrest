package orchestrator

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	v1 "github.com/garethgeorge/restora/gen/go/v1"
	"github.com/garethgeorge/restora/internal/protoutil"
	"github.com/garethgeorge/restora/pkg/restic"
	"go.uber.org/zap"
)

// RepoOrchestrator is responsible for managing a single repo.
type RepoOrchestrator struct {
	mu sync.Mutex

	l           *zap.Logger
	repoConfig  *v1.Repo
	repo        *restic.Repo
	initialized bool
}

// newRepoOrchestrator accepts a config and a repo that is configured with the properties of that config object.
func newRepoOrchestrator(repoConfig *v1.Repo, repo *restic.Repo) *RepoOrchestrator {
	return &RepoOrchestrator{
		repoConfig: repoConfig,
		repo:       repo,
		l:          zap.L().With(zap.String("repo", repoConfig.Id)),
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

	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.initialized {

		if err := r.repo.Init(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize repo: %w", err)
		}
		r.initialized = true
	}

	snapshots, err := r.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots for plan: %w", err)
	}

	r.l.Debug("got snapshots for plan", zap.String("repo", r.repoConfig.Id), zap.Int("count", len(snapshots)), zap.String("plan", plan.Id), zap.String("tag", tagForPlan(plan)))

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

	r.l.Debug("Backup completed", zap.String("repo", r.repoConfig.Id), zap.Duration("duration", time.Since(startTime)))
	return summary, nil
}

func (r *RepoOrchestrator) ListSnapshotFiles(ctx context.Context, snapshotId string, path string) ([]*v1.LsEntry, error) {
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

	l := r.l.With(zap.String("plan", plan.Id))

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

	policy := r.repoConfig.PrunePolicy

	var opts []restic.GenericOption
	if policy != nil {
		if policy.MaxUnusedBytes != 0 {
			opts = append(opts, restic.WithFlags("--max-unused", fmt.Sprintf("%v", policy.MaxUnusedBytes)))
		} else if policy.MaxUnusedPercent != 0 {
			opts = append(opts, restic.WithFlags("--max-unused", fmt.Sprintf("%v", policy.MaxUnusedPercent)))
		}
	} else {
		opts = append(opts, restic.WithFlags("--max-unused", "25%"))
	}

	r.l.Debug("Prune snapshots")
	err := r.repo.Prune(ctx, output, opts...)
	if err != nil {
		return fmt.Errorf("prune snapshots for repo %v: %w", r.repoConfig.Id, err)
	}
	return nil
}

func (r *RepoOrchestrator) Restore(ctx context.Context, snapshotId string, path string, target string, progressCallback func(event *v1.RestoreProgressEntry)) (*v1.RestoreProgressEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.l.Debug("Restore snapshot", zap.String("snapshot", snapshotId), zap.String("target", target))

	var opts []restic.GenericOption
	opts = append(opts, restic.WithFlags("--target", target))
	if path != "" {
		opts = append(opts, restic.WithFlags("--include", path))
	}

	summary, err := r.repo.Restore(ctx, snapshotId, func(event *restic.RestoreProgressEntry) {
		if progressCallback != nil {
			progressCallback(protoutil.RestoreProgressEntryToProto(event))
		}
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("restore snapshot %q for repo %v: %w", snapshotId, r.repoConfig.Id, err)
	}

	return protoutil.RestoreProgressEntryToProto(summary), nil
}

func (r *RepoOrchestrator) Unlock(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.l.Debug("Unlocking repo")
	r.repo.Unlock(ctx)

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
