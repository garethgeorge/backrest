package restic

import (
	"fmt"
	"os/exec"
)

type CmdError struct {
	Command string
	Err error
	Output string
}

func (e *CmdError) Error() string {
	m := fmt.Sprintf("command %s failed: %s", e.Command, e.Err.Error())
	if e.Output != "" {
		m += "\nDetails: \n" + e.Output
	}
	return m
}

func (e *CmdError) Unwrap() error {
	return e.Err
}

// NewCmdError creates a new error indicating that running a command failed.
func NewCmdError(cmd *exec.Cmd, output []byte, err error) *CmdError {
	cerr := &CmdError{
		Command: cmd.String(),
		Err: err,
	}

	if len(output) > 0 {
		if len(output) > 1000 {
			output = output[:1000]
		}
		cerr.Output = string(output)
	}
	return cerr
}