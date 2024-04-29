package hook

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/template"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var (
	defaultTemplate = `{{ .Summary }}`
)

type HookExecutor struct {
	oplog    *oplog.OpLog
	logStore *rotatinglog.RotatingLog
}

func NewHookExecutor(oplog *oplog.OpLog, bigOutputStore *rotatinglog.RotatingLog) *HookExecutor {
	return &HookExecutor{
		oplog:    oplog,
		logStore: bigOutputStore,
	}
}

// ExecuteHooks schedules tasks for the hooks subscribed to the given event. The vars map is used to substitute variables
// Hooks are pulled both from the provided plan and from the repo config.
func (e *HookExecutor) ExecuteHooks(flowID int64, repo *v1.Repo, plan *v1.Plan, events []v1.Hook_Condition, vars HookVars) error {
	operationBase := v1.Operation{
		Status: v1.OperationStatus_STATUS_INPROGRESS,
		PlanId: plan.GetId(),
		RepoId: repo.GetId(),
		FlowId: flowID,
	}

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
		operation.DisplayMessage = "running " + name
		operation.UnixTimeStartMs = curTimeMs()
		operation.Op = &v1.Operation_OperationRunHook{
			OperationRunHook: &v1.OperationRunHook{
				Name: name,
			},
		}
		zap.L().Info("running hook", zap.String("plan", plan.Id), zap.Int64("opId", operation.Id), zap.String("hook", name))
		if err := e.executeHook(operation, h, event, vars); err != nil {
			zap.S().Errorf("error on repo hook %v on condition %v: %v", idx, event.String(), err)
			if isHaltingError(err) {
				return fmt.Errorf("repo hook %v on condition %v: %w", idx, event.String(), err)
			}
		}
	}

	for idx, hook := range plan.GetHooks() {
		h := (*Hook)(hook)
		event := firstMatchingCondition(h, events)
		if event == v1.Hook_CONDITION_UNKNOWN {
			continue
		}

		name := fmt.Sprintf("plan/%v/hook/%v", plan.Id, idx)
		operation := proto.Clone(&operationBase).(*v1.Operation)
		operation.DisplayMessage = "running " + name
		operation.UnixTimeStartMs = curTimeMs()
		operation.Op = &v1.Operation_OperationRunHook{
			OperationRunHook: &v1.OperationRunHook{
				Name: name,
			},
		}

		zap.L().Info("running hook", zap.String("plan", plan.Id), zap.Int64("opId", operation.Id), zap.String("hook", name))
		if err := e.executeHook(operation, h, event, vars); err != nil {
			zap.S().Errorf("error on plan hook %v on condition %v: %v", idx, event.String(), err)
			if isHaltingError(err) {
				return fmt.Errorf("plan hook %v on condition %v: %w", idx, event.String(), err)
			}
		}
	}
	return nil
}

func firstMatchingCondition(hook *Hook, events []v1.Hook_Condition) v1.Hook_Condition {
	for _, event := range events {
		if slices.Contains(hook.Conditions, event) {
			return event
		}
	}
	return v1.Hook_CONDITION_UNKNOWN
}

func (e *HookExecutor) executeHook(op *v1.Operation, hook *Hook, event v1.Hook_Condition, vars HookVars) error {
	if err := e.oplog.Add(op); err != nil {
		zap.S().Errorf("execute hook: add operation: %v", err)
		return errors.New("couldn't create operation")
	}

	output := &bytes.Buffer{}
	fmt.Fprintf(output, "triggering condition: %v\n", event.String())

	var retErr error
	if err := hook.Do(event, vars, io.MultiWriter(output)); err != nil {
		output.Write([]byte(fmt.Sprintf("Error: %v", err)))
		err = applyHookErrorPolicy(hook.OnError, err)
		var cancelErr *HookErrorRequestCancel
		if errors.As(err, &cancelErr) {
			// if it was a cancel then it successfully indicated it's intent to the caller
			// no error should be displayed in the UI.
			op.Status = v1.OperationStatus_STATUS_SUCCESS
		} else {
			op.Status = v1.OperationStatus_STATUS_ERROR
		}
		retErr = err
	} else {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}

	outputRef, err := e.logStore.Write(output.Bytes())
	if err != nil {
		retErr = errors.Join(retErr, fmt.Errorf("write logstore: %w", err))
	}
	op.Logref = outputRef

	op.UnixTimeEndMs = curTimeMs()
	if err := e.oplog.Update(op); err != nil {
		retErr = errors.Join(retErr, fmt.Errorf("update oplog: %w", err))
	}
	return retErr
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
	case *v1.Hook_ActionDiscord:
		return h.doDiscord(action, vars, output)
	case *v1.Hook_ActionGotify:
		return h.doGotify(action, vars, output)
	case *v1.Hook_ActionSlack:
		return h.doSlack(action, vars, output)
	case *v1.Hook_ActionShoutrrr:
		return h.doShoutrrr(action, vars, output)
	default:
		return fmt.Errorf("unknown hook action: %v", action)
	}
}

func (h *Hook) renderTemplate(text string, vars HookVars) (string, error) {
	template, err := template.New("template").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := template.Execute(buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func (h *Hook) renderTemplateOrDefault(template string, defaultTmpl string, vars HookVars) (string, error) {
	if strings.Trim(template, " ") == "" {
		return h.renderTemplate(defaultTmpl, vars)
	}
	return h.renderTemplate(template, vars)
}

func applyHookErrorPolicy(onError v1.Hook_OnError, err error) error {
	if err == nil || errors.As(err, &HookErrorFatal{}) || errors.As(err, &HookErrorRequestCancel{}) {
		return err
	}

	if onError == v1.Hook_ON_ERROR_CANCEL {
		return &HookErrorRequestCancel{Err: err}
	} else if onError == v1.Hook_ON_ERROR_FATAL {
		return &HookErrorFatal{Err: err}
	}
	return err
}

// isHaltingError returns true if the error is a fatal error or a request to cancel the operation
func isHaltingError(err error) bool {
	var fatalErr *HookErrorFatal
	var cancelErr *HookErrorRequestCancel
	return errors.As(err, &fatalErr) || errors.As(err, &cancelErr)
}
