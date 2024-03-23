package orchestrator

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

var taskQueueDefaultPollInterval = 3 * time.Minute

type taskQueue struct {
	dequeueMu    sync.Mutex
	mu           sync.Mutex
	heap         scheduledTaskHeapByTime
	notify       chan struct{}
	ready        scheduledTaskHeapByPriorityThenTime
	pollInterval time.Duration

	Now func() time.Time
}

func newTaskQueue(now func() time.Time) taskQueue {
	return taskQueue{
		heap:         scheduledTaskHeapByTime{},
		ready:        scheduledTaskHeapByPriorityThenTime{},
		pollInterval: taskQueueDefaultPollInterval,
		Now:          now,
	}
}

func (t *taskQueue) curTime() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

func (t *taskQueue) Push(tasks ...scheduledTask) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, task := range tasks {
		task := task
		if task.task == nil {
			panic("task cannot be nil")
		}
		heap.Push(&t.heap, &task)
	}

	if t.notify != nil {
		t.notify <- struct{}{}
	}
}

func (t *taskQueue) Reset() []*scheduledTask {
	t.mu.Lock()
	defer t.mu.Unlock()

	oldTasks := t.heap.tasks
	oldTasks = append(oldTasks, t.ready.tasks...)
	t.heap.tasks = nil
	t.ready.tasks = nil

	if t.notify != nil {
		t.notify <- struct{}{}
	}
	return oldTasks
}

func (t *taskQueue) Dequeue(ctx context.Context) *scheduledTask {
	t.dequeueMu.Lock()
	defer t.dequeueMu.Unlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	t.notify = make(chan struct{}, 10)
	defer func() {
		close(t.notify)
		t.notify = nil
	}()

	for {
		first, ok := t.heap.Peek().(*scheduledTask)
		if !ok { // no tasks in heap.
			if t.ready.Len() > 0 {
				return heap.Pop(&t.ready).(*scheduledTask)
			}
			t.mu.Unlock()
			select {
			case <-ctx.Done():
				t.mu.Lock()
				return nil
			case <-t.notify:
			}
			t.mu.Lock()
			continue
		}

		now := t.curTime()

		// if there's a task in the ready queue AND the first task isn't ready yet then immediately return the ready task.
		ready, ok := t.ready.Peek().(*scheduledTask)
		if ok && now.Before(first.runAt) {
			heap.Pop(&t.ready)
			return ready
		}

		t.mu.Unlock()
		d := first.runAt.Sub(now)
		if t.pollInterval > 0 && d > t.pollInterval {
			// A poll interval may be set to work around clock changes
			// e.g. when a laptop wakes from sleep or the system clock is adjusted.
			d = t.pollInterval
		}
		timer := time.NewTimer(d)

		select {
		case <-timer.C:
			t.mu.Lock()
			if t.heap.Len() == 0 {
				break
			}

			for {
				first, ok := t.heap.Peek().(*scheduledTask)
				if !ok {
					break
				}
				if first.runAt.After(t.curTime()) {
					// task is not yet ready to run
					break
				}
				heap.Pop(&t.heap) // remove the task from the heap
				heap.Push(&t.ready, first)
			}

			if t.ready.Len() == 0 {
				break
			}
			return heap.Pop(&t.ready).(*scheduledTask)
		case <-t.notify: // new task was added, loop again to ensure we have the earliest task.
			t.mu.Lock()
			if !timer.Stop() {
				<-timer.C
			}
		case <-ctx.Done():
			t.mu.Lock()
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		}
	}
}

type scheduledTask struct {
	task      Task
	runAt     time.Time
	priority  int
	callbacks []func(error)
}

type scheduledTaskHeap struct {
	tasks      []*scheduledTask
	comparator func(i, j *scheduledTask) bool
}

func (h *scheduledTaskHeap) Len() int {
	return len(h.tasks)
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

type scheduledTaskHeapByTime struct {
	scheduledTaskHeap
}

var _ heap.Interface = &scheduledTaskHeapByTime{}

func (h *scheduledTaskHeapByTime) Less(i, j int) bool {
	return h.tasks[i].runAt.Before(h.tasks[j].runAt)
}

type scheduledTaskHeapByPriorityThenTime struct {
	scheduledTaskHeap
}

var _ heap.Interface = &scheduledTaskHeapByPriorityThenTime{}

func (h *scheduledTaskHeapByPriorityThenTime) Less(i, j int) bool {
	if h.tasks[i].priority != h.tasks[j].priority {
		return h.tasks[i].priority > h.tasks[j].priority
	}
	return h.tasks[i].runAt.Before(h.tasks[j].runAt)
}
