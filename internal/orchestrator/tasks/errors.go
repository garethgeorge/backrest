package tasks

import "errors"

// ErrTaskCancelled signals that the task is beign cancelled gracefully.
// This error is handled by marking the task as user cancelled.
// By default a task returning an error will be marked as failed otherwise.
var ErrTaskCancelled = errors.New("cancel task")
