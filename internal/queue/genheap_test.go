package queue

import (
	"container/heap"
	"testing"
)

type val struct {
	v int
}

func (v val) Less(other val) bool {
	return v.v < other.v
}

func (v val) Eq(other val) bool {
	return v.v == other.v
}

func TestGenericHeapInit(t *testing.T) {
	t.Parallel()
	genHeap := genericHeap[val]{{v: 3}, {v: 2}, {v: 1}}
	heap.Init(&genHeap)

	if genHeap.Len() != 3 {
		t.Errorf("expected length to be 3, got %d", genHeap.Len())
	}

	for _, i := range []int{1, 2, 3} {
		v := heap.Pop(&genHeap).(val)
		if v.v != i {
			t.Errorf("expected %d, got %d", i, v.v)
		}
	}
}

func TestGenericHeapPushPop(t *testing.T) {
	t.Parallel()
	genHeap := genericHeap[val]{} // empty heap
	heap.Push(&genHeap, val{v: 3})
	heap.Push(&genHeap, val{v: 2})
	heap.Push(&genHeap, val{v: 1})

	if genHeap.Len() != 3 {
		t.Errorf("expected length to be 3, got %d", genHeap.Len())
	}

	for _, i := range []int{1, 2, 3} {
		v := heap.Pop(&genHeap).(val)
		if v.v != i {
			t.Errorf("expected %d, got %d", i, v.v)
		}
	}
}
