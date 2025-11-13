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
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/djherbis/buffer"
	nio "github.com/djherbis/nio/v3"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/platformutil"
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
	platformutil.SetPlatformOptions(cmd)
	cmd.Env = append(cmd.Env, opt.extraEnv...)

	return cmd
}

type outputOpts struct {
	stdErrWriters []io.Writer
	stdOutWriters []io.Writer
}

func withStdErrTo(writer io.Writer) func(cmd *exec.Cmd, opts *outputOpts) {
	return func(cmd *exec.Cmd, opts *outputOpts) {
		opts.stdErrWriters = append(opts.stdErrWriters, writer)
	}
}

func withStdOutTo(writer io.Writer) func(cmd *exec.Cmd, opts *outputOpts) {
	return func(cmd *exec.Cmd, opts *outputOpts) {
		opts.stdOutWriters = append(opts.stdOutWriters, writer)
	}
}

func withAllTo(writer io.Writer) func(cmd *exec.Cmd, opts *outputOpts) {
	return func(cmd *exec.Cmd, opts *outputOpts) {
		sw := &ioutil.SynchronizedWriter{W: writer}
		opts.stdErrWriters = append(opts.stdErrWriters, sw)
		opts.stdOutWriters = append(opts.stdOutWriters, sw)
	}
}

func withLogWriterFromContext(ctx context.Context) func(cmd *exec.Cmd, opts *outputOpts) {
	return func(cmd *exec.Cmd, opts *outputOpts) {
		logger := LoggerFromContext(ctx)
		if logger != nil {
			fmt.Fprintf(logger, "command: %q\n", cmd)
			opts.stdErrWriters = append(opts.stdErrWriters, logger)
			opts.stdOutWriters = append(opts.stdOutWriters, logger)
		}
	}
}

func (r *Repo) handleOutput(cmd *exec.Cmd, opts ...func(cmd *exec.Cmd, opts *outputOpts)) {
	outputOpts := &outputOpts{}

	for _, opt := range opts {
		opt(cmd, outputOpts)
	}

	var stdOutWriter io.Writer
	if len(outputOpts.stdOutWriters) > 1 {
		stdOutWriter = io.MultiWriter(outputOpts.stdOutWriters...)
	} else if len(outputOpts.stdOutWriters) == 1 {
		stdOutWriter = outputOpts.stdOutWriters[0]
	}

	var stdErrWriter io.Writer
	if len(outputOpts.stdErrWriters) > 1 {
		stdErrWriter = io.MultiWriter(outputOpts.stdErrWriters...)
	} else if len(outputOpts.stdErrWriters) == 1 {
		stdErrWriter = outputOpts.stdErrWriters[0]
	}

	if stdOutWriter != nil {
		if cmd.Stdout != nil {
			cmd.Stdout = io.MultiWriter(cmd.Stdout, stdOutWriter)
		} else {
			cmd.Stdout = stdOutWriter
		}
	}
	if stdErrWriter != nil {
		if cmd.Stderr != nil {
			cmd.Stderr = io.MultiWriter(cmd.Stderr, stdErrWriter)
		} else {
			cmd.Stderr = stdErrWriter
		}
	}
}

// executeWithJSONOutput runs a command and parses its JSON output
func (r *Repo) executeWithJSONOutput(ctx context.Context, args []string, result interface{}, opts ...GenericOption) error {
	// Create a pipe
	errorCollector := errorMessageCollector{}
	stdoutOutput := bytes.NewBuffer(nil)

	// Run the command
	cmd := r.commandWithContext(ctx, args, opts...)
	r.handleOutput(cmd, withAllTo(&errorCollector), withStdOutTo(stdoutOutput), withLogWriterFromContext(ctx))
	if err := cmd.Run(); err != nil {
		return errorCollector.AddCmdOutputToError(cmd, err)
	}

	stdOutBytes := stdoutOutput.Bytes()

	parsedOutput, skippedWarnings, err := parseJSONSkippingWarnings(stdOutBytes, result)
	if err == nil {
		if skippedWarnings {
			zap.S().Warnf("Command %v output may have contained a skipped warning from restic that was not valid JSON: %s", args, string(parsedOutput))
		}
		return nil
	}

	return errorCollector.AddCmdOutputToError(cmd, fmt.Errorf("command output is not valid JSON: %w", err))
}

func parseJSONSkippingWarnings(stdOutBytes []byte, result interface{}) ([]byte, bool, error) {
	firstErr := json.Unmarshal(stdOutBytes, result)
	if firstErr == nil {
		return stdOutBytes, false, nil
	}

	trimmed := bytes.TrimRightFunc(stdOutBytes, unicode.IsSpace)
	skipped := false
	for len(trimmed) > 0 {
		if err := json.Unmarshal(trimmed, result); err == nil {
			return trimmed, skipped, nil
		}

		newlineIdx := bytes.IndexByte(trimmed, '\n')
		if newlineIdx == -1 {
			break
		}
		trimmed = trimmed[newlineIdx+1:]
		skipped = true
	}

	return nil, skipped, firstErr
}

// Exists checks if the repository exists.
// Returns true if exists, false if it does not exist OR an access error occurred.
func (r *Repo) Exists(ctx context.Context, opts ...GenericOption) error {
	r.checkExists.Do(func() {
		output := bytes.NewBuffer(nil)
		errorCollector := errorMessageCollector{}
		cmd := r.commandWithContext(ctx, []string{"cat", "config"}, opts...)
		r.handleOutput(cmd, withAllTo(&errorCollector), withStdOutTo(output), withLogWriterFromContext(ctx))
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 10 {
				err = ErrRepoNotFound
			}
			r.exists = errorCollector.AddCmdOutputToError(cmd, err)
		} else if err := json.Unmarshal(output.Bytes(), &r.repoConfig); err != nil {
			r.exists = errorCollector.AddCmdOutputToError(cmd, fmt.Errorf("command output is not valid JSON: %w", err))
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
		errorCollector := errorMessageCollector{}
		r.handleOutput(cmd, withAllTo(&errorCollector), withStdOutTo(output), withLogWriterFromContext(ctx))

		if err := cmd.Run(); err != nil {
			if strings.Contains(output.String(), "config file already exists") || strings.Contains(output.String(), "already initialized") {
				r.initialized = errAlreadyInitialized
			} else {
				r.initialized = errorCollector.AddCmdOutputToError(cmd, err)
			}
		} else {
			if err := json.Unmarshal(output.Bytes(), &r.repoConfig); err != nil {
				r.initialized = errorCollector.AddCmdOutputToError(cmd, fmt.Errorf("command output is not valid JSON: %w", err))
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

func handleResticExitError(err error, failureErr error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 3 {
			return ErrPartialBackup
		}
		return fmt.Errorf("exit code %d: %w", exitErr.ExitCode(), failureErr)
	}
	return err
}

func runCommandWithProgress[T ProgressEntryValidator](ctx context.Context, r *Repo, args []string, callback func(T), failureErr error, opts ...GenericOption) (T, error) {
	logger := LoggerFromContext(ctx)
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmdCtx = ContextWithLogger(cmdCtx, nil) // ensure no logger is used
	cmd := r.commandWithContext(cmdCtx, args, opts...)

	// Ensure the command is logged since we're overriding the logger
	if logger != nil {
		fmt.Fprintf(logger, "command: %q\n", cmd)
	} else {
		logger = io.Discard
	}

	buf := buffer.New(8 * 1024) // 8KB IO buffer for the realtime event parsing
	reader, writer := nio.Pipe(buf)
	r.handleOutput(cmd, withAllTo(writer))

	var readErr error
	var summary T
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := processProgressOutput[T](reader, logger, callback)
		summary = result
		if err != nil {
			readErr = fmt.Errorf("output processing: %w", err)
		}
	}()

	cmdErr := cmd.Run()
	writer.Close()
	wg.Wait()

	if cmdErr != nil || readErr != nil {
		if cmdErr != nil {
			cmdErr = handleResticExitError(cmdErr, failureErr)
		}
		return summary, newCmdError(cmd, errors.Join(cmdErr, readErr))
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

	return runCommandWithProgress(ctx, r, args, progressCallback, ErrBackupFailed, opts...)
}

func (r *Repo) Restore(ctx context.Context, snapshot string, callback func(*RestoreProgressEntry), opts ...GenericOption) (*RestoreProgressEntry, error) {
	opts = append(slices.Clone(opts), WithEnv("RESTIC_PROGRESS_FPS=2"))
	args := []string{"restore", "--json", snapshot}

	return runCommandWithProgress(ctx, r, args, callback, ErrRestoreFailed, opts...)
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
	cmd := r.commandWithContext(ctx, args, opts...)
	errorCollector := errorMessageCollector{}
	r.handleOutput(cmd, withAllTo(&errorCollector), withLogWriterFromContext(ctx))
	if err := cmd.Run(); err != nil {
		return errorCollector.AddCmdOutputToError(cmd, err)
	}
	return nil
}

func (r *Repo) Dump(ctx context.Context, snapshotID string, file string, dumpOutput io.Writer, opts ...GenericOption) error {
	args := []string{"dump", snapshotID, file}
	if runtime.GOOS == "windows" {
		args = append(args, "--archive", "zip")
	} else {
		args = append(args, "--archive", "tar")
	}
	cmd := r.commandWithContext(ctx, args, opts...)
	logWriter := LoggerFromContext(ctx)
	if logWriter == nil {
		logWriter = io.Discard
	}
	errorCollector := errorMessageCollector{}

	// Dump writes binary output to stdout, we should only ever capture and print stderr
	r.handleOutput(cmd, withStdOutTo(dumpOutput), withStdErrTo(logWriter), withStdErrTo(&errorCollector))
	if err := cmd.Run(); err != nil {
		return errorCollector.AddCmdOutputToError(cmd, err)
	}

	return nil
}

func (r *Repo) Prune(ctx context.Context, pruneOutput io.Writer, opts ...GenericOption) error {
	return r.runSimpleCommand(ctx, []string{"prune"}, pruneOutput, opts...)
}

func (r *Repo) Check(ctx context.Context, checkOutput io.Writer, opts ...GenericOption) error {
	return r.runSimpleCommand(ctx, []string{"check"}, checkOutput, opts...)
}

// runSimpleCommand executes a command with optional output capture
func (r *Repo) runSimpleCommand(ctx context.Context, args []string, outputWriter io.Writer, opts ...GenericOption) error {
	cmd := r.commandWithContext(ctx, args, opts...)
	errorCollector := errorMessageCollector{}
	if outputWriter != nil {
		r.handleOutput(cmd, withStdOutTo(outputWriter), withAllTo(&errorCollector), withLogWriterFromContext(ctx))
	}
	if err := cmd.Run(); err != nil {
		return errorCollector.AddCmdOutputToError(cmd, err)
	}
	return nil
}

func (r *Repo) ListDirectory(ctx context.Context, snapshot string, path string, opts ...GenericOption) (*Snapshot, []*LsEntry, error) {
	if path == "" {
		// an empty path can trigger very expensive operations (e.g. iterates all files in the snapshot)
		return nil, nil, errors.New("path must not be empty")
	}

	cmd := r.commandWithContext(ctx, []string{"ls", "--json", snapshot, path}, opts...)
	errorCollector := errorMessageCollector{}
	output := bytes.NewBuffer(nil)
	r.handleOutput(cmd, withStdOutTo(output), withAllTo(&errorCollector), withLogWriterFromContext(ctx))
	if err := cmd.Run(); err != nil {
		return nil, nil, errorCollector.AddCmdOutputToError(cmd, fmt.Errorf("error running command: %w", err))
	}

	snap, entries, err := readLs(output)
	if err != nil {
		return nil, nil, errorCollector.AddCmdOutputToError(cmd, fmt.Errorf("error parsing JSON: %w", err))
	}
	return snap, entries, nil
}

func (r *Repo) Unlock(ctx context.Context, opts ...GenericOption) error {
	errorCollector := errorMessageCollector{}
	cmd := r.commandWithContext(ctx, []string{"unlock"}, opts...)
	r.handleOutput(cmd, withAllTo(&errorCollector), withLogWriterFromContext(ctx))
	if err := cmd.Run(); err != nil {
		return errorCollector.AddCmdOutputToError(cmd, err)
	}
	return nil
}

func (r *Repo) Stats(ctx context.Context, opts ...GenericOption) (*RepoStats, error) {
	var stats RepoStats
	err := r.executeWithJSONOutput(ctx, []string{"stats", "--json", "--mode=raw-data"}, &stats, opts...)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// AddTags adds tags to the specified snapshots.
func (r *Repo) AddTags(ctx context.Context, snapshotIDs []string, tags []string, opts ...GenericOption) error {
	args := []string{"tag"}
	args = append(args, "--add", strings.Join(tags, ","))
	args = append(args, snapshotIDs...)

	errorCollector := errorMessageCollector{}
	cmd := r.commandWithContext(ctx, args, opts...)
	r.handleOutput(cmd, withAllTo(&errorCollector), withLogWriterFromContext(ctx))
	if err := cmd.Run(); err != nil {
		return errorCollector.AddCmdOutputToError(cmd, err)
	}
	return nil
}

func (r *Repo) GenericCommand(ctx context.Context, args []string, opts ...GenericOption) error {
	cmd := r.commandWithContext(ctx, args, opts...)
	r.handleOutput(cmd, withLogWriterFromContext(ctx))
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
