package orchestrator

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/google/shlex"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// RepoOrchestrator is responsible for managing a single repo.
type RepoOrchestrator struct {
	mu sync.Mutex

	l           *zap.Logger
	repoConfig  *v1.Repo
	repo        *restic.Repo
	initialized bool
}

// NewRepoOrchestrator accepts a config and a repo that is configured with the properties of that config object.
func NewRepoOrchestrator(repoConfig *v1.Repo, resticPath string) (*RepoOrchestrator, error) {
	var opts []restic.GenericOption
	opts = append(opts, restic.WithEnviron())

	if len(repoConfig.GetEnv()) > 0 {
		opts = append(opts, restic.WithEnv(repoConfig.GetEnv()...))
	}

	if p := repoConfig.GetPassword(); p != "" {
		opts = append(opts, restic.WithEnv("RESTIC_PASSWORD="+p))
	}

	for _, f := range repoConfig.GetFlags() {
		args, err := shlex.Split(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse flag %q for repo %q: %w", f, repoConfig.Id, err)
		}
		opts = append(opts, restic.WithFlags(args...))
	}

	if env := repoConfig.GetEnv(); len(env) != 0 {
		opts = append(opts, restic.WithEnv(repoConfig.GetEnv()...))
	}

	repo := restic.NewRepo(resticPath, repoConfig.GetUri(), opts...)

	return &RepoOrchestrator{
		repoConfig: repoConfig,
		repo:       repo,
		l:          zap.L().With(zap.String("repo", repoConfig.Id)),
	}, nil
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
		if err := r.repo.Init(ctx, restic.WithEnviron()); err != nil {
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
	opts = append(opts, restic.WithBackupIExcludes(plan.Iexcludes...))
	opts = append(opts, restic.WithBackupTags(tagForPlan(plan)))
	for _, f := range plan.GetBackupFlags() {
		args, err := shlex.Split(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse backup flag %q for plan %q: %w", f, plan.Id, err)
		}
		opts = append(opts, restic.WithBackupFlags(args...))
	}

	if len(snapshots) > 0 {
		// TODO: design a test strategy to verify that the backup parent is used correctly.
		opts = append(opts, restic.WithBackupParent(snapshots[len(snapshots)-1].Id))
	}

	summary, err := r.repo.Backup(ctx, progressCallback, opts...)
	if err != nil {
		return summary, fmt.Errorf("failed to backup: %w", err)
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

	result, err := r.repo.Forget(
		ctx, protoutil.RetentionPolicyFromProto(plan.Retention),
		restic.WithFlags("--tag", tagForPlan(plan)), restic.WithFlags("--group-by", "tag"))
	if err != nil {
		return nil, fmt.Errorf("get snapshots for repo %v: %w", r.repoConfig.Id, err)
	}

	var forgotten []*v1.ResticSnapshot
	for _, snapshot := range result.Remove {
		snapshotProto := protoutil.SnapshotToProto(&snapshot)
		if err := protoutil.ValidateSnapshot(snapshotProto); err != nil {
			return nil, fmt.Errorf("snapshot validation failed: %w", err)
		}
		forgotten = append(forgotten, snapshotProto)
	}

	zap.L().Debug("Forgot snapshots", zap.String("plan", plan.Id), zap.Int("count", len(forgotten)), zap.Any("policy", policy))

	return forgotten, nil
}

func (r *RepoOrchestrator) ForgetSnapshot(ctx context.Context, snapshotId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.l.Debug("Forget snapshot with ID", zap.String("snapshot", snapshotId))
	return r.repo.ForgetSnapshot(ctx, snapshotId)
}

func (r *RepoOrchestrator) Prune(ctx context.Context, output io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	policy := r.repoConfig.PrunePolicy
	if policy == nil {
		policy = &v1.PrunePolicy{
			MaxUnusedPercent: 25,
		}
	}

	var opts []restic.GenericOption
	if policy.MaxUnusedBytes != 0 {
		opts = append(opts, restic.WithFlags("--max-unused", fmt.Sprintf("%vB", policy.MaxUnusedBytes)))
	} else if policy.MaxUnusedPercent != 0 {
		opts = append(opts, restic.WithFlags("--max-unused", fmt.Sprintf("%v%%", policy.MaxUnusedPercent)))
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

// UnlockIfAutoEnabled unlocks the repo if the auto unlock feature is enabled.
func (r *RepoOrchestrator) UnlockIfAutoEnabled(ctx context.Context) error {
	if !r.repoConfig.AutoUnlock {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	zap.L().Debug("AutoUnlocking repo", zap.String("repo", r.repoConfig.Id))

	return r.repo.Unlock(ctx)
}

func (r *RepoOrchestrator) Unlock(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.l.Debug("Unlocking repo")
	r.repo.Unlock(ctx)

	return nil
}

func (r *RepoOrchestrator) Stats(ctx context.Context) (*v1.RepoStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.l.Debug("Get Stats")
	stats, err := r.repo.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats for repo %v: %w", r.repoConfig.Id, err)
	}

	return protoutil.RepoStatsToProto(stats), nil
}

func (r *RepoOrchestrator) Config() *v1.Repo {
	if r == nil {
		return nil
	}
	return proto.Clone(r.repoConfig).(*v1.Repo)
}

func tagForPlan(plan *v1.Plan) string {
	return fmt.Sprintf("plan:%s", plan.Id)
}

func sortSnapshotsByTime(snapshots []*restic.Snapshot) {
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].UnixTimeMs() < snapshots[j].UnixTimeMs()
	})
}
