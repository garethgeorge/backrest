package restic

import (
	"bytes"
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
func newCmdError(cmd *exec.Cmd, err error) *CmdError {
	shortCmd := cmd.String()
	if len(shortCmd) > 100 {
		shortCmd = shortCmd[:100] + "..."
	}

	cerr := &CmdError{
		Command: shortCmd,
		Err:     err,
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

type errorMessageCollector struct {
	Output       *bytes.Buffer
	DroppedBytes int
}

func (e *errorMessageCollector) Write(p []byte) (int, error) {
	if e.Output == nil {
		e.Output = &bytes.Buffer{}
	}
	if e.Output.Len() >= outputBufferLimit {
		e.DroppedBytes += len(p)
		return len(p), nil
	}
	return e.Output.Write(p)
}

func (e *errorMessageCollector) AddOutputToError(err error) error {
	if e.Output == nil {
		return err
	}
	if e.DroppedBytes > 0 {
		return &ErrorWithOutput{
			Err:    err,
			Output: fmt.Sprintf("%s\n... %d bytes truncated ...", e.Output.String(), e.DroppedBytes),
		}
	}
	return &ErrorWithOutput{
		Err:    err,
		Output: e.Output.String(),
	}
}

func (e *errorMessageCollector) AddCmdOutputToError(cmd *exec.Cmd, err error) error {
	return newCmdError(cmd, e.AddOutputToError(err))
}
