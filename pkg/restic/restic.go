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
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/djherbis/buffer"
	nio "github.com/djherbis/nio/v3"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"go.uber.org/zap"
)

var errAlreadyInitialized = errors.New("repo already initialized")
var ErrPartialBackup = errors.New("incomplete backup")
var ErrBackupFailed = errors.New("backup failed")
var ErrRestoreFailed = errors.New("restore failed")
var ErrRepoNotFound = errors.New("repo does not exist")

type Repo struct {
	cmd string
	uri string

	opts []GenericOption

	exists           error
	checkExists      sync.Once
	initialized      error // nil or errAlreadyInitialized if initialized, error if initialization failed.
	shouldInitialize sync.Once
	repoConfig       RepoConfig // set by init (which calls Exists)
}

// NewRepo instantiates a new repository.
func NewRepo(resticBin string, uri string, opts ...GenericOption) *Repo {
	opts = append(opts, WithEnv("RESTIC_REPOSITORY="+uri))

	return &Repo{
		cmd:  resticBin,
		uri:  uri,
		opts: opts,
	}
}

func (r *Repo) commandWithContext(ctx context.Context, args []string, opts ...GenericOption) *exec.Cmd {
	opt := &GenericOpts{}
	resolveOpts(opt, r.opts)
	resolveOpts(opt, opts)

	var fullCmd []string
	fullCmd = append(fullCmd, opt.prefixCmd...)
	if r.cmd != "" {
		fullCmd = append(fullCmd, r.cmd)
	}
	fullCmd = append(fullCmd, args...)
	fullCmd = append(fullCmd, opt.extraArgs...)

	cmd := exec.CommandContext(ctx, fullCmd[0], fullCmd[1:]...)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	logger := LoggerFromContext(ctx)
	if logger != nil {
		sw := &ioutil.SynchronizedWriter{W: logger}
		cmd.Stderr = sw
		cmd.Stdout = sw
		fmt.Fprintf(logger, "\ncommand: %q\n", fullCmd)
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

	mw := io.MultiWriter(handlers...)
	mw = &ioutil.SynchronizedWriter{W: mw}
	cmd.Stdout = mw
	cmd.Stderr = mw
}

// executeWithOutput runs a command and captures the output in a buffer
func (r *Repo) executeWithOutput(ctx context.Context, args []string, opts ...GenericOption) ([]byte, error) {
	cmd := r.commandWithContext(ctx, args, opts...)
	output := bytes.NewBuffer(nil)
	r.pipeCmdOutputToWriter(cmd, output)

	err := cmd.Run()
	if err != nil {
		return output.Bytes(), newCmdError(ctx, cmd, newErrorWithOutput(err, output.String()))
	}

	return output.Bytes(), nil
}

// executeWithJSONOutput runs a command and parses its JSON output
func (r *Repo) executeWithJSONOutput(ctx context.Context, args []string, result interface{}, opts ...GenericOption) error {
	output, err := r.executeWithOutput(ctx, args, opts...)
	if err != nil {
		return err
	}

	// Try to parse the entire output first
	origErr := json.Unmarshal(output, result)
	if origErr == nil {
		return nil
	}

	// Find the index afterwhich everything is whitespace
	allWhitespaceAfterIdx := len(output)
	for i, b := range output {
		if unicode.IsSpace(rune(b)) {
			allWhitespaceAfterIdx = i
		}
	}

	// If that fails, try by skipping bytes until a newline is found
	start := 0
	for start < allWhitespaceAfterIdx {
		if err := json.Unmarshal(output[start:], result); err == nil {
			zap.S().Warnf("Command %v output may have contained a skipped warning from restic that was not valid JSON: %s", args, string(output[start:]))
			return nil
		}
		start = start + bytes.IndexRune(output[start:], '\n')
		start++ // skip the newline itself
	}

	return newCmdError(ctx, r.commandWithContext(ctx, args),
		newErrorWithOutput(fmt.Errorf("command output is not valid JSON: %w", origErr), string(output)))
}

// Exists checks if the repository exists.
// Returns true if exists, false if it does not exist OR an access error occurred.
func (r *Repo) Exists(ctx context.Context, opts ...GenericOption) error {
	r.checkExists.Do(func() {
		output := bytes.NewBuffer(nil)
		cmd := r.commandWithContext(ctx, []string{"cat", "config"}, opts...)
		r.pipeCmdOutputToWriter(cmd, output)
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 10 {
				err = ErrRepoNotFound
			}
			r.exists = newCmdError(ctx, cmd, newErrorWithOutput(err, output.String()))
		} else if err := json.Unmarshal(output.Bytes(), &r.repoConfig); err != nil {
			r.exists = newCmdError(ctx, cmd, newErrorWithOutput(fmt.Errorf("command output is not valid JSON: %w", err), output.String()))
		} else {
			r.exists = nil
		}
	})
	return r.exists
}

// init initializes the repo, the command will be cancelled with the context.
func (r *Repo) init(ctx context.Context, opts ...GenericOption) error {
	if r.Exists(ctx, opts...) == nil {
		return nil
	}

	r.shouldInitialize.Do(func() {
		cmd := r.commandWithContext(ctx, []string{"init", "--json"}, opts...)
		output := bytes.NewBuffer(nil)
		r.pipeCmdOutputToWriter(cmd, output)

		if err := cmd.Run(); err != nil {
			if strings.Contains(output.String(), "config file already exists") || strings.Contains(output.String(), "already initialized") {
				r.initialized = errAlreadyInitialized
			} else {
				r.initialized = newCmdError(ctx, cmd, newCmdError(ctx, cmd, newErrorWithOutput(err, output.String())))
			}
		} else {
			if err := json.Unmarshal(output.Bytes(), &r.repoConfig); err != nil {
				r.initialized = newCmdError(ctx, cmd, newErrorWithOutput(fmt.Errorf("command output is not valid JSON: %w", err), output.String()))
			}
			r.exists = r.initialized
		}
	})

	return r.initialized
}

func (r *Repo) Init(ctx context.Context, opts ...GenericOption) error {
	return r.init(ctx, opts...)
}

func (r *Repo) Config(ctx context.Context, opts ...GenericOption) (RepoConfig, error) {
	if err := r.Exists(ctx, opts...); err != nil {
		return RepoConfig{}, err
	}
	return r.repoConfig, nil
}

type cmdRunnerWithProgress[T ProgressEntryValidator] struct {
	repo       *Repo
	callback   func(T)
	failureErr error
}

func (cr *cmdRunnerWithProgress[T]) Run(ctx context.Context, args []string, opts ...GenericOption) (T, error) {
	logger := LoggerFromContext(ctx)
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmdCtx = ContextWithLogger(cmdCtx, nil) // ensure no logger is used
	cmd := cr.repo.commandWithContext(cmdCtx, args, opts...)

	// Ensure the command is logged since we're overriding the logger
	if logger != nil {
		fmt.Fprintf(logger, "command: %q\n", cmd)
	}

	buf := buffer.New(32 * 1024) // 32KB IO buffer for the realtime event parsing
	reader, writer := nio.Pipe(buf)
	cr.repo.pipeCmdOutputToWriter(cmd, writer)

	var readErr error
	var summary T
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := processProgressOutput[T](reader, logger, cr.callback)
		summary = result
		if err != nil {
			readErr = fmt.Errorf("processing command output: %w", err)
		}
	}()

	cmdErr := cmd.Run()
	writer.Close()
	wg.Wait()

	if cmdErr != nil || readErr != nil {
		if cmdErr != nil {
			cmdErr = cr.repo.handleExitError(cmdErr, cr.failureErr)
		}
		return summary, newCmdError(ctx, cmd, errors.Join(cmdErr, readErr))
	}

	return summary, nil
}

func (r *Repo) Backup(ctx context.Context, paths []string, progressCallback func(*BackupProgressEntry), opts ...GenericOption) (*BackupProgressEntry, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("path %s does not exist: %w", p, err)
		}
	}

	args := []string{"backup", "--json"}
	args = append(args, paths...)
	opts = append(slices.Clone(opts), WithEnv("RESTIC_PROGRESS_FPS=2"))

	cr := cmdRunnerWithProgress[*BackupProgressEntry]{
		repo:       r,
		callback:   progressCallback,
		failureErr: ErrBackupFailed,
	}
	return cr.Run(ctx, args, opts...)
}

func (r *Repo) Restore(ctx context.Context, snapshot string, callback func(*RestoreProgressEntry), opts ...GenericOption) (*RestoreProgressEntry, error) {
	opts = append(slices.Clone(opts), WithEnv("RESTIC_PROGRESS_FPS=2"))
	args := []string{"restore", "--json", snapshot}

	cr := cmdRunnerWithProgress[*RestoreProgressEntry]{
		repo:       r,
		callback:   callback,
		failureErr: ErrRestoreFailed,
	}
	return cr.Run(ctx, args, opts...)
}

// handleExitError processes a command exit error and converts it to an appropriate error type
func (r *Repo) handleExitError(err error, failureErr error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 3 {
			return ErrPartialBackup
		} else {
			return fmt.Errorf("exit code %d: %w", exitErr.ExitCode(), failureErr)
		}
	}
	return err
}

func (r *Repo) Snapshots(ctx context.Context, opts ...GenericOption) ([]*Snapshot, error) {
	var snapshots []*Snapshot
	if err := r.executeWithJSONOutput(ctx, []string{"snapshots", "--json"}, &snapshots, opts...); err != nil {
		return nil, err
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

	var results []ForgetResult
	if err := r.executeWithJSONOutput(ctx, args, &results, opts...); err != nil {
		return nil, err
	}

	if len(results) != 1 {
		return nil, fmt.Errorf("expected 1 output from forget, got %v", len(results))
	}

	if err := results[0].Validate(); err != nil {
		return nil, fmt.Errorf("invalid forget result: %w", err)
	}

	return &results[0], nil
}

func (r *Repo) ForgetSnapshot(ctx context.Context, snapshotId string, opts ...GenericOption) error {
	args := []string{"forget", "--json", snapshotId}
	_, err := r.executeWithOutput(ctx, args, opts...)
	return err
}

func (r *Repo) Prune(ctx context.Context, pruneOutput io.Writer, opts ...GenericOption) error {
	return r.runSimpleCommand(ctx, []string{"prune"}, pruneOutput, opts...)
}

func (r *Repo) Check(ctx context.Context, checkOutput io.Writer, opts ...GenericOption) error {
	cmd := r.commandWithContext(ctx, []string{"check"}, opts...)
	if checkOutput != nil {
		r.pipeCmdOutputToWriter(cmd, checkOutput)
	}
	if err := cmd.Run(); err != nil {
		return newCmdError(ctx, cmd, err)
	}
	return nil
}

// runSimpleCommand executes a command with optional output capture
func (r *Repo) runSimpleCommand(ctx context.Context, args []string, outputWriter io.Writer, opts ...GenericOption) error {
	cmd := r.commandWithContext(ctx, args, opts...)
	if outputWriter != nil {
		r.pipeCmdOutputToWriter(cmd, outputWriter)
	}
	if err := cmd.Run(); err != nil {
		return newCmdError(ctx, cmd, err)
	}
	return nil
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
		return nil, nil, newCmdError(ctx, cmd, newErrorWithOutput(err, output.String()))
	}

	snap, entries, err := readLs(output)
	if err != nil {
		return nil, nil, newCmdError(ctx, cmd, fmt.Errorf("error parsing JSON: %w", err))
	}
	return snap, entries, nil
}

func (r *Repo) Unlock(ctx context.Context, opts ...GenericOption) error {
	_, err := r.executeWithOutput(ctx, []string{"unlock"}, opts...)
	return err
}

func (r *Repo) Stats(ctx context.Context, opts ...GenericOption) (*RepoStats, error) {
	var stats RepoStats
	err := r.executeWithJSONOutput(ctx, []string{"stats", "--json", "--mode=raw-data"}, &stats, opts...)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *Repo) Mount(ctx context.Context, dir string, mountTimeout time.Duration, opts ...GenericOption) error {
	// Check if already mounted
	if info, err := os.Stat(dir); err == nil {
		if info.IsDir() {
			// Try to detect if it's already a FUSE mount by checking if we can list it
			if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
				return fmt.Errorf("directory %s appears to already be mounted", dir)
			}
		}
	}

	// Start the mount command in background
	cmd := r.commandWithContext(ctx, []string{"mount", dir}, opts...)
	if err := cmd.Start(); err != nil {
		return newCmdError(ctx, cmd, err)
	}

	// Wait for mount to become available
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(mountTimeout)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		case <-timeout.C:
			cmd.Process.Kill()
			return fmt.Errorf("mount timeout")
		case <-ticker.C:
			if _, err := os.Stat(dir); err == nil {
				if ents, err := os.ReadDir(filepath.Join(dir, "ids")); err == nil && len(ents) > 0 {
					return nil
				}
			}

			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				return newCmdError(ctx, cmd, fmt.Errorf("restic mount process exited with code %d", cmd.ProcessState.ExitCode()))
			}
		}
	}
}

// AddTags adds tags to the specified snapshots.
func (r *Repo) AddTags(ctx context.Context, snapshotIDs []string, tags []string, opts ...GenericOption) error {
	args := []string{"tag"}
	args = append(args, "--add", strings.Join(tags, ","))
	args = append(args, snapshotIDs...)

	_, err := r.executeWithOutput(ctx, args, opts...)
	return err
}

func (r *Repo) GenericCommand(ctx context.Context, args []string, opts ...GenericOption) error {
	cmd := r.commandWithContext(ctx, args, opts...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
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
	prefixCmd []string
}

func resolveOpts(opt *GenericOpts, opts []GenericOption) {
	for _, o := range opts {
		o(opt)
	}
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

func WithPrefixCommand(args ...string) GenericOption {
	return func(opts *GenericOpts) {
		opts.prefixCmd = append(opts.prefixCmd, args...)
	}
}
