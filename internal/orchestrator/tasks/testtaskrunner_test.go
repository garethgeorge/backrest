package tasks

import (
	"context"
	"errors"
	"io"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
)

type testTaskRunner struct {
	config *v1.Config
	oplog  *oplog.OpLog

	// Configurable for Run() testing
	orchestrator   RepoOrchestrator
	hookCalls      []hookCall
	scheduledTasks []scheduledTaskCall
	onExecuteHooks func(ctx context.Context, events []v1.Hook_Condition, vars HookVars) error
}

type hookCall struct {
	Events []v1.Hook_Condition
	Vars   HookVars
}

type scheduledTaskCall struct {
	Task     Task
	Priority int
}

var _ TaskRunner = &testTaskRunner{}

func newTestTaskRunner(_ testing.TB, config *v1.Config, oplog *oplog.OpLog) *testTaskRunner {
	return &testTaskRunner{
		config: config,
		oplog:  oplog,
	}
}

func (t *testTaskRunner) InstanceID() string {
	return t.config.Instance
}

func (t *testTaskRunner) GetOperation(id int64) (*v1.Operation, error) {
	return t.oplog.Get(id)
}

func (t *testTaskRunner) CreateOperation(op ...*v1.Operation) error {
	for _, o := range op {
		if o.InstanceId != "" {
			continue
		}
		o.InstanceId = t.InstanceID()
	}
	return t.oplog.Add(op...)
}

func (t *testTaskRunner) UpdateOperation(op ...*v1.Operation) error {
	for _, o := range op {
		if o.InstanceId != "" {
			continue
		}
		o.InstanceId = t.InstanceID()
	}
	return t.oplog.Update(op...)
}

func (t *testTaskRunner) DeleteOperation(id ...int64) error {
	return t.oplog.Delete(id...)
}

func (t *testTaskRunner) ExecuteHooks(ctx context.Context, events []v1.Hook_Condition, vars HookVars) error {
	t.hookCalls = append(t.hookCalls, hookCall{Events: events, Vars: vars})
	if t.onExecuteHooks != nil {
		return t.onExecuteHooks(ctx, events, vars)
	}
	return nil
}

func (t *testTaskRunner) QueryOperations(q oplog.Query, fn func(*v1.Operation) error) error {
	return t.oplog.Query(q, fn)
}

func (t *testTaskRunner) GetRepo(repoID string) (*v1.Repo, error) {
	cfg := config.FindRepo(t.config, repoID)
	if cfg == nil {
		return nil, errors.New("repo not found")
	}
	return cfg, nil
}

func (t *testTaskRunner) GetPlan(planID string) (*v1.Plan, error) {
	cfg := config.FindPlan(t.config, planID)
	if cfg == nil {
		return nil, errors.New("plan not found")
	}
	return cfg, nil
}

func (t *testTaskRunner) GetRepoOrchestrator(repoID string) (RepoOrchestrator, error) {
	if t.orchestrator == nil {
		return nil, errors.New("no repo orchestrator configured")
	}
	return t.orchestrator, nil
}

func (t *testTaskRunner) ScheduleTask(task Task, priority int) error {
	t.scheduledTasks = append(t.scheduledTasks, scheduledTaskCall{Task: task, Priority: priority})
	return nil
}

func (t *testTaskRunner) Config() *v1.Config {
	return t.config
}

func (t *testTaskRunner) Logger(ctx context.Context) *zap.Logger {
	return zap.L()
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func (t *testTaskRunner) LogrefWriter() (id string, w io.WriteCloser, err error) {
	return "test-logref", &nopWriteCloser{io.Discard}, nil
}
