package orchestrator

import (
	"context"
	"reflect"
	"testing"
	"time"
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
