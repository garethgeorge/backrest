package tasks

import (
	"fmt"
	"time"
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
