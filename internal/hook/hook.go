package hook

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	cfg "github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/hook/types"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

func TasksTriggeredByEvent(config *v1.Config, repoID string, planID string, parentOp *v1.Operation, events []v1.Hook_Condition, vars interface{}) ([]tasks.Task, error) {
	var taskSet []tasks.Task

	repo := cfg.FindRepo(config, repoID)
	if repo == nil {
		return nil, fmt.Errorf("repo %v not found", repoID)
	}
	plan := cfg.FindPlan(config, planID)
	if plan == nil && planID != "" {
		return nil, fmt.Errorf("plan %v not found", planID)
	}

	for idx, hook := range repo.GetHooks() {
		event := firstMatchingCondition(hook, events)
		if event == v1.Hook_CONDITION_UNKNOWN {
			continue
		}

		name := fmt.Sprintf("repo/%v/hook/%v", repo.Id, idx)
		task, err := newOneoffRunHookTask(name, config.Instance, repoID, planID, parentOp, time.Now(), hook, event, vars)
		if err != nil {
			return nil, err
		}
		taskSet = append(taskSet, task)
	}

	for idx, hook := range plan.GetHooks() {
		event := firstMatchingCondition(hook, events)
		if event == v1.Hook_CONDITION_UNKNOWN {
			continue
		}

		name := fmt.Sprintf("plan/%v/hook/%v", plan.Id, idx)
		task, err := newOneoffRunHookTask(name, config.Instance, repoID, planID, parentOp, time.Now(), hook, event, vars)
		if err != nil {
			return nil, err
		}
		taskSet = append(taskSet, task)
	}

	return taskSet, nil
}

func newOneoffRunHookTask(title, instanceID, repoID, planID string, parentOp *v1.Operation, at time.Time, hook *v1.Hook, event v1.Hook_Condition, vars interface{}) (tasks.Task, error) {
	h, err := types.DefaultRegistry().GetHandler(hook)
	if err != nil {
		return nil, fmt.Errorf("no handler for hook type %T", hook.Action)
	}

	title = h.Name() + " hook " + title

	return &tasks.GenericOneoffTask{
		OneoffTask: tasks.OneoffTask{
			BaseTask: tasks.BaseTask{
				TaskName:   fmt.Sprintf("run hook %v", title),
				TaskRepoID: repoID,
				TaskPlanID: planID,
			},
			FlowID: parentOp.GetFlowId(),
			RunAt:  at,
			ProtoOp: &v1.Operation{
				InstanceId: instanceID,
				RepoId:     repoID,
				PlanId:     planID,
				FlowId:     parentOp.GetFlowId(),

				DisplayMessage: fmt.Sprintf("running %v triggered by %v", title, event.String()),
				Op: &v1.Operation_OperationRunHook{
					OperationRunHook: &v1.OperationRunHook{
						Name:      title,
						Condition: event,
						ParentOp:  parentOp.GetId(),
					},
				},
			},
		},
		Do: func(ctx context.Context, st tasks.ScheduledTask, taskRunner tasks.TaskRunner) error {
			// TODO: this is a hack to get around the fact that vars is an interface{} .
			v := reflect.ValueOf(&vars).Elem()
			clone := reflect.New(v.Elem().Type()).Elem()
			clone.Set(v.Elem()) // copy vars to clone
			if field := v.Elem().FieldByName("Event"); field.IsValid() {
				clone.FieldByName("Event").Set(reflect.ValueOf(event))
			}

			if err := h.Execute(ctx, hook, clone, taskRunner); err != nil {
				err = applyHookErrorPolicy(hook.OnError, err)
				return err
			}
			return nil
		},
	}, nil
}

func firstMatchingCondition(hook *v1.Hook, events []v1.Hook_Condition) v1.Hook_Condition {
	for _, event := range events {
		if slices.Contains(hook.Conditions, event) {
			return event
		}
	}
	return v1.Hook_CONDITION_UNKNOWN
}

func curTimeMs() int64 {
	return time.Now().UnixNano() / 1000000
}

type Hook v1.Hook

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

// IsHaltingError returns true if the error is a fatal error or a request to cancel the operation
func IsHaltingError(err error) bool {
	var fatalErr *HookErrorFatal
	var cancelErr *HookErrorRequestCancel
	return errors.As(err, &fatalErr) || errors.As(err, &cancelErr)
}
