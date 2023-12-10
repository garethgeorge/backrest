package restic

import (
	"fmt"
	"os/exec"
)

const outputBufferLimit = 1000

type CmdError struct {
	Command string
	Err     error
	Output  string
}

func (e *CmdError) Error() string {
	m := fmt.Sprintf("command %q failed: %s", e.Command, e.Err.Error())
	if e.Output != "" {
		m += "\nDetails: \n" + e.Output
	}
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
func newCmdError(cmd *exec.Cmd, output string, err error) *CmdError {
	cerr := &CmdError{
		Command: cmd.String(),
		Err:     err,
	}

	if len(output) >= outputBufferLimit {
		cerr.Output = output[:outputBufferLimit] + "\n...[truncated]"
	}

	return cerr
}

func newCmdErrorPreformatted(cmd *exec.Cmd, output string, err error) *CmdError {
	return &CmdError{
		Command: cmd.String(),
		Err:     err,
	}
}
