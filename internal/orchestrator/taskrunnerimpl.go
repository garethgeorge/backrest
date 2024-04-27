package orchestrator

import (
	"bytes"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// taskRunnerImpl is an implementation of TaskRunner for the default orchestrator.
type taskRunnerImpl struct {
	orchestrator *Orchestrator
	t            tasks.Task
	logs         bytes.Buffer
	logsWriter   ioutil.LockedWriter
	repo         *v1.Repo // cache, populated on first call to Repo()
	plan         *v1.Plan // cache, populated on first call to Plan()
}

var _ tasks.TaskRunner = &taskRunnerImpl{}

func (t *taskRunnerImpl) FindRepo() (*v1.Repo, error) {
	if t.repo != nil {
		return t.repo, nil
	}
	var err error
	t.repo, err = t.orchestrator.GetRepo(t.t.RepoID())
	return t.repo, err
}

func (t *taskRunnerImpl) FindPlan() (*v1.Plan, error) {
	if t.plan != nil {
		return t.plan, nil
	}
	var err error
	t.plan, err = t.orchestrator.GetPlan(t.t.PlanID())
	return t.plan, err
}

func newTaskRunnerImpl(orchestrator *Orchestrator, task tasks.Task) *taskRunnerImpl {
	impl := &taskRunnerImpl{
		orchestrator: orchestrator,
		t:            task,
	}
	impl.logsWriter = ioutil.LockedWriter{W: &impl.logs}
	return impl
}

func (t *taskRunnerImpl) CreateOperation(op *v1.Operation) error {
	return t.orchestrator.OpLog.Add(op)
}

func (t *taskRunnerImpl) UpdateOperation(op *v1.Operation) error {
	return t.orchestrator.OpLog.Update(op)
}

// Logger returns a logger for the run of the task.
// It will log to the default logger and a file for this task.
func (t *taskRunnerImpl) Logger() *zap.Logger {
	p := zap.NewProductionEncoderConfig()
	p.EncodeTime = zapcore.ISO8601TimeEncoder
	fe := zapcore.NewJSONEncoder(p)
	l := zap.New(zapcore.NewTee(
		zap.L().Core(),
		zapcore.NewCore(fe, zapcore.AddSync(&t.logsWriter), zapcore.DebugLevel),
	))
	return l.Named(t.t.Name())
}

func (t *taskRunnerImpl) AppendRawLog(data []byte) error {
	_, err := t.logsWriter.Write(data)
	return err
}

func (t *taskRunnerImpl) Orchestrator() *Orchestrator {
	return t.orchestrator
}

func (t *taskRunnerImpl) OpLog() *oplog.OpLog {
	return t.orchestrator.OpLog
}

func (t *taskRunnerImpl) ExecuteHooks(events []v1.Hook_Condition, vars hook.HookVars) error {
	repoID := t.t.RepoID()
	planID := t.t.PlanID()
	var repo *v1.Repo
	var plan *v1.Plan
	if repoID == "" {
		var err error
		repo, err = t.FindRepo()
		if err != nil {
			return err
		}
	}
	if planID == "" {
		var err error
		plan, err = t.FindPlan()
		if err != nil {
			return err
		}
	}
	return t.orchestrator.hookExecutor.ExecuteHooks(repo, plan, events, vars)
}

func (t *taskRunnerImpl) GetRepo(repoID string) (*v1.Repo, error) {
	return t.orchestrator.GetRepo(repoID)
}

func (t *taskRunnerImpl) GetPlan(planID string) (*v1.Plan, error) {
	return t.orchestrator.GetPlan(planID)
}

func (t *taskRunnerImpl) GetRepoOrchestrator(repoID string) (*repo.RepoOrchestrator, error) {
	return t.orchestrator.GetRepoOrchestrator(repoID)
}

func (t *taskRunnerImpl) ScheduleTask(task tasks.Task, priority int) error {
	return t.orchestrator.ScheduleTask(task, priority)
}
