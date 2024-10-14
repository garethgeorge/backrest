package queue

import (
	"context"
	"math/rand"
	"slices"
	"testing"
	"time"
)

// TestTPQEnqueue tests that enqueued elements are retruned highest priority first.
func TestTPQPriority(t *testing.T) {
	t.Parallel()
	tpq := NewTimePriorityQueue[val]()

	now := time.Now().Add(-time.Second)
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now, i, val{i})
	}

	if tpq.Len() != 100 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	for i := 99; i >= 0; i-- {
		v := tpq.Dequeue(context.Background())
		if v.v != i {
			t.Errorf("expected %d, got %d", i, v)
		}
	}
}

func TestTPQMixedReadinessStates(t *testing.T) {
	t.Parallel()
	tpq := NewTimePriorityQueue[val]()

	now := time.Now()
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now.Add(-100*time.Millisecond), i, val{i})
	}
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now.Add(100*time.Millisecond), i, val{i})
	}

	if tpq.Len() != 200 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	for j := 0; j < 2; j++ {
		for i := 99; i >= 0; i-- {
			v := tpq.Dequeue(context.Background())
			if v.v != i {
				t.Errorf("pass %d expected %d, got %d", j, i, v)
			}
		}
	}
}

func TestTPQStress(t *testing.T) {
	t.Parallel()
	tpq := NewTimePriorityQueue[val]()
	start := time.Now()

	totalEnqueued := 0
	totalEnqueuedSum := 0

	go func() {
		ctx, _ := context.WithDeadline(context.Background(), start.Add(1*time.Second))
		for ctx.Err() == nil {
			v := rand.Intn(100) + 1
			tpq.Enqueue(time.Now().Add(time.Duration(rand.Intn(1000)-500)*time.Millisecond), rand.Intn(5), val{v})
			totalEnqueuedSum += v
			totalEnqueued++
		}
	}()

	ctx, _ := context.WithDeadline(context.Background(), start.Add(3*time.Second))
	totalDequeued := 0
	sum := 0
	for ctx.Err() == nil || totalDequeued < totalEnqueued {
		v := tpq.Dequeue(ctx)
		if v.v != 0 {
			totalDequeued++
			sum += v.v
		}
	}

	if totalDequeued != totalEnqueued {
		t.Errorf("expected totalDequeued to be %d, got %d", totalEnqueued, totalDequeued)
	}

	if sum != totalEnqueuedSum {
		t.Errorf("expected sum to be %d, got %d", totalEnqueuedSum, sum)
	}
}

func TestTPQRemove(t *testing.T) {
	t.Parallel()
	tpq := NewTimePriorityQueue[val]()

	now := time.Now().Add(-time.Second) // make sure the time is in the past
	for i := 0; i < 100; i++ {
		tpq.Enqueue(now, -i, val{i})
	}

	if tpq.Len() != 100 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	// remove all even numbers, dequeue the odd numbers
	for i := 0; i < 100; i += 2 {
		tpq.Remove(val{i})
		v := tpq.Dequeue(context.Background())
		if v.v != i+1 {
			t.Errorf("expected %d, got %d", i+1, v)
		}
	}

	if tpq.Len() != 0 {
		t.Errorf("expected length to be 0, got %d", tpq.Len())
	}
}

func TestTPQReset(t *testing.T) {
	t.Parallel()
	tpq := NewTimePriorityQueue[val]()

	now := time.Now() // make sure the time is in the past
	for i := 0; i < 50; i++ {
		tpq.Enqueue(now.Add(time.Second), i, val{i})
	}
	for i := 50; i < 100; i++ {
		tpq.Enqueue(now.Add(-time.Second), i, val{i})
	}

	if tpq.Len() != 100 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	dv := tpq.Dequeue(context.Background())
	if dv.v != 99 {
		t.Errorf("expected 99, got %d", dv.v)
	}

	vals := tpq.Reset()

	if len(vals) != 99 {
		t.Errorf("expected length to be 100, got %d", len(vals))
	}

	slices.SortFunc(vals, func(i, j val) int {
		if i.v > j.v {
			return 1
		}
		return -1
	})

	for i := 0; i < 99; i++ {
		if vals[i].v != i {
			t.Errorf("expected %d, got %d", i, vals[i].v)
		}
	}

	if tpq.Len() != 0 {
		t.Errorf("expected length to be 0, got %d", tpq.Len())
	}
}

func TestTPQGetAll(t *testing.T) {
	t.Parallel()
	tpq := NewTimePriorityQueue[val]()
	now := time.Now()

	for i := 0; i < 100; i++ {
		tpq.Enqueue(now.Add(time.Second), i, val{i})
	}

	if tpq.Len() != 100 {
		t.Errorf("expected length to be 100, got %d", tpq.Len())
	}

	vals := tpq.GetAll()

	if len(vals) != 100 {
		t.Errorf("expected length to be 100, got %d", len(vals))
	}

	slices.SortFunc(vals, func(i, j val) int {
		if i.v > j.v {
			return 1
		}
		return -1
	})

	for i := 0; i < 100; i++ {
		if vals[i].v != i {
			t.Errorf("expected %d, got %d", i, vals[i].v)
		}
	}
}
