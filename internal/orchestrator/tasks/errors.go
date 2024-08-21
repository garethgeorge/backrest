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

// TaskRetryError is returned when a task should be retried after a specified backoff duration.
type TaskRetryError struct {
	Err     error
	Backoff time.Duration
}

func (e TaskRetryError) Error() string {
	return fmt.Sprintf("retry after %v: %v", e.Backoff, e.Err.Error())
}

func (e TaskRetryError) Is(err error) bool {
	other, ok := err.(TaskRetryError)
	if !ok {
		return false
	}
	return e.Backoff == other.Backoff && e.Err == other.Err
}

func (e TaskRetryError) Unwrap() error {
	return e.Err
}
