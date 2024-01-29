package hook

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"slices"
	"text/template"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ExecuteHooks schedules tasks for the hooks subscribed to the given event. The vars map is used to substitute variables
// Hooks are pulled both from the provided plan and from the repo config.
func ExecuteHooks(oplog *oplog.OpLog, repo *v1.Repo, plan *v1.Plan, snapshotId string, events []v1.Hook_Condition, vars HookVars) {
	operationBase := v1.Operation{
		Status:     v1.OperationStatus_STATUS_INPROGRESS,
		PlanId:     plan.Id,
		RepoId:     plan.Repo,
		SnapshotId: snapshotId,
	}

	vars.SnapshotId = snapshotId
	vars.Repo = repo
	vars.Plan = plan
	vars.CurTime = time.Now()

	for idx, hook := range repo.GetHooks() {
		h := (*Hook)(hook)
		event := firstMatchingCondition(h, events)
		if event == v1.Hook_CONDITION_UNKNOWN {
			continue
		}

		name := fmt.Sprintf("repo/%v/hook/%v", repo.Id, idx)
		operation := proto.Clone(&operationBase).(*v1.Operation)
		operation.UnixTimeStartMs = curTimeMs()
		operation.Op = &v1.Operation_OperationRunHook{
			OperationRunHook: &v1.OperationRunHook{
				Name: name,
			},
		}
		zap.L().Info("Running hook", zap.String("plan", plan.Id), zap.Int64("opId", operation.Id), zap.String("hook", name))
		executeHook(oplog, operation, h, event, vars)
	}

	for idx, hook := range plan.GetHooks() {
		h := (*Hook)(hook)
		event := firstMatchingCondition(h, events)
		if event == v1.Hook_CONDITION_UNKNOWN {
			continue
		}

		name := fmt.Sprintf("plan/%v/hook/%v", plan.Id, idx)
		operation := proto.Clone(&operationBase).(*v1.Operation)
		operation.UnixTimeStartMs = curTimeMs()
		operation.Op = &v1.Operation_OperationRunHook{
			OperationRunHook: &v1.OperationRunHook{
				Name: name,
			},
		}
		zap.L().Info("Running hook", zap.String("plan", plan.Id), zap.Int64("opId", operation.Id), zap.String("hook", name))
		executeHook(oplog, operation, (*Hook)(hook), event, vars)
	}
}

func firstMatchingCondition(hook *Hook, events []v1.Hook_Condition) v1.Hook_Condition {
	for _, event := range events {
		if slices.Contains(hook.Conditions, event) {
			return event
		}
	}
	return v1.Hook_CONDITION_UNKNOWN
}

func executeHook(oplog *oplog.OpLog, op *v1.Operation, hook *Hook, event v1.Hook_Condition, vars HookVars) {
	if err := oplog.Add(op); err != nil {
		zap.S().Errorf("execute hook: add operation: %v", err)
		return
	}

	output := &bytes.Buffer{}
	pr, pw := io.Pipe()
	go func() {
		defer pr.Close()
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			zap.S().Infof("hook output: %v", scanner.Text())
		}
	}()
	defer pw.Close()

	if err := hook.Do(event, vars, io.MultiWriter(output, pw)); err != nil {
		output.Write([]byte(fmt.Sprintf("Error: %v", err)))
		op.DisplayMessage = err.Error()
		op.Status = v1.OperationStatus_STATUS_ERROR
	} else {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}

	if err := oplog.SetBigData(op.Id, "hook.log", output.Bytes()); err != nil {
		zap.S().Errorf("execute hook: set big data %q: %v", "hook.log", err)
	}
	op.Op.(*v1.Operation_OperationRunHook).OperationRunHook.OutputRef = "hook.log"

	op.UnixTimeEndMs = curTimeMs()
	if err := oplog.Update(op); err != nil {
		zap.S().Errorf("execute hook: update operation: %v", err)
		return
	}
}

func curTimeMs() int64 {
	return time.Now().UnixNano() / 1000000
}

type Hook v1.Hook

func (h *Hook) Do(event v1.Hook_Condition, vars HookVars, output io.Writer) error {
	if !slices.Contains(h.Conditions, event) {
		return nil
	}

	vars.Event = event

	switch action := h.Action.(type) {
	case *v1.Hook_ActionCommand:
		return h.doCommand(action, vars, output)
	default:
		return fmt.Errorf("unknown hook action: %v", action)
	}
}

func (h *Hook) makeSubstitutions(text string, vars HookVars) (string, error) {
	template, err := template.New("command").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := template.Execute(buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
