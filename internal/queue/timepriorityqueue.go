package queue

import (
	"container/heap"
	"context"
	"time"
)

// TimePriorityQueue is a priority queue that dequeues elements at (or after) a specified time, and prioritizes elements based on a priority value. It is safe for concurrent use.
type TimePriorityQueue[T equals[T]] struct {
	tqueue TimeQueue[priorityEntry[T]]
	ready  GenericHeap[priorityEntry[T]]
}

func NewTimePriorityQueue[T equals[T]]() *TimePriorityQueue[T] {
	return &TimePriorityQueue[T]{
		tqueue: TimeQueue[priorityEntry[T]]{},
		ready:  GenericHeap[priorityEntry[T]]{},
	}
}

func (t *TimePriorityQueue[T]) Len() int {
	t.tqueue.mu.Lock()
	defer t.tqueue.mu.Unlock()
	return t.tqueue.heap.Len() + t.ready.Len()
}

func (t *TimePriorityQueue[T]) Peek() T {
	t.tqueue.mu.Lock()
	defer t.tqueue.mu.Unlock()

	if t.ready.Len() > 0 {
		return t.ready.Peek().v
	}
	if t.tqueue.heap.Len() > 0 {
		return t.tqueue.heap.Peek().v.v
	}
	var zero T
	return zero
}

func (t *TimePriorityQueue[T]) Reset() []T {
	t.tqueue.mu.Lock()
	defer t.tqueue.mu.Unlock()
	var res []T
	for t.ready.Len() > 0 {
		res = append(res, heap.Pop(&t.ready).(priorityEntry[T]).v)
	}
	for t.tqueue.heap.Len() > 0 {
		res = append(res, heap.Pop(&t.tqueue.heap).(timeQueueEntry[priorityEntry[T]]).v.v)
	}
	return res
}

func (t *TimePriorityQueue[T]) GetAll() []T {
	t.tqueue.mu.Lock()
	defer t.tqueue.mu.Unlock()
	res := make([]T, 0, t.tqueue.heap.Len()+t.ready.Len())
	for _, entry := range t.tqueue.heap {
		res = append(res, entry.v.v)
	}
	for _, entry := range t.ready {
		res = append(res, entry.v)
	}
	return res
}

func (t *TimePriorityQueue[T]) Remove(v T) {
	t.tqueue.mu.Lock()
	defer t.tqueue.mu.Unlock()

	for idx := 0; idx < t.tqueue.heap.Len(); idx++ {
		if t.tqueue.heap[idx].v.v.Eq(v) {
			heap.Remove(&t.tqueue.heap, idx)
			return
		}
	}

	for idx := 0; idx < t.ready.Len(); idx++ {
		if t.ready[idx].v.Eq(v) {
			heap.Remove(&t.ready, idx)
			return
		}
	}
}

func (t *TimePriorityQueue[T]) Enqueue(at time.Time, priority int, v T) {
	t.tqueue.Enqueue(at, priorityEntry[T]{at, priority, v})
}

func (t *TimePriorityQueue[T]) Dequeue(ctx context.Context) T {
	t.tqueue.mu.Lock()
	for {
		for t.tqueue.heap.Len() > 0 {
			thead := t.tqueue.heap.Peek() // peek at the head of the time queue
			if thead.at.Before(time.Now()) {
				tqe := heap.Pop(&t.tqueue.heap).(timeQueueEntry[priorityEntry[T]])
				heap.Push(&t.ready, tqe.v)
			} else {
				break
			}
		}
		if t.ready.Len() > 0 {
			defer t.tqueue.mu.Unlock()
			return heap.Pop(&t.ready).(priorityEntry[T]).v
		}
		t.tqueue.mu.Unlock()
		// wait for the next element to be ready
		val := t.tqueue.Dequeue(ctx)
		t.tqueue.mu.Lock()
		heap.Push(&t.ready, val)
	}
}

type priorityEntry[T equals[T]] struct {
	at       time.Time
	priority int
	v        T
}

func (t priorityEntry[T]) Less(other priorityEntry[T]) bool {
	return t.priority > other.priority
}

func (t priorityEntry[T]) Eq(other priorityEntry[T]) bool {
	return t.at == other.at && t.priority == other.priority && t.v.Eq(other.v)
}
