package queue

// genericHeap is a generic heap implementation that can be used with any type that satisfies the constraints.Ordered interface.
type GenericHeap[T Comparable[T]] []T

func (h GenericHeap[T]) Len() int {
	return len(h)
}

func (h GenericHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push pushes an element onto the heap. Do not call directly, use heap.Push
func (h *GenericHeap[T]) Push(x interface{}) {
	*h = append(*h, x.(T))
}

// Pop pops an element from the heap. Do not call directly, use heap.Pop
func (h *GenericHeap[T]) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h GenericHeap[T]) Peek() T {
	if len(h) == 0 {
		var zero T
		return zero
	}
	return h[0]
}

func (h GenericHeap[T]) Less(i, j int) bool {
	return h[i].Less(h[j])
}

type Comparable[T any] interface {
	Less(other T) bool
}
