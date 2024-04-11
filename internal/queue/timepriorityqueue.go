package queue

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

// TimePriorityQueue is a priority queue that dequeues elements at (or after) a specified time, and prioritizes elements based on a priority value. It is safe for concurrent use.
type TimePriorityQueue[T any] struct {
	mu     sync.Mutex
	tqueue TimeQueue[priorityEntry[T]]
	ready  genericHeap[priorityEntry[T]]
}

func NewTimePriorityQueue[T any]() *TimePriorityQueue[T] {
	return &TimePriorityQueue[T]{
		tqueue: TimeQueue[priorityEntry[T]]{},
		ready:  genericHeap[priorityEntry[T]]{},
	}
}

func (t *TimePriorityQueue[T]) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tqueue.Len() + t.ready.Len()
}

func (t *TimePriorityQueue[T]) Peek() T {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ready.Len() > 0 {
		return t.ready.Peek().v
	}
	return t.tqueue.Peek().v
}

func (t *TimePriorityQueue[T]) Reset() []T {
	t.mu.Lock()
	defer t.mu.Unlock()
	var res []T
	for t.ready.Len() > 0 {
		res = append(res, heap.Pop(&t.ready).(priorityEntry[T]).v)
	}
	for t.tqueue.Len() > 0 {
		res = append(res, heap.Pop(&t.tqueue.heap).(timeQueueEntry[priorityEntry[T]]).v.v)
	}
	return res
}

func (t *TimePriorityQueue[T]) Enqueue(at time.Time, priority int, v T) {
	t.mu.Lock()
	t.tqueue.Enqueue(at, priorityEntry[T]{at, priority, v})
	t.mu.Unlock()
}

func (t *TimePriorityQueue[T]) Dequeue(ctx context.Context) T {
	t.mu.Lock()
	for {
		for t.tqueue.heap.Len() > 0 {
			thead := t.tqueue.Peek() // peek at the head of the time queue
			if thead.at.Before(time.Now()) {
				tqe := heap.Pop(&t.tqueue.heap).(timeQueueEntry[priorityEntry[T]])
				heap.Push(&t.ready, tqe.v)
			} else {
				break
			}
		}
		if t.ready.Len() > 0 {
			defer t.mu.Unlock()
			return heap.Pop(&t.ready).(priorityEntry[T]).v
		}
		t.mu.Unlock()
		// wait for the next element to be ready
		val := t.tqueue.Dequeue(ctx)
		t.mu.Lock()
		heap.Push(&t.ready, val)
	}
}

type priorityEntry[T any] struct {
	at       time.Time
	priority int
	v        T
}

func (t priorityEntry[T]) Less(other priorityEntry[T]) bool {
	return t.priority > other.priority
}
