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
	"slices"
	"strings"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

var errAlreadyInitialized = errors.New("repo already initialized")
var ErrPartialBackup = errors.New("incomplete backup")
var ErrBackupFailed = errors.New("backup failed")

type Repo struct {
	cmd         string
	repo        *v1.Repo
	initialized bool

	extraArgs []string
	extraEnv  []string
}

// NewRepo instantiates a new repository. TODO: should not accept a v1.Repo, should instead be configured by parameters.
func NewRepo(resticBin string, repo *v1.Repo, opts ...GenericOption) *Repo {
	opt := &GenericOpts{}
	for _, o := range opts {
		o(opt)
	}

	if slices.IndexFunc(opt.extraArgs, func(a string) bool {
		return strings.Contains(a, "sftp.args")
	}) == -1 {
		opt.extraArgs = append(opt.extraArgs, "-o", "sftp.args=-oBatchMode=yes")
	}

	return &Repo{
		cmd:         resticBin, // TODO: configurable binary path
		repo:        repo,
		initialized: false,
		extraArgs:   opt.extraArgs,
		extraEnv:    opt.extraEnv,
	}
}

func (r *Repo) buildEnv() []string {
	env := []string{
		"RESTIC_REPOSITORY=" + r.repo.GetUri(),
		"RESTIC_PASSWORD=" + r.repo.GetPassword(),
	}
	env = append(env, r.extraEnv...)
	env = append(env, r.repo.GetEnv()...)
	return env
}

// init initializes the repo, the command will be cancelled with the context.
func (r *Repo) init(ctx context.Context, opts ...GenericOption) error {
	if r.initialized {
		return nil
	}

	opt := resolveOpts(opts)

	var args = []string{"init", "--json"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	if output, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(output), "config file already exists") || strings.Contains(string(output), "already initialized") {
			return errAlreadyInitialized
		}
		return newCmdError(cmd, string(output), err)
	}

	r.initialized = true
	return nil
}

func (r *Repo) Init(ctx context.Context, opts ...GenericOption) error {
	if err := r.init(ctx, opts...); err != nil && !errors.Is(err, errAlreadyInitialized) {
		return fmt.Errorf("init failed: %w", err)
	}
	return nil
}

func (r *Repo) Backup(ctx context.Context, progressCallback func(*BackupProgressEntry), opts ...BackupOption) (*BackupProgressEntry, error) {
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
	args = append(args, r.extraArgs...)
	args = append(args, opt.paths...)
	args = append(args, opt.extraArgs...)

	output := newOutputCapturer(outputBufferLimit)
	reader, writer := io.Pipe()
	capture := io.MultiWriter(output, writer)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Stderr = capture
	cmd.Stdout = capture

	if err := cmd.Start(); err != nil {
		return nil, newCmdError(cmd, "", err)
	}

	var wg sync.WaitGroup
	var summary *BackupProgressEntry
	var cmdErr error
	var readErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		summary, err = readBackupProgressEntries(cmd, reader, progressCallback)
		if err != nil {
			readErr = fmt.Errorf("processing command output: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer writer.Close()
		defer wg.Done()
		if err := cmd.Wait(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				if exitErr.ExitCode() == 3 {
					cmdErr = ErrPartialBackup
				} else {
					cmdErr = fmt.Errorf("exit code %v: %w", exitErr.ExitCode(), ErrBackupFailed)
				}
				return
			}
			cmdErr = err
		}
	}()

	wg.Wait()

	if cmdErr != nil || readErr != nil {
		return summary, newCmdErrorPreformatted(cmd, output.String(), errors.Join(cmdErr, readErr))
	}

	return summary, nil
}

func (r *Repo) Snapshots(ctx context.Context, opts ...GenericOption) ([]*Snapshot, error) {
	opt := resolveOpts(opts)

	args := []string{"snapshots", "--json"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, newCmdError(cmd, "", err)
	}

	var snapshots []*Snapshot
	if err := json.Unmarshal(output, &snapshots); err != nil {
		return nil, newCmdError(cmd, "", fmt.Errorf("command output is not valid JSON: %w", err))
	}
	for _, snapshot := range snapshots {
		if err := snapshot.Validate(); err != nil {
			return nil, fmt.Errorf("invalid snapshot: %w", err)
		}
	}
	return snapshots, nil
}

func (r *Repo) Forget(ctx context.Context, policy *RetentionPolicy, opts ...GenericOption) (*ForgetResult, error) {
	// first run the forget command
	opt := resolveOpts(opts)

	args := []string{"forget", "--json"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)
	args = append(args, policy.toForgetFlags()...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, newCmdError(cmd, string(output), err)
	}

	var result []ForgetResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, newCmdError(cmd, string(output), fmt.Errorf("command output is not valid JSON: %w", err))
	}
	if len(result) != 1 {
		return nil, fmt.Errorf("expected 1 output from forget, got %v", len(result))
	}
	if err := result[0].Validate(); err != nil {
		return nil, newCmdError(cmd, string(output), fmt.Errorf("invalid forget result: %w", err))
	}

	return &result[0], nil
}

func (r *Repo) Prune(ctx context.Context, pruneOutput io.Writer, opts ...GenericOption) error {
	opt := resolveOpts(opts)

	args := []string{"prune"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	var output = newOutputCapturer(outputBufferLimit)
	var writer io.Writer = output
	if pruneOutput != nil {
		writer = io.MultiWriter(pruneOutput, output)
	}
	cmd.Stdout = writer
	cmd.Stderr = writer

	writer.Write([]byte("command: " + strings.Join(cmd.Args, " ") + "\n"))

	if err := cmd.Run(); err != nil {
		return newCmdErrorPreformatted(cmd, output.String(), err)
	}

	return nil
}

func (r *Repo) Restore(ctx context.Context, snapshot string, callback func(*RestoreProgressEntry), opts ...GenericOption) (*RestoreProgressEntry, error) {
	opt := resolveOpts(opts)

	args := []string{"restore", snapshot, "--json"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	output := newOutputCapturer(outputBufferLimit)
	reader, writer := io.Pipe()
	capture := io.MultiWriter(output, writer)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)
	cmd.Stderr = capture
	cmd.Stdout = capture

	if err := cmd.Start(); err != nil {
		return nil, newCmdError(cmd, "", err)
	}

	var wg sync.WaitGroup
	var summary *RestoreProgressEntry
	var cmdErr error
	var readErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		summary, err = readRestoreProgressEntries(cmd, reader, callback)
		if err != nil {
			readErr = fmt.Errorf("processing command output: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer writer.Close()
		defer wg.Done()
		if err := cmd.Wait(); err != nil {
			cmdErr = err
		}
	}()

	wg.Wait()

	if cmdErr != nil || readErr != nil {
		return nil, newCmdError(cmd, output.String(), errors.Join(cmdErr, readErr))
	}

	return summary, nil
}

func (r *Repo) ListDirectory(ctx context.Context, snapshot string, path string, opts ...GenericOption) (*Snapshot, []*LsEntry, error) {
	if path == "" {
		// an empty path can trigger very expensive operations (e.g. iterates all files in the snapshot)
		return nil, nil, errors.New("path must not be empty")
	}

	opt := resolveOpts(opts)

	args := []string{"ls", "--json", snapshot, path}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, newCmdError(cmd, string(output), err)
	}

	snapshots, entries, err := readLs(bytes.NewBuffer(output))
	if err != nil {
		return nil, nil, newCmdError(cmd, string(output), err)
	}

	return snapshots, entries, nil
}

func (r *Repo) Unlock(ctx context.Context, opts ...GenericOption) error {
	opt := resolveOpts(opts)

	args := []string{"unlock"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return newCmdError(cmd, string(output), err)
	}

	return nil
}

func (r *Repo) Stats(ctx context.Context, opts ...GenericOption) (*RepoStats, error) {
	opt := resolveOpts(opts)

	args := []string{"stats", "--json", "--mode=raw-data"}
	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.buildEnv()...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, newCmdError(cmd, string(output), err)
	}

	var stats RepoStats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, newCmdError(cmd, string(output), fmt.Errorf("command output is not valid JSON: %w", err))
	}

	return &stats, nil
}

type RetentionPolicy struct {
	KeepLastN          int    // keep the last n snapshots.
	KeepHourly         int    // keep the last n hourly snapshots.
	KeepDaily          int    // keep the last n daily snapshots.
	KeepWeekly         int    // keep the last n weekly snapshots.
	KeepMonthly        int    // keep the last n monthly snapshots.
	KeepYearly         int    // keep the last n yearly snapshots.
	KeepWithinDuration string // keep snapshots within a duration e.g. 1y2m3d4h5m6s
}

func (r *RetentionPolicy) toForgetFlags() []string {
	flags := []string{}
	if r.KeepLastN != 0 {
		flags = append(flags, "--keep-last", fmt.Sprintf("%d", r.KeepLastN))
	}
	if r.KeepHourly != 0 {
		flags = append(flags, "--keep-hourly", fmt.Sprintf("%d", r.KeepHourly))
	}
	if r.KeepDaily != 0 {
		flags = append(flags, "--keep-daily", fmt.Sprintf("%d", r.KeepDaily))
	}
	if r.KeepWeekly != 0 {
		flags = append(flags, "--keep-weekly", fmt.Sprintf("%d", r.KeepWeekly))
	}
	if r.KeepMonthly != 0 {
		flags = append(flags, "--keep-monthly", fmt.Sprintf("%d", r.KeepMonthly))
	}
	if r.KeepYearly != 0 {
		flags = append(flags, "--keep-yearly", fmt.Sprintf("%d", r.KeepYearly))
	}
	if r.KeepWithinDuration != "" {
		flags = append(flags, "--keep-within", r.KeepWithinDuration)
	}
	return flags
}

type BackupOpts struct {
	paths     []string
	extraArgs []string
}

type BackupOption func(opts *BackupOpts)

func WithBackupPaths(paths ...string) BackupOption {
	return func(opts *BackupOpts) {
		opts.paths = append(opts.paths, paths...)
	}
}

func WithBackupExcludes(excludes ...string) BackupOption {
	return func(opts *BackupOpts) {
		for _, exclude := range excludes {
			opts.extraArgs = append(opts.extraArgs, "--exclude", exclude)
		}
	}
}

func WithBackupTags(tags ...string) BackupOption {
	return func(opts *BackupOpts) {
		for _, tag := range tags {
			opts.extraArgs = append(opts.extraArgs, "--tag", tag)
		}
	}
}

func WithBackupParent(parent string) BackupOption {
	return func(opts *BackupOpts) {
		opts.extraArgs = append(opts.extraArgs, "--parent", parent)
	}
}

type GenericOpts struct {
	extraArgs []string
	extraEnv  []string
}

func resolveOpts(opts []GenericOption) *GenericOpts {
	opt := &GenericOpts{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

type GenericOption func(opts *GenericOpts)

func WithFlags(flags ...string) GenericOption {
	return func(opts *GenericOpts) {
		opts.extraArgs = append(opts.extraArgs, flags...)
	}
}

func WithTags(tags ...string) GenericOption {
	return func(opts *GenericOpts) {
		for _, tag := range tags {
			opts.extraArgs = append(opts.extraArgs, "--tag", tag)
		}
	}
}

func WithEnv(env ...string) GenericOption {
	return func(opts *GenericOpts) {
		opts.extraEnv = append(opts.extraEnv, env...)
	}
}

var EnvToPropagate = []string{
	// *nix systems
	"PATH", "HOME", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME",
	// windows
	"APPDATA", "LOCALAPPDATA",
}

func WithPropagatedEnvVars(extras ...string) GenericOption {
	var extension []string

	for _, env := range EnvToPropagate {
		if val, ok := os.LookupEnv(env); ok {
			extension = append(extension, env+"="+val)
		}
	}

	return WithEnv(extension...)
}

func WithEnviron() GenericOption {
	return WithEnv(os.Environ()...)
}
