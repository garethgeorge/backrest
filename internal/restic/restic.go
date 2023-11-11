package restic

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"golang.org/x/sync/errgroup"
)

type Repo struct {
	cmd string
	repo *v1.Repo
	flags []string
	env []string
	initialized bool
}

func NewRepo(repo *v1.Repo, opts ...RepoOption) *Repo {
	var opt RepoOpts
	for _, o := range opts {
		o(&opt)
	}

	return &Repo{
		cmd: "restic", // TODO: configurable binary path
		repo: repo,
		flags: opt.flags,
		env: opt.env,
	}
}

func (r *Repo) buildEnv() []string {
	env := []string{
		"RESTIC_REPOSITORY=" + r.repo.GetUri(),
		"RESTIC_PASSWORD=" + r.repo.GetPassword(),
	}
	env = append(env, r.repo.GetEnv()...)
	env = append(env, r.env...)
	return env
}

// init initializes the repo, the command will be cancelled with the context.
func (r *Repo) init(ctx context.Context) error {
	if r.initialized {
		return nil
	}

	var args = []string{"init", "--json"}
	args = append(args, r.flags...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return NewCmdError(cmd, output, err)
	}

	r.initialized = true
	return nil
}

func (r *Repo) Backup(ctx context.Context, progressCallback func(*BackupEvent), opts ...BackupOption) (*BackupEvent, error) {
	if err := r.init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize repo: %w", err)
	}
	
	opt := &BackupOpts{}
	for _, o := range opts {
		o(opt)
	}

	args := []string{"backup", "--json", "--exclude-caches"}
	args = append(args, r.flags...)
	args = append(args, opt.paths...)

	for _, e := range opt.excludes {
		args = append(args, "--exclude", e)
	}

	reader, writer := io.Pipe()

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Stderr = writer
	cmd.Stdout = writer

	if err := cmd.Start(); err != nil {
		return nil, NewCmdError(cmd, nil, err)
	}
	
	var summary *BackupEvent
	var errgroup errgroup.Group
	
	errgroup.Go(func() error {
		var err error
		summary, err = readBackupEvents(cmd, reader, progressCallback)
		if err != nil {
			return fmt.Errorf("processing command output: %w", err)
		}
		return nil
	})

	errgroup.Go(func() error {
		defer writer.Close()
		if err := cmd.Wait(); err != nil {
			return NewCmdError(cmd, nil, err)
		}
		return nil
	})

	if err := errgroup.Wait(); err != nil {
		return nil, err
	}

	return summary, nil
}

type RepoOpts struct {
	env []string // global env overrides
	flags []string // global flags
}

type RepoOption func(opts *RepoOpts)

// WithHostEnv copies values from the host environment into the restic environment.
func WithRepoHostEnv() RepoOption {
	return func(opts *RepoOpts) {
		opts.env = append(opts.env, "HOME=" + os.Getenv("HOME"), "XDG_CACHE_HOME=" + os.Getenv("XDG_CACHE_HOME"))
	}
}

func WithRepoEnv(env ...string) RepoOption {
	return func(opts *RepoOpts) {
		opts.env = append(opts.env, env...)
	}
}

func WithRepoFlags(flags ...string) RepoOption {
	return func(opts *RepoOpts) {
		opts.flags = append(opts.flags, flags...)
	}
}

type BackupOpts struct {
	paths []string
	excludes []string
}

type BackupOption func(opts *BackupOpts)

func WithBackupPaths(paths ...string) BackupOption {
	return func(opts *BackupOpts) {
		opts.paths = append(opts.paths, paths...)
	}
}

func WithBackupExcludes(excludes ...string) BackupOption {
	return func(opts *BackupOpts) {
		opts.excludes = append(opts.excludes, excludes...)
	}
}
