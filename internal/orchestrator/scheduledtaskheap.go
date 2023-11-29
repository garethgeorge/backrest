package orchestrator

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type taskQueue struct {
	dequeueMu sync.Mutex
	mu        sync.Mutex
	heap      scheduledTaskHeap
	notify    chan struct{}

	Now func() time.Time
}

func (t *taskQueue) curTime() time.Time {
	if t.Now == nil {
		return time.Now()
	}
	return t.Now()
}

func (t *taskQueue) Push(task scheduledTask) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if task.task == nil {
		panic("task cannot be nil")
	}

	heap.Push(&t.heap, &task)
	if t.notify != nil {
		t.notify <- struct{}{}
	}
}

func (t *taskQueue) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.heap.tasks = nil
	if t.notify != nil {
		t.notify <- struct{}{}
	}
}

func (t *taskQueue) Dequeue(ctx context.Context) *scheduledTask {
	t.dequeueMu.Lock()
	defer t.dequeueMu.Unlock()

	t.notify = make(chan struct{}, 1)
	defer func() {
		t.notify = nil
	}()

	t.mu.Lock()
	for {
		first, ok := t.heap.Peek().(*scheduledTask)
		if !ok { // no tasks in heap.
			t.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil
			case <-t.notify:
			}
			t.mu.Lock()
			continue
		}
		t.mu.Unlock()
		timer := time.NewTimer(first.runAt.Sub(t.curTime()))

		t.mu.Lock()
		select {
		case <-timer.C:
			if t.heap.Len() == 0 {
				break
			}
			first = t.heap.Peek().(*scheduledTask)
			if first.runAt.After(t.curTime()) {
				// task is not yet ready to run
				break
			}

			heap.Pop(&t.heap) // remove the task from the heap
			t.mu.Unlock()
			return first
		case <-t.notify: // new task was added, loop again to ensure we have the earliest task.
			if !timer.Stop() {
				<-timer.C
			}
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			t.mu.Unlock()
			return nil
		}
	}
}

type scheduledTask struct {
	task  Task
	runAt time.Time
}

type scheduledTaskHeap struct {
	tasks []*scheduledTask
}

var _ heap.Interface = &scheduledTaskHeap{}

func (h *scheduledTaskHeap) Len() int {
	return len(h.tasks)
}

func (h *scheduledTaskHeap) Less(i, j int) bool {
	return h.tasks[i].runAt.Before(h.tasks[j].runAt)
}

func (h *scheduledTaskHeap) Swap(i, j int) {
	h.tasks[i], h.tasks[j] = h.tasks[j], h.tasks[i]
}

func (h *scheduledTaskHeap) Push(x interface{}) {
	h.tasks = append(h.tasks, x.(*scheduledTask))
}

func (h *scheduledTaskHeap) Pop() interface{} {
	old := h.tasks
	n := len(old)
	x := old[n-1]
	h.tasks = old[0 : n-1]
	return x
}

func (h *scheduledTaskHeap) Peek() interface{} {
	if len(h.tasks) == 0 {
		return nil
	}
	return h.tasks[0]
}
