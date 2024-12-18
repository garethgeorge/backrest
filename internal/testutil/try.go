package testutil

import (
	"context"
	"testing"
	"time"
)

func tryHelper(t *testing.T, ctx context.Context, f func() error) error {
	ctx, cancel := WithDeadlineFromTest(t, ctx)
	defer cancel()

	var err error
	interval := 10 * time.Millisecond
	for {
		timer := time.NewTimer(interval)
		interval += 10 * time.Millisecond
		select {
		case <-ctx.Done():
			timer.Stop()
			return err
		case <-timer.C:
			timer.Stop()
			err = f()
			if err == nil {
				return nil
			}
		}
	}
}

// try is a helper that spins until the condition becomes true OR the context is done.
func Try(t *testing.T, ctx context.Context, f func() error) {
	t.Helper()
	if err := tryHelper(t, ctx, f); err != nil {
		t.Fatalf("timeout before OK: %v", err)
	}
}

func TryNonfatal(t *testing.T, ctx context.Context, f func() error) {
	t.Helper()
	if err := tryHelper(t, ctx, f); err != nil {
		t.Errorf("timeout before OK: %v", err)
	}
}

func Retry(t *testing.T, ctx context.Context, f func() error) error {
	t.Helper()
	return tryHelper(t, ctx, f)
}
