package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/hashicorp/go-multierror"
)

type Task interface {
	Name() string                               // huamn readable name for this task.
	Next(now time.Time) *time.Time              // when this task would like to be run.
	Run(ctx context.Context) error              // run the task.
	Cancel(withStatus v1.OperationStatus) error // informat the task that it's scheduled execution will be skipped (either STATUS_USER_CANCELLED or STATUS_SYSTEM_CANCELLED).
	OperationId() int64                         // the id of the operation associated with this task (if any).
}

type TaskWithOperation struct {
	orch    *Orchestrator
	op      *v1.Operation
	running atomic.Bool
}

func (t *TaskWithOperation) OperationId() int64 {
	if t.op == nil {
		return 0
	}
	return t.op.Id
}

func (t *TaskWithOperation) setOperation(op *v1.Operation) error {
	if t.op != nil {
		return errors.New("task already has an operation")
	}
	if err := t.orch.OpLog.Add(op); err != nil {
		return fmt.Errorf("task failed to add operation to oplog: %v", err)
	}
	t.op = op
	return nil
}

func (t *TaskWithOperation) runWithOpAndContext(ctx context.Context, do func(ctx context.Context, op *v1.Operation) error) error {
	if t.op == nil {
		return errors.New("task has no operation, a call to setOperation first is required")
	}
	if t.running.Load() {
		return errors.New("task is already running")
	}

	t.running.Store(true)
	defer t.running.Store(false)
	defer func() {
		t.op = nil
	}()

	return WithOperation(t.orch.OpLog, t.op, func() error {
		return do(ctx, t.op)
	})
}

// Cancel marks a task as cancelled. Note that, unintuitively, it is actually an error to call cancel on a running task.
func (t *TaskWithOperation) Cancel(withStatus v1.OperationStatus) error {
	if t.running.Load() {
		return errors.New("cannot cancel a running task") // should never happen.
	}
	if t.op == nil {
		return nil
	}
	t.op.Status = withStatus
	t.op.UnixTimeEndMs = curTimeMillis()
	if err := t.orch.OpLog.Update(t.op); err != nil {
		return fmt.Errorf("failed to update operation %v in oplog: %w", t.op.Id, err)
	}
	t.op = nil
	return nil
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
