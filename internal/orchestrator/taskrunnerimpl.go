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
	repo         *v1.Repo   // cache, populated on first call to Repo()
	plan         *v1.Plan   // cache, populated on first call to Plan()
	config       *v1.Config // cache, populated on first call to Config()
}

var _ tasks.TaskRunner = &taskRunnerImpl{}

func newTaskRunnerImpl(orchestrator *Orchestrator, task tasks.Task, op *v1.Operation) *taskRunnerImpl {
	return &taskRunnerImpl{
		orchestrator: orchestrator,
		t:            task,
		op:           op,
	}
}

func (t *taskRunnerImpl) findRepo() (*v1.Repo, error) {
	if t.repo != nil {
		return t.repo, nil
	}
	var err error
	t.repo, err = t.orchestrator.GetRepo(t.t.RepoID())
	return t.repo, err
}

func (t *taskRunnerImpl) findPlan() (*v1.Plan, error) {
	if t.plan != nil {
		return t.plan, nil
	}
	var err error
	t.plan, err = t.orchestrator.GetPlan(t.t.PlanID())
	return t.plan, err
}

func (t *taskRunnerImpl) CreateOperation(op *v1.Operation) error {
	op.InstanceId = t.orchestrator.config.Instance
	return t.orchestrator.OpLog.Add(op)
}

func (t *taskRunnerImpl) UpdateOperation(op *v1.Operation) error {
	op.InstanceId = t.orchestrator.config.Instance
	return t.orchestrator.OpLog.Update(op)
}

func (t *taskRunnerImpl) Orchestrator() *Orchestrator {
	return t.orchestrator
}

func (t *taskRunnerImpl) OpLog() *oplog.OpLog {
	return t.orchestrator.OpLog
}

func (t *taskRunnerImpl) ExecuteHooks(ctx context.Context, events []v1.Hook_Condition, vars tasks.HookVars) error {
	vars.Task = t.t.Name()
	if t.op != nil {
		vars.Duration = time.Since(time.UnixMilli(t.op.UnixTimeStartMs))
	}

	vars.CurTime = time.Now()

	repoID := t.t.RepoID()
	planID := t.t.PlanID()
	var repo *v1.Repo
	var plan *v1.Plan
	if repoID != "" {
		var err error
		repo, err = t.findRepo()
		if err != nil {
			return err
		}
		vars.Repo = repo
	}
	if planID != "" {
		plan, _ = t.findPlan()
		vars.Plan = plan
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
		return t.findRepo() // optimization for the common case of the current repo
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
	logID := uuid.New().String()
	writer, err := t.orchestrator.logStore.Create(logID, t.op.GetId(), time.Duration(0))
	return logID, writer, err
}
