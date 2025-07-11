package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// taskRunnerImpl is an implementation of TaskRunner for the default orchestrator.
type taskRunnerImpl struct {
	orchestrator *Orchestrator
	t            tasks.Task
	op           *v1.Operation
	plan         *v1.Plan   // cache, populated on first call to Plan()
	config       *v1.Config // cache, populated on first call to Config()
}

var _ tasks.TaskRunner = &taskRunnerImpl{}

func newTaskRunnerImpl(orchestrator *Orchestrator, task tasks.Task, op *v1.Operation) *taskRunnerImpl {
	return &taskRunnerImpl{
		config:       orchestrator.config,
		orchestrator: orchestrator,
		t:            task,
		op:           op,
	}
}

func (t *taskRunnerImpl) findPlan() (*v1.Plan, error) {
	if t.plan != nil {
		return t.plan, nil
	}
	var err error
	t.plan, err = t.orchestrator.GetPlan(t.t.PlanID())
	return t.plan, err
}

func (t *taskRunnerImpl) InstanceID() string {
	return t.config.Instance
}

func (t *taskRunnerImpl) GetOperation(id int64) (*v1.Operation, error) {
	return t.orchestrator.OpLog.Get(id)
}

func (t *taskRunnerImpl) CreateOperation(op ...*v1.Operation) error {
	for _, o := range op {
		if o.InstanceId != "" {
			continue
		}
		o.InstanceId = t.InstanceID()
	}
	return t.orchestrator.OpLog.Add(op...)
}

func (t *taskRunnerImpl) UpdateOperation(op ...*v1.Operation) error {
	for _, o := range op {
		if o.InstanceId != "" {
			continue
		}
		o.InstanceId = t.InstanceID()
	}
	return t.orchestrator.OpLog.Update(op...)
}

func (t *taskRunnerImpl) DeleteOperation(id ...int64) error {
	return t.orchestrator.OpLog.Delete(id...)
}

func (t *taskRunnerImpl) Orchestrator() *Orchestrator {
	return t.orchestrator
}

func (t *taskRunnerImpl) QueryOperations(q oplog.Query, fn func(*v1.Operation) error) error {
	return t.orchestrator.OpLog.Query(q, fn)
}

func (t *taskRunnerImpl) ExecuteHooks(ctx context.Context, events []v1.Hook_Condition, vars tasks.HookVars) error {
	vars.Task = t.t.Name()
	if t.op != nil {
		vars.Duration = time.Since(time.UnixMilli(t.op.UnixTimeStartMs))
	}

	vars.CurTime = time.Now()

	repoID := t.t.RepoID()
	planID := t.t.PlanID()
	var plan *v1.Plan
	if repo := t.t.Repo(); repo != nil {
		vars.Repo = repo
	}
	if planID != "" {
		plan, _ = t.findPlan()
		vars.Plan = plan
	}
	if vars.Plan == nil {
		vars.Plan = &v1.Plan{
			Id: t.t.PlanID(), // make a fake plan that conveys only the ID, e.g. for unassociated operations OR system plan.
		}
	}

	hookTasks, err := hook.TasksTriggeredByEvent(t.Config(), repoID, planID, t.op, events, vars)
	if err != nil {
		return err
	}

	for _, task := range hookTasks {
		st, err := t.orchestrator.CreateUnscheduledTask(task, tasks.TaskPriorityDefault, time.Now())
		if err != nil {
			return fmt.Errorf("creating task for hook: %w", err)
		}
		if err := t.orchestrator.RunTask(ctx, st); hook.IsHaltingError(err) {
			var cancelErr *hook.HookErrorRequestCancel
			var retryErr *hook.HookErrorRetry
			if errors.As(err, &cancelErr) {
				return fmt.Errorf("%v: %w: %w", task.Name(), &tasks.TaskCancelledError{}, cancelErr.Err)
			} else if errors.As(err, &retryErr) {
				return fmt.Errorf("%v: %w", task.Name(), &tasks.TaskRetryError{
					Err:     retryErr.Err,
					Backoff: retryErr.Backoff,
				})
			}
			return fmt.Errorf("%v: %w", task.Name(), err)
		}
	}
	return nil
}

func (t *taskRunnerImpl) GetRepo(repoID string) (*v1.Repo, error) {
	if repoID == t.t.RepoID() {
		return t.t.Repo(), nil
	}
	return t.orchestrator.GetRepo(repoID)
}

func (t *taskRunnerImpl) GetPlan(planID string) (*v1.Plan, error) {
	if planID == t.t.PlanID() {
		return t.findPlan() // optimization for the common case of the current plan
	}
	return t.orchestrator.GetPlan(planID)
}

func (t *taskRunnerImpl) GetRepoOrchestrator(repoID string) (*repo.RepoOrchestrator, error) {
	return t.orchestrator.GetRepoOrchestrator(repoID)
}

func (t *taskRunnerImpl) ScheduleTask(task tasks.Task, priority int) error {
	return t.orchestrator.ScheduleTask(task, priority)
}

func (t *taskRunnerImpl) Config() *v1.Config {
	if t.config != nil {
		return t.config
	}
	t.config = t.orchestrator.Config()
	return t.config
}

func (t *taskRunnerImpl) Logger(ctx context.Context) *zap.Logger {
	return logging.Logger(ctx, "[tasklog] ").Named(t.t.Name())
}

func (t *taskRunnerImpl) LogrefWriter() (string, io.WriteCloser, error) {
	logID := "c-" + uuid.New().String()
	writer, err := t.orchestrator.logStore.Create(logID, t.op.GetId(), time.Duration(0))
	return logID, writer, err
}
