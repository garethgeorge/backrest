package types

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/garethgeorge/backrest/internal/platformutil"
	"github.com/google/shlex"
)

type commandHandler struct{}

func (commandHandler) Name() string {
	return "command"
}

func (commandHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	command, err := hookutil.RenderTemplate(h.GetActionCommand().GetCommand(), vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	writer := logging.WriterFromContext(ctx)

	// Parse out the shell to use if a #! prefix is present
	shell := []string{"sh"}
	if runtime.GOOS == "windows" {
		shell = []string{"powershell", "-NoLogo", "-NoProfile", "-Command", "-"}
	}

	if len(command) > 2 && command[0:2] == "#!" {
		nextLine := strings.Index(command, "\n")
		if nextLine == -1 {
			nextLine = len(command)
		}
		shell, err = shlex.Split(strings.Trim(command[2:nextLine], " "))
		if err != nil {
			return fmt.Errorf("parsing shell for command: %w", err)
		} else if len(shell) == 0 {
			return errors.New("must specify shell for command")
		}
		command = command[nextLine+1:]
	}

	scriptWriter := &ioutil.LinePrefixer{W: writer, Prefix: []byte("[script] ")}
	fmt.Fprintf(scriptWriter, "%v\n%v\n", shell, command)
	scriptWriter.Close()
	outputWriter := &ioutil.LinePrefixer{W: writer, Prefix: []byte("[output] ")}
	defer outputWriter.Close()

	// Run the command in the specified shell
	execCmd := exec.Command(shell[0], shell[1:]...)
	platformutil.SetPlatformOptions(execCmd)
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
