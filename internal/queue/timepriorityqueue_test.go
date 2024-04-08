package queue

import (
	"context"
	"testing"
	"time"
)

// TestTPQEnqueue tests that enqueued elements are retruned highest priority first.
func TestTPQPriority(t *testing.T) {
	tpq := NewTimePriorityQueue[int]()

	now := time.Now().Add(-time.Second)
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now, i, i)
	}

	if tpq.Len() != 100 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	for i := 99; i >= 0; i-- {
		v := tpq.Dequeue(context.Background())
		if v != i {
			t.Errorf("expected %d, got %d", i, v)
		}
	}
}

func TestTPQMixedReadinessStates(t *testing.T) {
	tpq := NewTimePriorityQueue[int]()

	now := time.Now()
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now.Add(-100*time.Millisecond), i, i)
	}
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now.Add(100*time.Millisecond), i, i)
	}

	if tpq.Len() != 200 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	for j := 0; j < 2; j++ {
		for i := 99; i >= 0; i-- {
			v := tpq.Dequeue(context.Background())
			if v != i {
				t.Errorf("pass %d expected %d, got %d", j, i, v)
			}
		}
	}
}
