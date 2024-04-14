package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// TaskRunner is an interface for running tasks. It is used by tasks to create operations and write logs.
type TaskRunner interface {
	// CreateOperation creates the operation in storage and sets the operation ID in the task.
	CreateOperation(*v1.Operation) error
	// UpdateOperation updates the operation in storage. It must be called after CreateOperation.
	UpdateOperation(*v1.Operation) error
	// WriteLog writes the log to storage and returns the reference.
	WriteLog([]byte) (string, error)
}

// ScheduledTask is a task that is scheduled to run at a specific time.
type ScheduledTask struct {
	Task  Task          // the task to run
	RunAt time.Time     // the time at which the task should be run.
	Op    *v1.Operation // optional operation associated with this execution of the task.
}

// Task is a task that can be scheduled to run at a specific time.
type Task interface {
	Name() string                                                             // huamn readable name for this task.
	Next(now time.Time, runner TaskRunner) ScheduledTask                      // returns the next scheduled task.
	Run(ctx context.Context, execInfo ScheduledTask, runner TaskRunner) error // run the task.
}

func LogToOperation(ctx context.Context, op *v1.Operation, runner TaskRunner) (context.Context, func() error) {
	capture := ioutil.NewOutputCapturer(32_000) // 32k of logs
	ctxWithLogger := restic.ContextWithLogger(ctx, capture)
	return ctxWithLogger, func() error {
		if bytes := capture.Bytes(); len(bytes) > 0 {
			ref, e := runner.WriteLog(bytes)
			if e != nil {
				return fmt.Errorf("failed to write log to logstore: %w", e)
			}
			op.Logref = ref
			zap.S().Debugf("wrote operation log to %v", ref)
		}
		return nil
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
