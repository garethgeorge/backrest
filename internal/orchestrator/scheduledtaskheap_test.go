package orchestrator

import (
	"context"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

type heapTestTask struct {
	name string
}

var _ Task = &heapTestTask{}

func (t *heapTestTask) Name() string {
	return t.name
}

func (t *heapTestTask) Next(now time.Time) *time.Time {
	return nil
}

func (t *heapTestTask) Run(ctx context.Context) error {
	return nil
}

func (t *heapTestTask) Cancel(withStatus v1.OperationStatus) error {
	return nil
}

func (t *heapTestTask) OperationId() int64 {
	return 0
}

func TestTaskQueueOrdering(t *testing.T) {
	h := taskQueue{}

	h.Push(scheduledTask{runAt: time.Now().Add(1 * time.Millisecond), task: &heapTestTask{name: "1"}})
	h.Push(scheduledTask{runAt: time.Now().Add(2 * time.Millisecond), task: &heapTestTask{name: "2"}})
	h.Push(scheduledTask{runAt: time.Now().Add(2 * time.Millisecond), task: &heapTestTask{name: "3"}})

	wantSeq := []string{"1", "2", "3"}
	seq := []string{}
	for i := 0; i < 3; i++ {
		task := h.Dequeue(context.Background())
		if task == nil || task.task == nil {
			t.Fatal("expected task")
		}
		seq = append(seq, task.task.Name())
	}

	if !reflect.DeepEqual(seq, wantSeq) {
		t.Errorf("got %v, want %v", seq, wantSeq)
	}
}

func TestLiveTaskEnqueue(t *testing.T) {
	h := taskQueue{}

	go func() {
		time.Sleep(1 * time.Millisecond)
		h.Push(scheduledTask{runAt: time.Now().Add(1 * time.Millisecond), task: &heapTestTask{name: "1"}})
	}()

	t1 := h.Dequeue(context.Background())
	if t1.task.Name() != "1" {
		t.Errorf("got %s, want 1", t1.task.Name())
	}
}

func TestTaskQueueReset(t *testing.T) {
	h := taskQueue{}

	h.Push(scheduledTask{runAt: time.Now().Add(1 * time.Millisecond), task: &heapTestTask{name: "1"}})
	h.Push(scheduledTask{runAt: time.Now().Add(2 * time.Millisecond), task: &heapTestTask{name: "2"}})
	h.Push(scheduledTask{runAt: time.Now().Add(2 * time.Millisecond), task: &heapTestTask{name: "3"}})

	if h.Dequeue(context.Background()).task.Name() != "1" {
		t.Fatal("expected 1")
	}
	h.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(1 * time.Millisecond)
		cancel()
	}()

	if h.Dequeue(ctx) != nil {
		t.Fatal("expected nil task")
	}
}

func TestTasksOrderedByPriority(t *testing.T) {
	h := taskQueue{}

	now := time.Now()
	h.Push(scheduledTask{runAt: now, task: &heapTestTask{name: "4"}, priority: 1})
	h.Push(scheduledTask{runAt: now, task: &heapTestTask{name: "3"}, priority: 2})
	h.Push(scheduledTask{runAt: now.Add(10 * time.Millisecond), task: &heapTestTask{name: "5"}, priority: 3})
	h.Push(scheduledTask{runAt: now, task: &heapTestTask{name: "2"}, priority: 3})
	h.Push(scheduledTask{runAt: now.Add(-10 * time.Millisecond), task: &heapTestTask{name: "1"}, priority: 3})

	wantSeq := []string{"1", "2", "3", "4", "5"}

	seq := []string{}

	for i := 0; i < 5; i++ {
		task := h.Dequeue(context.Background())
		if task == nil || task.task == nil {
			t.Fatal("expected task")
		}
		seq = append(seq, task.task.Name())
	}

	if !reflect.DeepEqual(seq, wantSeq) {
		t.Errorf("got %v, want %v", seq, wantSeq)
	}
}

func TestFuzzTaskQueue(t *testing.T) {
	h := taskQueue{}

	count := 100

	// Setup a bunch of tasks with random priorities and run times.
	tasks := make([]scheduledTask, count)
	for i := 0; i < count; i++ {
		at := time.Now().Add(time.Duration(rand.Intn(200)-50) * time.Millisecond)
		tasks[i] = scheduledTask{runAt: at, priority: 0, task: &heapTestTask{name: strconv.Itoa(i)}}
		h.Push(tasks[i])
	}

	seq := []string{}
	for i := 0; i < count; i++ {
		task := h.Dequeue(context.Background())
		if task == nil || task.task == nil {
			t.Fatal("expected task")
		}
		seq = append(seq, task.task.Name())
	}

	var expectOrdering []string
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].runAt.Equal(tasks[j].runAt) {
			return tasks[i].priority < tasks[j].priority
		}
		return tasks[i].runAt.Before(tasks[j].runAt)
	})

	for _, task := range tasks {
		expectOrdering = append(expectOrdering, task.task.Name())
	}

	if !reflect.DeepEqual(seq, expectOrdering) {
		t.Errorf("got %v, want %v", seq, expectOrdering)
	}
}
