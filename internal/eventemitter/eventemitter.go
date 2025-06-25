package eventemitter

import (
	"sync"
)

// EventEmitter emits events to subscribers, events can be dropped if subscribers are not ready to receive them.
type EventEmitter[T any] struct {
	subscribers     map[chan T]struct{}
	mu              sync.Mutex // protects concurrent access to subscribers map
	DefaultCapacity int        // default capacity for channels, can be set to avoid blocking
}

func (e *EventEmitter[T]) Subscribe() chan T {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.subscribers == nil {
		e.subscribers = make(map[chan T]struct{}, e.DefaultCapacity)
	}

	ch := make(chan T)
	e.subscribers[ch] = struct{}{} // use empty struct to avoid memory overhead
	return ch
}

func (e *EventEmitter[T]) Unsubscribe(ch chan T) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.subscribers == nil {
		return // no subscribers to remove
	}
	if _, exists := e.subscribers[ch]; exists {
		delete(e.subscribers, ch)
		close(ch)
	}
}

func (e *EventEmitter[T]) Emit(event T) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.subscribers == nil {
		return // no subscribers to emit to
	}

	for ch := range e.subscribers {
		select {
		case ch <- event:
		default:
			// If the channel is full, we skip sending to avoid blocking.
			// This is a fire-and-forget approach.
		}
	}
}

func (e *EventEmitter[T]) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.subscribers == nil {
		return // nothing to clear
	}

	for ch := range e.subscribers {
		close(ch) // close each channel to signal no more events will be sent
	}
	e.subscribers = make(map[chan T]struct{}) // reset subscribers map
}

type BlockingEventEmitter[T any] struct {
	EventEmitter[T]
}

func NewBlocking[T any]() *BlockingEventEmitter[T] {
	return &BlockingEventEmitter[T]{
		EventEmitter: *New[T](),
	}
}

func (e *BlockingEventEmitter[T]) Emit(event T) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.subscribers == nil {
		return // no subscribers to emit to
	}

	for ch := range e.subscribers {
		ch <- event // block until the event is received
	}
}
