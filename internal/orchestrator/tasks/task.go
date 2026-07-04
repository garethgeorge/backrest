package tasks

import (
	"context"
	"io"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// RepoOrchestrator is the interface for repo operations that tasks depend on.
// The concrete implementation is in the repo package.
type RepoOrchestrator interface {
	UnlockIfAutoEnabled(ctx context.Context) error
	Backup(ctx context.Context, plan *v1.Plan, dryRun bool, progressCallback func(event *restic.BackupProgressEntry)) (*restic.BackupProgressEntry, error)
	Forget(ctx context.Context, policy *v1.RetentionPolicy, opts ...restic.GenericOption) ([]*v1.ResticSnapshot, error)
	ForgetSnapshot(ctx context.Context, snapshotId string) error
	Prune(ctx context.Context, output io.Writer) error
	Check(ctx context.Context, output io.Writer) error
	Stats(ctx context.Context) (*v1.RepoStats, error)
	Restore(ctx context.Context, snapshotId string, snapshotPath string, target string, progressCallback func(event *v1.RestoreProgressEntry)) (*v1.RestoreProgressEntry, error)
	Snapshots(ctx context.Context) ([]*restic.Snapshot, error)
	AddTags(ctx context.Context, snapshotIDs []string, tags []string) error
	RunCommand(ctx context.Context, command string, writer io.Writer) error
}

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
	// GetOperation returns the operation with the given ID.
	GetOperation(id int64) (*v1.Operation, error)
	// CreateOperation creates the operation in storage and sets the operation ID in the task.
	CreateOperation(...*v1.Operation) error
	// UpdateOperation updates the operation in storage. It must be called after CreateOperation.
	UpdateOperation(...*v1.Operation) error
	// DeleteOperation deletes the operation from storage.
	DeleteOperation(...int64) error
	// QueryOperations queries the operation log.
	QueryOperations(oplog.Query, func(*v1.Operation) error) error
	// ExecuteHooks
	ExecuteHooks(ctx context.Context, events []v1.Hook_Condition, vars HookVars) error
	// GetRepo returns the repo with the given ID.
	GetRepo(repoID string) (*v1.Repo, error)
	// GetPlan returns the plan with the given ID.
	GetPlan(planID string) (*v1.Plan, error)
	// GetRepoOrchestrator returns the orchestrator for the repo with the given ID.
	GetRepoOrchestrator(repoID string) (RepoOrchestrator, error)
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

type GenericOneoffTask struct {
	BaseTask
	RunAt       time.Time
	FlowID      int64 // the ID of the flow this task is associated with.
	DidSchedule bool
	ProtoOp     *v1.Operation // the prototype operation for this class of task.
	Do          func(ctx context.Context, st ScheduledTask, runner TaskRunner) error
}

func (o *GenericOneoffTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if o.DidSchedule {
		return NeverScheduledTask, nil
	}
	o.DidSchedule = true

	var op *v1.Operation
	if o.ProtoOp != nil {
		op = proto.Clone(o.ProtoOp).(*v1.Operation)
		op.FlowId = o.FlowID
	}

	return ScheduledTask{
		RunAt: o.RunAt,
		Op:    op,
	}, nil
}

func (g *GenericOneoffTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	return g.Do(ctx, st, runner)
}
