package restic

import (
	"context"
	"fmt"
	"os/exec"
)

const outputBufferLimit = 1000

type CmdError struct {
	Command string
	Err     error
}

func (e *CmdError) Error() string {
	m := fmt.Sprintf("command %q failed: %s", e.Command, e.Err.Error())
	return m
}

func (e *CmdError) Unwrap() error {
	return e.Err
}

func (e *CmdError) Is(target error) bool {
	_, ok := target.(*CmdError)
	return ok
}

// newCmdError creates a new error indicating that running a command failed.
func newCmdError(ctx context.Context, cmd *exec.Cmd, err error) *CmdError {
	cerr := &CmdError{
		Command: cmd.String(),
		Err:     err,
	}

	if logger := LoggerFromContext(ctx); logger != nil {
		logger.Write([]byte(cerr.Error()))
	}
	return cerr
}

type ErrorWithOutput struct {
	Err    error
	Output string
}

func (e *ErrorWithOutput) Error() string {
	return fmt.Sprintf("%v\nOutput:\n%s", e.Err, e.Output)
}

func (e *ErrorWithOutput) Unwrap() error {
	return e.Err
}

func (e *ErrorWithOutput) Is(target error) bool {
	_, ok := target.(*ErrorWithOutput)
	return ok
}

// newErrorWithOutput creates a new error with the given output.
func newErrorWithOutput(err error, output string) error {
	if output == "" {
		return err
	}

	if len(output) > outputBufferLimit {
		output = output[:outputBufferLimit] + fmt.Sprintf("\n... %d bytes truncated ...\n", len(output)-outputBufferLimit)
	}

	return &ErrorWithOutput{
		Err:    err,
		Output: output,
	}
}
