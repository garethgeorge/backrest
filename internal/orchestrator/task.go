package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var NeverScheduledTask = ScheduledTask{}

// TaskRunner is an interface for running tasks. It is used by tasks to create operations and write logs.
type TaskRunner interface {
	// CreateOperation creates the operation in storage and sets the operation ID in the task.
	CreateOperation(*v1.Operation) error
	// UpdateOperation updates the operation in storage. It must be called after CreateOperation.
	UpdateOperation(*v1.Operation) error
	// ExecuteHooks
	ExecuteHooks(events []v1.Hook_Condition, vars hook.HookVars) error
	// Logger returns a logger for the run of the task.
	Logger() *zap.Logger
	// AppendRawLog writes the raw log data to the log for this task.
	// this data will be handled in a way that is appropriate for large logs (e.g. stored in a file, not printed to stdout).
	AppendRawLog([]byte) error
	// Orchestrator returns the orchestrator that is running the task.
	Orchestrator() *Orchestrator
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

func (s ScheduledTask) cancel(oplog *oplog.OpLog) error {
	if s.Op == nil {
		return nil
	}

	opCopy := proto.Clone(s.Op).(*v1.Operation)
	opCopy.Status = v1.OperationStatus_STATUS_SYSTEM_CANCELLED
	opCopy.DisplayMessage = "operation cancelled"
	opCopy.UnixTimeEndMs = curTimeMillis()
	if err := oplog.Update(opCopy); err != nil {
		return err
	}

	return nil
}

// Task is a task that can be scheduled to run at a specific time.
type Task interface {
	Name() string                                                             // human readable name for this task.
	Next(now time.Time, runner TaskRunner) ScheduledTask                      // returns the next scheduled task.
	Run(ctx context.Context, execInfo ScheduledTask, runner TaskRunner) error // run the task.
	PlanID() string                                                           // the ID of the plan this task is associated with.
	RepoID() string                                                           // the ID of the repo this task is associated with.
}

type BaseTask struct {
	TaskName   string
	TaskPlanID string
	TaskRepoID string
}

func (b BaseTask) Name() string {
	return b.TaskName
}

func (b BaseTask) PlanID() string {
	return b.TaskPlanID
}

func (b BaseTask) RepoID() string {
	return b.TaskRepoID
}

func WithResticLogger(ctx context.Context, runner TaskRunner) (context.Context, func()) {
	capturer := ioutil.NewOutputCapturer(32_000) // 32k of logs
	return restic.ContextWithLogger(ctx, capturer), func() {
		if bytes := capturer.Bytes(); len(bytes) > 0 {
			if e := runner.AppendRawLog(bytes); e != nil {
				runner.Logger().Error("failed to append restic logs", zap.Error(e))
			}
		}
	}
}

// WithOperation is a utility that creates an operation to track the function's execution.
// timestamps are automatically added and the status is automatically updated if an error occurs.
func WithOperation(oplog *oplog.OpLog, op *v1.Operation, do func() error) error {
	op.UnixTimeStartMs = curTimeMillis() // update the start time from the planned time to the actual time.
	if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_UNKNOWN {
		op.Status = v1.OperationStatus_STATUS_INPROGRESS
	}
	if op.Id != 0 {
		if err := oplog.Update(op); err != nil {
			return fmt.Errorf("failed to add operation to oplog: %w", err)
		}
	} else {
		if err := oplog.Add(op); err != nil {
			return fmt.Errorf("failed to add operation to oplog: %w", err)
		}
	}
	err := do()
	if err != nil {
		op.Status = v1.OperationStatus_STATUS_ERROR
		op.DisplayMessage = err.Error()
	}
	op.UnixTimeEndMs = curTimeMillis()
	if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}
	if e := oplog.Update(op); e != nil {
		return multierror.Append(err, fmt.Errorf("failed to update operation in oplog: %w", e))
	}
	return err
}

func timeToUnixMillis(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond()/1000000)
}

func curTimeMillis() int64 {
	return timeToUnixMillis(time.Now())
}
