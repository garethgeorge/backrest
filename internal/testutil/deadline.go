package testutil

import (
	"context"
	"testing"
	"time"
)

var defaultDeadlineMargin = 5 * time.Second

func WithDeadlineFromTest(t *testing.T, ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := t.Deadline(); ok {
		return context.WithDeadline(ctx, deadline.Add(-defaultDeadlineMargin))
	}
	return ctx, func() {}
}
