package repo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/google/shlex"
	"go.uber.org/zap"
)

// RepoOrchestrator implements higher level repository operations on top of
// the restic package. It can be thought of as a controller for a repo.
type RepoOrchestrator struct {
	mu sync.Mutex

	config     *v1.Config
	repoConfig *v1.Repo
	repo       *restic.Repo
}

// NewRepoOrchestrator accepts a config and a repo that is configured with the properties of that config object.
func NewRepoOrchestrator(config *v1.Config, repoConfig *v1.Repo, resticPath string) (*RepoOrchestrator, error) {
	if config.Instance == "" {
		return nil, errors.New("instance is a required field in the backrest config")
	}

	var opts []restic.GenericOption
	if p := repoConfig.GetPassword(); p != "" {
		opts = append(opts, restic.WithEnv("RESTIC_PASSWORD="+p))
	}

	opts = append(opts, restic.WithEnviron())

	if env := repoConfig.GetEnv(); len(env) != 0 {
		for _, e := range env {
			opts = append(opts, restic.WithEnv(ExpandEnv(e)))
		}
	}

	for _, f := range repoConfig.GetFlags() {
		args, err := shlex.Split(ExpandEnv(f))
		if err != nil {
			return nil, fmt.Errorf("parse flag %q for repo %q: %w", f, repoConfig.Id, err)
		}
		opts = append(opts, restic.WithFlags(args...))
	}

	// Resolve command prefix
	if extraOpts, err := resolveCommandPrefix(repoConfig.GetCommandPrefix()); err != nil {
		return nil, fmt.Errorf("resolve command prefix: %w", err)
	} else {
		opts = append(opts, extraOpts...)
	}

	// Add BatchMode=yes to sftp.args if it's not already set.
	if slices.IndexFunc(repoConfig.GetFlags(), func(a string) bool {
		return strings.Contains(a, "sftp.args")
	}) == -1 {
		opts = append(opts, restic.WithFlags("-o", "sftp.args=-oBatchMode=yes"))
	}

	repo := restic.NewRepo(resticPath, repoConfig.GetUri(), opts...)

	return &RepoOrchestrator{
		config:     config,
		repoConfig: repoConfig,
		repo:       repo,
	}, nil
}

func (r *RepoOrchestrator) logger(ctx context.Context) *zap.Logger {
	return logging.Logger(ctx, "[repo-manager] ").With(zap.String("repo", r.repoConfig.Id))
}

func (r *RepoOrchestrator) Exists(ctx context.Context) error {
	return r.repo.Exists(ctx)
}

func (r *RepoOrchestrator) Init(ctx context.Context) error {
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	return r.repo.Init(ctx)
}

func (r *RepoOrchestrator) Snapshots(ctx context.Context) ([]*restic.Snapshot, error) {
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	snapshots, err := r.repo.Snapshots(ctx)
	if err != nil {
		return nil, fmt.Errorf("get snapshots for repo %v: %w", r.repoConfig.Id, err)
	}
	sortSnapshotsByTime(snapshots)
	return snapshots, nil
}

func (r *RepoOrchestrator) SnapshotsForPlan(ctx context.Context, plan *v1.Plan) ([]*restic.Snapshot, error) {
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	tags := []string{TagForPlan(plan.Id)}
	if r.config.Instance != "" {
		tags = append(tags, TagForInstance(r.config.Instance))
	}

	snapshots, err := r.repo.Snapshots(ctx, restic.WithFlags("--tag", strings.Join(tags, ",")))
	if err != nil {
		return nil, fmt.Errorf("get snapshots for plan %q: %w", plan.Id, err)
	}
	sortSnapshotsByTime(snapshots)
	return snapshots, nil
}

func (r *RepoOrchestrator) Backup(ctx context.Context, plan *v1.Plan, progressCallback func(event *restic.BackupProgressEntry)) (*restic.BackupProgressEntry, error) {
	l := r.logger(ctx)
	l.Debug("repo orchestrator starting backup", zap.String("repo", r.repoConfig.Id))

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshots, err := r.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots for plan: %w", err)
	}

	l.Debug("got snapshots for plan", zap.String("repo", r.repoConfig.Id), zap.Int("count", len(snapshots)), zap.String("plan", plan.Id), zap.String("tag", TagForPlan(plan.Id)))

	startTime := time.Now()

	var opts []restic.GenericOption
	opts = append(opts, restic.WithFlags(
		"--exclude-caches",
		"--tag", TagForPlan(plan.Id),
	))

	if r.config.Instance != "" {
		opts = append(opts, restic.WithFlags("--tag", TagForInstance(r.config.Instance)))
	} else {
		zap.L().Warn("Creating a backup without an 'instance' tag as no value is set in the config. In a future backrest release this will be an error.")
	}

	for _, exclude := range plan.Excludes {
		opts = append(opts, restic.WithFlags("--exclude", exclude))
	}
	for _, iexclude := range plan.Iexcludes {
		opts = append(opts, restic.WithFlags("--iexclude", iexclude))
	}
	if len(snapshots) > 0 {
		opts = append(opts, restic.WithFlags("--parent", snapshots[len(snapshots)-1].Id))
	}

	for _, f := range plan.GetBackupFlags() {
		args, err := shlex.Split(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse backup flag %q for plan %q: %w", f, plan.Id, err)
		}
		opts = append(opts, restic.WithFlags(args...))
	}

	ctx, flush := forwardResticLogs(ctx)
	defer flush()
	l.Debug("starting backup", zap.String("plan", plan.Id))
	summary, err := r.repo.Backup(ctx, plan.Paths, progressCallback, opts...)
	if err != nil {
		return summary, fmt.Errorf("failed to backup: %w", err)
	}

	l.Debug("backup completed", zap.Duration("duration", time.Since(startTime)))
	return summary, nil
}

func (r *RepoOrchestrator) ListSnapshotFiles(ctx context.Context, snapshotId string, path string) ([]*v1.LsEntry, error) {
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

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

func (r *RepoOrchestrator) Forget(ctx context.Context, plan *v1.Plan, tags []string) ([]*v1.ResticSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	policy := plan.Retention
	if policy == nil {
		return nil, fmt.Errorf("plan %q has no retention policy", plan.Id)
	}

	result, err := r.repo.Forget(
		ctx, protoutil.RetentionPolicyFromProto(plan.Retention),
		restic.WithFlags("--tag", strings.Join(tags, ",")),
		restic.WithFlags("--group-by", ""),
	)
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

	r.logger(ctx).Debug("forget snapshots", zap.String("plan", plan.Id), zap.Int("count", len(forgotten)), zap.Any("policy", policy))

	return forgotten, nil
}

func (r *RepoOrchestrator) ForgetSnapshot(ctx context.Context, snapshotId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	r.logger(ctx).Debug("forget snapshot with ID", zap.String("snapshot", snapshotId), zap.String("repo", r.repoConfig.Id))
	return r.repo.ForgetSnapshot(ctx, snapshotId)
}

func (r *RepoOrchestrator) Prune(ctx context.Context, output io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

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

	r.logger(ctx).Debug("prune snapshots")
	err := r.repo.Prune(ctx, output, opts...)
	if err != nil {
		return fmt.Errorf("prune snapshots for repo %v: %w", r.repoConfig.Id, err)
	}
	return nil
}

func (r *RepoOrchestrator) Check(ctx context.Context, output io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	var opts []restic.GenericOption
	if r.repoConfig.CheckPolicy != nil {
		switch m := r.repoConfig.CheckPolicy.Mode.(type) {
		case *v1.CheckPolicy_ReadDataSubsetPercent:
			if m.ReadDataSubsetPercent > 0 {
				opts = append(opts, restic.WithFlags(fmt.Sprintf("--read-data-subset=%.4f%%", m.ReadDataSubsetPercent)))
			}
		case *v1.CheckPolicy_StructureOnly:
		default:
		}
	}

	r.logger(ctx).Debug("checking repo")
	err := r.repo.Check(ctx, output, opts...)
	if err != nil {
		return fmt.Errorf("check repo %v: %w", r.repoConfig.Id, err)
	}
	return nil
}

func (r *RepoOrchestrator) Restore(ctx context.Context, snapshotId string, snapshotPath string, target string, progressCallback func(event *v1.RestoreProgressEntry)) (*v1.RestoreProgressEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	r.logger(ctx).Debug("restore snapshot", zap.String("snapshot", snapshotId), zap.String("target", target))

	var opts []restic.GenericOption
	opts = append(opts, restic.WithFlags("--target", target))

	if snapshotPath != "" {
	    dir := path.Dir(snapshotPath)
	    base := path.Base(snapshotPath)
	    if dir != "" {
		snapshotId = snapshotId + ":" + dir
	    }
	    if base != "" {
		opts = append(opts, restic.WithFlags("--include", base))
	    }
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
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	r.logger(ctx).Debug("auto-unlocking repo", zap.String("repo", r.repoConfig.Id))

	return r.repo.Unlock(ctx)
}

func (r *RepoOrchestrator) Unlock(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger(ctx).Debug("unlocking repo", zap.String("repo", r.repoConfig.Id))
	r.repo.Unlock(ctx)

	return nil
}

func (r *RepoOrchestrator) Stats(ctx context.Context) (*v1.RepoStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	r.logger(ctx).Debug("getting repo stats", zap.String("repo", r.repoConfig.Id))
	stats, err := r.repo.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats for repo %v: %w", r.repoConfig.Id, err)
	}

	return protoutil.RepoStatsToProto(stats), nil
}

func (r *RepoOrchestrator) AddTags(ctx context.Context, snapshotIDs []string, tags []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	for idx, snapshotIDs := range chunkBy(snapshotIDs, 20) {
		r.logger(ctx).Debug("adding tag to snapshots", zap.Strings("snapshots", snapshotIDs), zap.Strings("tags", tags))
		if err := r.repo.AddTags(ctx, snapshotIDs, tags); err != nil {
			return fmt.Errorf("batch %v: %w", idx, err)
		}
	}

	return nil
}

// RunCommand runs a command in the repo's environment.
// NOTE: this function does not lock the repo.
func (r *RepoOrchestrator) RunCommand(ctx context.Context, command string, writer io.Writer) error {
	ctx, flush := forwardResticLogs(ctx)
	defer flush()

	r.logger(ctx).Debug("running command", zap.String("command", command))
	args, err := shlex.Split(command)
	if err != nil {
		return fmt.Errorf("parse command: %w", err)
	}

	ctx = restic.ContextWithLogger(ctx, writer)
	return r.repo.GenericCommand(ctx, args)
}

func (r *RepoOrchestrator) Config() *v1.Repo {
	if r == nil {
		return nil
	}
	return r.repoConfig
}

func (r *RepoOrchestrator) RepoGUID() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg, err := r.repo.Config(context.Background())
	return cryptoutil.TruncateID(cfg.Id, cryptoutil.DefaultIDBits), err
}

func sortSnapshotsByTime(snapshots []*restic.Snapshot) {
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].UnixTimeMs() < snapshots[j].UnixTimeMs()
	})
}

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}
