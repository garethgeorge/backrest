package hook

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
	"text/template"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"go.uber.org/zap"
)

// ExecuteHooks schedules tasks for the hooks subscribed to the given event. The vars map is used to substitute variables
// Hooks are pulled both from the provided plan and from the repo config.
func ExecuteHooks(orch *orchestrator.Orchestrator, plan *v1.Plan, linkSnapshot string, event v1.Hook_Condition, vars map[string]string) {
	repo, err := orch.GetRepo(plan.Repo)
	if err != nil {
		zap.S().Errorf("execute hooks: get repo %q: %v", plan.Repo, err)
		return
	}

	repoCfg := repo.Config()

	for idx, hook := range repoCfg.Hooks {
		if !slices.Contains(hook.Conditions, event) {
			continue
		}

		operation := v1.Operation{
			UnixTimeStartMs: curTimeMs(),
			Status:          v1.OperationStatus_STATUS_INPROGRESS,
			PlanId:          plan.Id,
			RepoId:          plan.Repo,
			SnapshotId:      linkSnapshot,
			Op: &v1.Operation_OperationRunHook{
				OperationRunHook: &v1.OperationRunHook{
					HookDesc: fmt.Sprintf("repo/%v/hook/%v", repoCfg.Id, idx),
				},
			},
		}

		executeHook(orch, &operation, (*Hook)(hook), event, vars)
	}

	for idx, hook := range plan.Hooks {
		if !slices.Contains(hook.Conditions, event) {
			continue
		}

		operation := v1.Operation{
			UnixTimeStartMs: curTimeMs(),
			Status:          v1.OperationStatus_STATUS_INPROGRESS,
			PlanId:          plan.Id,
			RepoId:          plan.Repo,
			SnapshotId:      linkSnapshot,
			Op: &v1.Operation_OperationRunHook{
				OperationRunHook: &v1.OperationRunHook{
					HookDesc: fmt.Sprintf("plan/%v/hook/%v", plan.Id, idx),
				},
			},
		}

		executeHook(orch, &operation, (*Hook)(hook), event, vars)
	}
}

func executeHook(orch *orchestrator.Orchestrator, op *v1.Operation, hook *Hook, events v1.Hook_Condition, vars map[string]string) {
	if err := orch.OpLog.Add(op); err != nil {
		zap.S().Errorf("execute hook: add operation: %v", err)
		return
	}

	output := &bytes.Buffer{}

	if err := hook.Do(v1.Hook_CONDITION_SNAPSHOT_START, vars, output); err != nil {
		output.Write([]byte(fmt.Sprintf("Error: %v", err)))
		op.DisplayMessage = output.String()
		op.Status = v1.OperationStatus_STATUS_ERROR
	} else {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}

	op.UnixTimeEndMs = curTimeMs()

	if err := orch.OpLog.Update(op); err != nil {
		zap.S().Errorf("execute hook: update operation: %v", err)
		return
	}
}

func curTimeMs() int64 {
	return time.Now().UnixNano() / 1000000
}

type Hook v1.Hook

type HookVars map[string]string

func (h *Hook) Do(event v1.Hook_Condition, vars map[string]string, output io.Writer) error {
	if !slices.Contains(h.Conditions, event) {
		return nil
	}

	substs := make(map[string]string)
	for k, v := range vars {
		substs[k] = v
	}
	installTemplateFuncs(substs)

	switch action := h.Action.(type) {
	case *v1.Hook_ActionCommand:
		return h.doCommand(action, substs, output)
	default:
		return fmt.Errorf("unknown hook action: %v", action)
	}
}

func (h *Hook) doCommand(cmd *v1.Hook_ActionCommand, substs map[string]string, output io.Writer) error {
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

	output.Write([]byte(fmt.Sprintf("Running command:\n#! %v\n%v", shell, command)))

	// Run the command in the specified shell
	execCmd := exec.Command(shell)
	execCmd.Stdin = strings.NewReader(command)

	execCmd.Stderr = output
	execCmd.Stdout = output

	return execCmd.Run()
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

func installTemplateFuncs(vars map[string]string) template.FuncMap {
	return template.FuncMap{
		"EventName": func(cond v1.Hook_Condition) string {
			switch cond {
			case v1.Hook_CONDITION_SNAPSHOT_START:
				return "snapshot start"
			case v1.Hook_CONDITION_SNAPSHOT_END:
				return "snapshot end"
			case v1.Hook_CONDITION_ANY_ERROR:
				return "error"
			case v1.Hook_CONDITION_SNAPSHOT_ERROR:
				return "snapshot error"
			default:
				return "unknown"
			}
		},
		"IsError": func(cond v1.Hook_Condition) bool {
			return cond == v1.Hook_CONDITION_ANY_ERROR || cond == v1.Hook_CONDITION_SNAPSHOT_ERROR
		},
	}
}
