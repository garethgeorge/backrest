package hook

import (
	"fmt"
	"time"
)

// HookErrorCancel requests that the calling operation cancel itself. It must be handled explicitly caller. Subsequent hooks will be skipped.
type HookErrorRequestCancel struct {
	Err error
}

func (e HookErrorRequestCancel) Error() string {
	return fmt.Sprintf("cancel: %v", e.Err.Error())
}

func (e HookErrorRequestCancel) Unwrap() error {
	return e.Err
}

// HookErrorFatal stops evaluation of subsequent hooks and will propagate to the hook flow's caller
type HookErrorFatal struct {
	Err error
}

func (e HookErrorFatal) Error() string {
	return fmt.Sprintf("fatal: %v", e.Err.Error())
}

func (e HookErrorFatal) Unwrap() error {
	return e.Err
}

type RetryBackoffPolicy = func(attempt int) time.Duration

// HookErrorRetry requests that the calling operation retry after a specified backoff duration
type HookErrorRetry struct {
	Err     error
	Backoff RetryBackoffPolicy
}

func (e HookErrorRetry) Error() string {
	return fmt.Sprintf("retry: %v", e.Err.Error())
}
