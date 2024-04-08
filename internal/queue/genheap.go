package queue

// genericHeap is a generic heap implementation that can be used with any type that satisfies the constraints.Ordered interface.
type genericHeap[T comparable[T]] []T

func (h genericHeap[T]) Len() int {
	return len(h)
}

func (h genericHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push pushes an element onto the heap. Do not call directly, use heap.Push
func (h *genericHeap[T]) Push(x interface{}) {
	*h = append(*h, x.(T))
}

// Pop pops an element from the heap. Do not call directly, use heap.Pop
func (h *genericHeap[T]) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h genericHeap[T]) Peek() T {
	if len(h) == 0 {
		var zero T
		return zero
	}
	return h[0]
}

func (h genericHeap[T]) Less(i, j int) bool {
	return h[i].Less(h[j])
}

type comparable[T any] interface {
	Less(other T) bool
}
