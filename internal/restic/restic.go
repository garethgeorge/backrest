package restic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/hashicorp/go-multierror"
)

type Repo struct {
	mu sync.Mutex
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

func (r *Repo) Init(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initialized = false
	return r.init(ctx)
}

func (r *Repo) Backup(ctx context.Context, progressCallback func(*BackupEvent), opts ...BackupOption) (*BackupEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize repo: %w", err)
	}
	
	opt := &BackupOpts{}
	for _, o := range opts {
		o(opt)
	}

	for _, p := range opt.paths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("path %s does not exist: %w", p, err)
		}
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
	
	var wg sync.WaitGroup
	var summary *BackupEvent
	var cmdErr error 
	var readErr error
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		summary, err = readBackupEvents(cmd, reader, progressCallback)
		if err != nil {
			readErr = fmt.Errorf("processing command output: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer writer.Close()
		defer wg.Done()
		if err := cmd.Wait(); err != nil {
			cmdErr = NewCmdError(cmd, nil, err)
		}
	}()

	wg.Wait()
	
	var err error
	if cmdErr != nil || readErr != nil {
		err = multierror.Append(nil, cmdErr, readErr)
	}
	return summary, err
}

func (r *Repo) Snapshots(ctx context.Context) ([]*Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize repo: %w", err)
	}

	args := []string{"snapshots", "--json"}
	args = append(args, r.flags...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, NewCmdError(cmd, output, err)
	}

	var snapshots []*Snapshot
	if err := json.Unmarshal(output, &snapshots); err != nil {
		return nil, NewCmdError(cmd, output, fmt.Errorf("command output is not valid JSON: %w", err))
	}

	return snapshots, nil
}

func (r *Repo) ListDirectory(ctx context.Context, snapshot string, path string) (*Snapshot, []*LsEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if path == "" {
		// an empty path can trigger very expensive operations (e.g. iterates all files in the snapshot)
		return nil, nil, errors.New("path must not be empty")
	}

	if err := r.init(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize repo: %w", err)
	}

	args := []string{"ls", "--json", snapshot, path}
	args = append(args, r.flags...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, NewCmdError(cmd, output, err)
	}


	snapshots, entries, err := readLs(bytes.NewBuffer(output))
	if err != nil {
		return nil, nil, NewCmdError(cmd, output, err)
	}

	return snapshots, entries, nil
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
