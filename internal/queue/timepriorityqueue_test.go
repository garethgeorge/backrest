package queue

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

// TestTPQEnqueue tests that enqueued elements are retruned highest priority first.
func TestTPQPriority(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestTPQStress(t *testing.T) {
	tpq := NewTimePriorityQueue[int]()
	start := time.Now()

	totalEnqueued := 0
	totalEnqueuedSum := 0

	go func() {
		ctx, _ := context.WithDeadline(context.Background(), start.Add(1*time.Second))
		for ctx.Err() == nil {
			v := rand.Intn(100)
			tpq.Enqueue(time.Now().Add(time.Duration(rand.Intn(1000)-500)*time.Millisecond), rand.Intn(5), v)
			totalEnqueuedSum += v
			totalEnqueued++
		}
	}()

	ctx, _ := context.WithDeadline(context.Background(), start.Add(3*time.Second))
	totalDequeued := 0
	sum := 0
	for ctx.Err() == nil || totalDequeued < totalEnqueued {
		sum += tpq.Dequeue(ctx)
		totalDequeued++
	}

	if sum != totalEnqueuedSum {
		t.Errorf("expected sum to be %d, got %d", totalEnqueuedSum, sum)
	}
}
