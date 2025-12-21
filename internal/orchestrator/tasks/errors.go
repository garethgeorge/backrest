package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

// TaskCancelledError is returned when a task is cancelled.
type TaskCancelledError struct {
}

func (e TaskCancelledError) Error() string {
	return "task cancelled"
}

func (e TaskCancelledError) Is(err error) bool {
	_, ok := err.(TaskCancelledError)
	return ok
}

type RetryBackoffPolicy = func(attempt int) time.Duration

// TaskRetryError is returned when a task should be retried after a specified backoff duration.
type TaskRetryError struct {
	Err     error
	Backoff RetryBackoffPolicy
}

func (e TaskRetryError) Error() string {
	return fmt.Sprintf("retry: %v", e.Err.Error())
}

func (e TaskRetryError) Unwrap() error {
	return e.Err
}

// NotifyError triggers the ANY_ERROR hook and any specific error conditions provided.
// It returns the original error for convenience in chaining.
func NotifyError(ctx context.Context, runner TaskRunner, taskName string, err error, conditions ...v1.Hook_Condition) error {
	if err == nil {
		return nil
	}
	allConditions := append([]v1.Hook_Condition{v1.Hook_CONDITION_ANY_ERROR}, conditions...)
	// Log the error via hooks. We ignore the error from ExecuteHooks itself to avoid masking the original error.
	_ = runner.ExecuteHooks(ctx, allConditions, HookVars{
		Task:  taskName,
		Error: err.Error(),
	})
	return err
}
