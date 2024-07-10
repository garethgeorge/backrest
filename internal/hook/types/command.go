package types

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type commandHandler struct{}

func (commandHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	command, err := hookutil.RenderTemplate(h.GetActionCommand().GetCommand(), vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	writer := logging.WriterFromContext(ctx)

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

	scriptWriter := &ioutil.LinePrefixer{W: writer, Prefix: []byte("[script] ")}
	defer scriptWriter.Close()
	outputWriter := &ioutil.LinePrefixer{W: writer, Prefix: []byte("[output] ")}
	defer outputWriter.Close()
	fmt.Fprintf(scriptWriter, "------- script -------\n#! %v\n%v\n", shell, command)

	// Run the command in the specified shell
	execCmd := exec.Command(shell)
	execCmd.Stdin = strings.NewReader(command)

	stdout := &ioutil.SynchronizedWriter{W: outputWriter}
	execCmd.Stderr = stdout
	execCmd.Stdout = stdout

	return execCmd.Run()
}

func (commandHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionCommand{})
}

func init() {
	DefaultRegistry().RegisterHandler(&commandHandler{})
}
