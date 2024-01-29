package hook

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func (h *Hook) doCommand(cmd *v1.Hook_ActionCommand, vars HookVars, output io.Writer) error {
	command, err := h.makeSubstitutions(cmd.ActionCommand.Command, vars)
	if err != nil {
		return fmt.Errorf("template formatting: %w", err)
	}

	// Parse out the shell to use if a #! prefix is present
	shell := "sh"
	if len(command) > 2 && command[0:2] == "#!" {
		nextLine := strings.Index(command, "\n")
		if nextLine == -1 {
			nextLine = len(command)
		}
		shell = strings.Trim(command[2:nextLine], " ")
		command = command[nextLine+1:]
	}

	output.Write([]byte(fmt.Sprintf("Running command:\n#! %v\n%v\n", shell, command)))

	// Run the command in the specified shell
	execCmd := exec.Command(shell)
	execCmd.Stdin = strings.NewReader(command)

	execCmd.Stderr = output
	execCmd.Stdout = output

	return execCmd.Run()
}
