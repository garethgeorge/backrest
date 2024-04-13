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

	"github.com/djherbis/buffer"
	nio "github.com/djherbis/nio/v3"
	"github.com/garethgeorge/backrest/internal/ioutil"
)

var errAlreadyInitialized = errors.New("repo already initialized")
var ErrPartialBackup = errors.New("incomplete backup")
var ErrBackupFailed = errors.New("backup failed")

type Repo struct {
	cmd         string
	uri         string
	initialized bool

	extraArgs []string
	extraEnv  []string
}

// NewRepo instantiates a new repository.
func NewRepo(resticBin string, uri string, opts ...GenericOption) *Repo {
	opt := &GenericOpts{}
	for _, o := range opts {
		o(opt)
	}

	opt.extraEnv = append(opt.extraEnv, "RESTIC_REPOSITORY="+uri)

	return &Repo{
		cmd:         resticBin, // TODO: configurable binary path
		uri:         uri,
		initialized: false,
		extraArgs:   opt.extraArgs,
		extraEnv:    opt.extraEnv,
	}
}

func (r *Repo) commandWithContext(ctx context.Context, args []string, opts ...GenericOption) *exec.Cmd {
	opt := resolveOpts(opts)

	args = append(args, r.extraArgs...)
	args = append(args, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, r.cmd, args...)
	cmd.Env = append(cmd.Env, r.extraEnv...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	addLoggingToCommand(ctx, cmd)

	if logger := LoggerFromContext(ctx); logger != nil {
		fmt.Fprintf(logger, "\ncommand: %v %v\n", r.cmd, strings.Join(args, " "))
	}

	return cmd
}

func (r *Repo) pipeCmdOutputToWriter(cmd *exec.Cmd, handlers ...io.Writer) {
	stdoutHandlers := slices.Clone(handlers)
	stderrHandlers := slices.Clone(handlers)

	if cmd.Stdout != nil {
		handlers = append(stdoutHandlers, cmd.Stdout)
	}
	if cmd.Stderr != nil {
		handlers = append(stderrHandlers, cmd.Stderr)
	}

	cmd.Stdout = io.MultiWriter(handlers...)
	cmd.Stderr = io.MultiWriter(handlers...)
}

// init initializes the repo, the command will be cancelled with the context.
func (r *Repo) init(ctx context.Context, opts ...GenericOption) error {
	if r.initialized {
		return nil
	}

	cmd := r.commandWithContext(ctx, []string{"init", "--json"}, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)

	if err := cmd.Run(); err != nil {
		if strings.Contains(output.String(), "config file already exists") || strings.Contains(output.String(), "already initialized") {
			return errAlreadyInitialized
		}
		return newCmdError(ctx, cmd, output.String(), err)
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

func (r *Repo) Backup(ctx context.Context, paths []string, progressCallback func(*BackupProgressEntry), opts ...GenericOption) (*BackupProgressEntry, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("path %s does not exist: %w", p, err)
		}
	}

	args := []string{"backup", "--json", "--exclude-caches"}
	args = append(args, paths...)

	cmd := r.commandWithContext(ctx, args, opts...)
	fullOutput := ioutil.NewOutputCapturer(outputBufferLimit)
	buf := buffer.New(32 * 1024) // 32KB IO buffer for the realtime event parsing
	reader, writer := nio.Pipe(buf)
	r.pipeCmdOutputToWriter(cmd, fullOutput, writer)

	var readErr error
	var summary *BackupProgressEntry
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		summary, err = readBackupProgressEntries(reader, progressCallback)
		if err != nil {
			readErr = fmt.Errorf("processing command output: %w", err)
			_ = cmd.Cancel() // cancel the command to prevent it from hanging now that we're not reading from it.
		}
	}()

	cmdErr := cmd.Run()
	writer.Close()
	wg.Wait()

	if cmdErr != nil || readErr != nil {
		if cmdErr != nil {
			var exitErr *exec.ExitError
			if errors.As(cmdErr, &exitErr) {
				if exitErr.ExitCode() == 3 {
					cmdErr = ErrPartialBackup
				} else {
					cmdErr = fmt.Errorf("exit code %d: %w", exitErr.ExitCode(), ErrBackupFailed)
				}
			}
		}
		return summary, newCmdErrorPreformatted(ctx, cmd, string(fullOutput.Bytes()), errors.Join(cmdErr, readErr))
	}
	return summary, nil
}

func (r *Repo) Snapshots(ctx context.Context, opts ...GenericOption) ([]*Snapshot, error) {
	cmd := r.commandWithContext(ctx, []string{"snapshots", "--json"}, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)

	if err := cmd.Run(); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), err)
	}

	var snapshots []*Snapshot
	if err := json.Unmarshal(output.Bytes(), &snapshots); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), fmt.Errorf("command output is not valid JSON: %w", err))
	}

	for _, snapshot := range snapshots {
		if err := snapshot.Validate(); err != nil {
			return nil, fmt.Errorf("invalid snapshot: %w", err)
		}
	}
	return snapshots, nil
}

func (r *Repo) Forget(ctx context.Context, policy *RetentionPolicy, opts ...GenericOption) (*ForgetResult, error) {
	args := []string{"forget", "--json"}
	args = append(args, policy.toForgetFlags()...)

	cmd := r.commandWithContext(ctx, args, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)
	if err := cmd.Run(); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), err)
	}

	var result []ForgetResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), fmt.Errorf("command output is not valid JSON: %w", err))
	}
	if len(result) != 1 {
		return nil, fmt.Errorf("expected 1 output from forget, got %v", len(result))
	}
	if err := result[0].Validate(); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), fmt.Errorf("invalid forget result: %w", err))
	}

	return &result[0], nil
}

func (r *Repo) ForgetSnapshot(ctx context.Context, snapshotId string, opts ...GenericOption) error {
	args := []string{"forget", "--json", snapshotId}

	cmd := r.commandWithContext(ctx, args, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)
	if err := cmd.Run(); err != nil {
		return newCmdError(ctx, cmd, output.String(), err)
	}

	return nil
}

func (r *Repo) Prune(ctx context.Context, pruneOutput io.Writer, opts ...GenericOption) error {
	args := []string{"prune"}
	cmd := r.commandWithContext(ctx, args, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)
	if pruneOutput != nil {
		r.pipeCmdOutputToWriter(cmd, pruneOutput)
	}
	if err := cmd.Run(); err != nil {
		return newCmdError(ctx, cmd, output.String(), err)
	}
	return nil
}

func (r *Repo) Restore(ctx context.Context, snapshot string, callback func(*RestoreProgressEntry), opts ...GenericOption) (*RestoreProgressEntry, error) {
	cmd := r.commandWithContext(ctx, []string{"restore", "--json", snapshot}, opts...)
	capture := ioutil.NewOutputCapturer(outputBufferLimit) // for error reporting.
	reader, writer := io.Pipe()
	r.pipeCmdOutputToWriter(cmd, writer, capture)

	var readErr error
	var summary *RestoreProgressEntry
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		summary, err = readRestoreProgressEntries(reader, callback)
		if err != nil {
			readErr = fmt.Errorf("processing command output: %w", err)
			_ = cmd.Cancel() // cancel the command to prevent it from hanging now that we're not reading from it.
		}
	}()

	cmdErr := cmd.Run()
	writer.Close()
	wg.Wait()
	if cmdErr != nil || readErr != nil {
		if cmdErr != nil {
			var exitErr *exec.ExitError
			if errors.As(cmdErr, &exitErr) {
				if exitErr.ExitCode() == 3 {
					cmdErr = ErrPartialBackup
				} else {
					cmdErr = fmt.Errorf("exit code %d: %w", exitErr.ExitCode(), ErrBackupFailed)
				}
			}
		}

		return summary, newCmdErrorPreformatted(ctx, cmd, string(capture.Bytes()), errors.Join(cmdErr, readErr))
	}
	return summary, nil
}

func (r *Repo) ListDirectory(ctx context.Context, snapshot string, path string, opts ...GenericOption) (*Snapshot, []*LsEntry, error) {
	if path == "" {
		// an empty path can trigger very expensive operations (e.g. iterates all files in the snapshot)
		return nil, nil, errors.New("path must not be empty")
	}

	cmd := r.commandWithContext(ctx, []string{"ls", "--json", snapshot, path}, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)

	if err := cmd.Run(); err != nil {
		return nil, nil, newCmdError(ctx, cmd, output.String(), err)
	}

	snapshots, entries, err := readLs(output)
	if err != nil {
		return nil, nil, newCmdError(ctx, cmd, output.String(), err)
	}

	return snapshots, entries, nil
}

func (r *Repo) Unlock(ctx context.Context, opts ...GenericOption) error {
	cmd := r.commandWithContext(ctx, []string{"unlock"}, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)
	if err := cmd.Run(); err != nil {
		return newCmdError(ctx, cmd, output.String(), err)
	}
	return nil
}

func (r *Repo) Stats(ctx context.Context, opts ...GenericOption) (*RepoStats, error) {
	cmd := r.commandWithContext(ctx, []string{"stats", "--json", "--mode=raw-data"}, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)

	if err := cmd.Run(); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), err)
	}

	var stats RepoStats
	if err := json.Unmarshal(output.Bytes(), &stats); err != nil {
		return nil, newCmdError(ctx, cmd, output.String(), fmt.Errorf("command output is not valid JSON: %w", err))
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
