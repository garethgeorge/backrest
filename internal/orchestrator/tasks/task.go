package tasks

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var NeverScheduledTask = ScheduledTask{}

const (
	PlanForUnassociatedOperations       = "_unassociated_"
	InstanceIDForUnassociatedOperations = "_unassociated_"
	PlanForSystemTasks                  = "_system_" // plan for system tasks e.g. garbage collection, prune, stats, etc.

	TaskPriorityStats          = 0
	TaskPriorityDefault        = 1 << 1 // default priority
	TaskPriorityForget         = 1 << 2
	TaskPriorityIndexSnapshots = 1 << 3
	TaskPriorityCheck          = 1 << 4 // check should always run after prune.
	TaskPriorityPrune          = 1 << 5
	TaskPriorityInteractive    = 1 << 6 // highest priority
)

// TaskRunner is an interface for running tasks. It is used by tasks to create operations and write logs.
type TaskRunner interface {
	// InstanceID returns the instance ID executing this task.
	InstanceID() string
	// CreateOperation creates the operation in storage and sets the operation ID in the task.
	CreateOperation(*v1.Operation) error
	// UpdateOperation updates the operation in storage. It must be called after CreateOperation.
	UpdateOperation(*v1.Operation) error
	// ExecuteHooks
	ExecuteHooks(ctx context.Context, events []v1.Hook_Condition, vars HookVars) error
	// OpLog returns the oplog for the operations.
	OpLog() *oplog.OpLog
	// GetRepo returns the repo with the given ID.
	GetRepo(repoID string) (*v1.Repo, error)
	// GetPlan returns the plan with the given ID.
	GetPlan(planID string) (*v1.Plan, error)
	// GetRepoOrchestrator returns the orchestrator for the repo with the given ID.
	GetRepoOrchestrator(repoID string) (*repo.RepoOrchestrator, error)
	// ScheduleTask schedules a task to run at a specific time.
	ScheduleTask(task Task, priority int) error
	// Config returns the current config.
	Config() *v1.Config
	// Logger returns the logger.
	Logger(ctx context.Context) *zap.Logger
	// LogrefWriter returns a writer that can be used to track streaming operation output.
	LogrefWriter() (id string, w io.WriteCloser, err error)
}

type TaskExecutor interface {
	RunTask(ctx context.Context, st ScheduledTask) error
}

// ScheduledTask is a task that is scheduled to run at a specific time.
type ScheduledTask struct {
	Task  Task          // the task to run
	RunAt time.Time     // the time at which the task should be run.
	Op    *v1.Operation // operation associated with this execution of the task.
}

func (s ScheduledTask) Eq(other ScheduledTask) bool {
	return s.Task == other.Task && s.RunAt.Equal(other.RunAt)
}

func (s ScheduledTask) Less(other ScheduledTask) bool {
	if s.RunAt.Equal(other.RunAt) {
		return s.Task.Name() < other.Task.Name()
	}
	return s.RunAt.Before(other.RunAt)
}

// Task is a task that can be scheduled to run at a specific time.
type Task interface {
	Name() string                                                       // human readable name for this task.
	Type() string                                                       // simple string 'type' for this task.
	Next(now time.Time, runner TaskRunner) (ScheduledTask, error)       // returns the next scheduled task.
	Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error // run the task.
	PlanID() string                                                     // the ID of the plan this task is associated with.
	RepoID() string                                                     // the ID of the repo this task is associated with.
	Repo() *v1.Repo                                                     // the repo this task is associated with.
}

type BaseTask struct {
	TaskType   string
	TaskName   string
	TaskPlanID string
	TaskRepo   *v1.Repo
}

func (b BaseTask) Type() string {
	return b.TaskType
}

func (b BaseTask) Name() string {
	return b.TaskName
}

func (b BaseTask) PlanID() string {
	return b.TaskPlanID
}

func (b BaseTask) RepoID() string {
	if b.TaskRepo == nil {
		return ""
	}
	return b.TaskRepo.Id
}

func (b BaseTask) Repo() *v1.Repo {
	return b.TaskRepo
}

type OneoffTask struct {
	BaseTask
	RunAt       time.Time
	FlowID      int64 // the ID of the flow this task is associated with.
	DidSchedule bool
	ProtoOp     *v1.Operation // the prototype operation for this class of task.
}

func (o *OneoffTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if o.DidSchedule {
		return NeverScheduledTask, nil
	}
	o.DidSchedule = true

	var op *v1.Operation
	if o.ProtoOp != nil {
		op = proto.Clone(o.ProtoOp).(*v1.Operation)
		op.RepoId = o.TaskRepo.Id
		op.PlanId = o.TaskPlanID
		op.RepoGuid = o.TaskRepo.Guid
		op.FlowId = o.FlowID
		op.UnixTimeStartMs = timeToUnixMillis(o.RunAt) // TODO: this should be updated before Run is called.
		op.Status = v1.OperationStatus_STATUS_PENDING
	}

	return ScheduledTask{
		RunAt: o.RunAt,
		Op:    op,
	}, nil
}

type GenericOneoffTask struct {
	OneoffTask
	Do func(ctx context.Context, st ScheduledTask, runner TaskRunner) error
}

func (g *GenericOneoffTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	return g.Do(ctx, st, runner)
}

func timeToUnixMillis(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond()/1000000)
}

func curTimeMillis() int64 {
	return timeToUnixMillis(time.Now())
}

type testTaskRunner struct {
	config *v1.Config // the config to use for the task runner.
	oplog  *oplog.OpLog
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

func (t *testTaskRunner) CreateOperation(op *v1.Operation) error {
	panic("not implemented")
}

func (t *testTaskRunner) UpdateOperation(op *v1.Operation) error {
	panic("not implemented")
}

func (t *testTaskRunner) ExecuteHooks(ctx context.Context, events []v1.Hook_Condition, vars HookVars) error {
	panic("not implemented")
}

func (t *testTaskRunner) OpLog() *oplog.OpLog {
	return t.oplog
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

func (t *testTaskRunner) GetRepoOrchestrator(repoID string) (*repo.RepoOrchestrator, error) {
	panic("not implemented")
}

func (t *testTaskRunner) ScheduleTask(task Task, priority int) error {
	panic("not implemented")
}

func (t *testTaskRunner) Config() *v1.Config {
	return t.config
}

func (t *testTaskRunner) Logger(ctx context.Context) *zap.Logger {
	return zap.L()
}

func (t *testTaskRunner) LogrefWriter() (id string, w io.WriteCloser, err error) {
	panic("not implemented")
}
