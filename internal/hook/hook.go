package hook

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
	"text/template"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

type Hook v1.Hook

type HookVars map[string]string

func (h *Hook) Do(event v1.Hook_Condition, vars map[string]string) error {
	if !slices.Contains(h.Conditions, event) {
		return nil
	}

	substs := make(map[string]string)
	for k, v := range vars {
		substs[k] = v
	}

	switch event {
	case v1.Hook_CONDITION_BACKUP_START:
		substs["EVENT"] = "backup start"
	case v1.Hook_CONDITION_BACKUP_END:
		substs["EVENT"] = "backup end"
	case v1.Hook_CONDITION_ERROR:
		substs["EVENT"] = "error"
	default:
		return fmt.Errorf("unknown hook event: %v", event)
	}

	switch action := h.Action.(type) {
	case *v1.Hook_ActionCommand:
		return h.doCommand(action, substs)
	default:
		return fmt.Errorf("unknown hook action: %v", action)
	}
}

func (h *Hook) doCommand(cmd *v1.Hook_ActionCommand, substs map[string]string) error {
	command := h.makeSubstitutions(cmd.ActionCommand.Command, substs)

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

	// Run the command in the specified shell
	execCmd := exec.Command(shell)
	execCmd.Stdin = strings.NewReader(command)
	reader, writer := io.Pipe()

	execCmd.Stderr = writer
	execCmd.Stdout = writer

	go func() {
		// read from the reader one line at a time until EOF
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			zap.S().Infof("hook output: %v", scanner.Text())
		}
	}()

	return execCmd.Run()
}

func (h *Hook) doDiscord(cmd *v1.Hook_ActionDiscord, substs map[string]string) error {
	return nil
}

func (h *Hook) makeSubstitutions(text string, substs map[string]string) string {
	template, err := template.New("command").Parse(text)
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	template.Execute(buf, substs)

	return buf.String()
}
