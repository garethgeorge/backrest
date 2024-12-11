package testutil

import (
	"context"
	"testing"
	"time"
)

func tryHelper(t *testing.T, ctx context.Context, f func() error) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var err error
	for {
		select {
		case <-ctx.Done():
			return err
		case <-ticker.C:
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
