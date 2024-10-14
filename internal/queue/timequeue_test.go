package queue

import (
	"context"
	"math/rand"
	"slices"
	"testing"
	"time"
)

func TestTimeQueue(t *testing.T) {
	t.Parallel()
	tqueue := NewTimeQueue[val]()

	for i := 0; i < 100; i++ {
		tqueue.Enqueue(time.Now().Add(time.Millisecond*time.Duration(i*10)), val{v: i})
	}

	for i := 0; i < 100; i++ {
		v := tqueue.Dequeue(context.Background())
		if v.v != i {
			t.Errorf("expected %d, got %d", i, v.v)
		}
	}
}

func TestFuzzTimeQueue(t *testing.T) {
	t.Parallel()

	// generate random values and enqueue them
	values := make([]val, 100)
	for i := 0; i < 100; i++ {
		values[i] = val{v: rand.Intn(1000) - 500}
	}

	tqueue := NewTimeQueue[val]()
	now := time.Now()
	for _, v := range values {
		tqueue.Enqueue(now.Add(time.Millisecond*time.Duration(v.v)), v)
	}

	slices.SortFunc(values, func(i, j val) int {
		if i.v > j.v {
			return 1
		}
		return -1
	})

	// dequeue the values and check if they are in the correct order
	for i := 0; i < 100; i++ {
		v := tqueue.Dequeue(context.Background())
		if v.v != values[i].v {
			t.Errorf("expected %d, got %d", values[i].v, v.v)
		}
	}
}

func TestTimeQueueEnqueueWhileWaiting(t *testing.T) {
	t.Parallel()

	tqueue := NewTimeQueue[val]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	go func() {
		time.Sleep(time.Millisecond * 50)
		tqueue.Enqueue(time.Now(), val{v: 1})
	}()

	v := tqueue.Dequeue(ctx)
	if v.v != 1 {
		t.Errorf("expected 1, got %d", v.v)
	}
}

func TestTimeQueueDequeueTimeout(t *testing.T) {
	t.Parallel()

	tqueue := NewTimeQueue[val]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	v := tqueue.Dequeue(ctx)
	if v.v != 0 {
		t.Errorf("expected 0, got %d", v.v)
	}
}
