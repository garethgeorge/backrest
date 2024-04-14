package queue

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// TimeQueue is a priority queue that dequeues elements at (or after) a specified time. It is safe for concurrent use.
type TimeQueue[T equals[T]] struct {
	heap genericHeap[timeQueueEntry[T]]

	dequeueMu sync.Mutex
	mu        sync.Mutex
	notify    atomic.Pointer[chan struct{}]
}

func NewTimeQueue[T equals[T]]() *TimeQueue[T] {
	return &TimeQueue[T]{
		heap: genericHeap[timeQueueEntry[T]]{},
	}
}

func (t *TimeQueue[T]) Enqueue(at time.Time, v T) {
	t.mu.Lock()
	heap.Push(&t.heap, timeQueueEntry[T]{at, v})
	t.mu.Unlock()
	if n := t.notify.Load(); n != nil {
		select {
		case *n <- struct{}{}:
		default:
		}
	}
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

func (t *TimeQueue[T]) Remove(v T) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for idx := 0; idx < t.heap.Len(); idx++ {
		if t.heap[idx].v.Eq(v) {
			heap.Remove(&t.heap, idx)
			return
		}
	}
}

func (t *TimeQueue[T]) GetAll() []T {
	t.mu.Lock()
	defer t.mu.Unlock()

	res := make([]T, 0, t.heap.Len())
	for _, entry := range t.heap {
		res = append(res, entry.v)
	}
	return res
}

func (t *TimeQueue[T]) Dequeue(ctx context.Context) T {
	t.dequeueMu.Lock()
	defer t.dequeueMu.Unlock()

	notify := make(chan struct{}, 1)
	t.notify.Store(&notify)
	defer t.notify.Store(nil)

	for {
		t.mu.Lock()
		var wait time.Duration
		if t.heap.Len() > 0 {
			val := t.heap.Peek()
			wait = time.Until(val.at)
			if wait <= 0 {
				defer t.mu.Unlock()
				return heap.Pop(&t.heap).(timeQueueEntry[T]).v
			}
		}
		if wait == 0 || wait > 3*time.Minute {
			wait = 3 * time.Minute
		}
		t.mu.Unlock()

		timer := time.NewTimer(wait)

		select {
		case <-timer.C:
			t.mu.Lock()
			if len(t.heap) == 0 {
				t.mu.Unlock()
				continue
			}
			val := t.heap.Peek()
			if val.at.After(time.Now()) {
				t.mu.Unlock()
				continue
			}
			heap.Pop(&t.heap)
			t.mu.Unlock()
			return val.v
		case <-notify: // new task was added, loop again to ensure we have the earliest task.
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

type equals[T any] interface {
	Eq(other T) bool
}
