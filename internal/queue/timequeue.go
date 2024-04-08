package queue

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

// TimeQueue is a priority queue that dequeues elements at (or after) a specified time. It is safe for concurrent use.
type TimeQueue[T any] struct {
	heap genericHeap[timeQueueEntry[T]]

	dequeueMu sync.Mutex
	mu        sync.Mutex
	notify    chan struct{}
}

func NewTimeQueue[T any]() *TimeQueue[T] {
	return &TimeQueue[T]{
		heap: genericHeap[timeQueueEntry[T]]{},
	}
}

func (t *TimeQueue[T]) Enqueue(at time.Time, v T) {
	t.mu.Lock()
	heap.Push(&t.heap, timeQueueEntry[T]{at, v})
	if t.notify != nil {
		t.notify <- struct{}{}
	}
	t.mu.Unlock()
}

func (t *TimeQueue[T]) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.heap.Len()
}

func (t *TimeQueue[T]) Peek() T {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.heap.Len() == 0 {
		var zero T
		return zero
	}
	return t.heap.Peek().v
}

func (t *TimeQueue[T]) Reset() []T {
	t.mu.Lock()
	defer t.mu.Unlock()

	var res []T
	for t.heap.Len() > 0 {
		res = append(res, heap.Pop(&t.heap).(timeQueueEntry[T]).v)
	}
	return res
}

func (t *TimeQueue[T]) Dequeue(ctx context.Context) T {
	t.dequeueMu.Lock()
	defer t.dequeueMu.Unlock()

	t.mu.Lock()
	t.notify = make(chan struct{}, 1)
	defer func() {
		t.mu.Lock()
		close(t.notify)
		t.notify = nil
		t.mu.Unlock()
	}()
	t.mu.Unlock()

	for {
		t.mu.Lock()

		var wait time.Duration
		if t.heap.Len() == 0 {
			wait = 3 * time.Minute
		} else {
			val := t.heap.Peek()
			wait = time.Until(val.at)
			if wait <= 0 {
				t.mu.Unlock()
				return heap.Pop(&t.heap).(timeQueueEntry[T]).v
			}
		}
		t.mu.Unlock()

		timer := time.NewTimer(wait)

		select {
		case <-timer.C:
			t.mu.Lock()
			val, ok := heap.Pop(&t.heap).(timeQueueEntry[T])
			if !ok || val.at.After(time.Now()) {
				t.mu.Unlock()
				continue
			}
			t.mu.Unlock()
			return val.v
		case <-t.notify: // new task was added, loop again to ensure we have the earliest task.
			if !timer.Stop() {
				<-timer.C
			}
			continue
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			var zero T
			return zero
		}
	}
}

type timeQueueEntry[T any] struct {
	at time.Time
	v  T
}

func (t timeQueueEntry[T]) Less(other timeQueueEntry[T]) bool {
	return t.at.Before(other.at)
}

func (t timeQueueEntry[T]) Eq(other timeQueueEntry[T]) bool {
	return t.at.Equal(other.at)
}
