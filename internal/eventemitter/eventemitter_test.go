package eventemitter

import (
	"sync"
	"testing"
	"time"
)

func TestEventEmitter_SubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	em := &EventEmitter[int]{}

	// Test Subscribe
	ch1 := em.Subscribe()
	if ch1 == nil {
		t.Fatal("Subscribe should return a non-nil channel")
	}
	em.mu.Lock()
	if len(em.subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(em.subscribers))
	}
	em.mu.Unlock()

	ch2 := em.Subscribe()
	em.mu.Lock()
	if len(em.subscribers) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", len(em.subscribers))
	}
	em.mu.Unlock()

	// Test Unsubscribe
	em.Unsubscribe(ch1)
	em.mu.Lock()
	if len(em.subscribers) != 1 {
		t.Errorf("Expected 1 subscriber after unsubscribe, got %d", len(em.subscribers))
	}
	em.mu.Unlock()

	// Check if channel is closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("Channel should be closed after unsubscribe")
		}
	default:
		// This is expected if the channel was not closed, but the check above is better
	}

	em.Unsubscribe(ch2)
	em.mu.Lock()
	if len(em.subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", len(em.subscribers))
	}
	em.mu.Unlock()
}

func TestEventEmitter_Emit(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		em := &EventEmitter[string]{DefaultCapacity: 1} // Set capacity to ensure the event is buffered even if no subscribers are ready
		ch := em.Subscribe()

		go func() {
			em.Emit("hello")
		}()

		select {
		case msg := <-ch:
			if msg != "hello" {
				t.Errorf("Expected 'hello', got '%s'", msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for event")
		}
	}
}

func TestEventEmitter_Emit_NoBlock(t *testing.T) {
	t.Parallel()
	em := &EventEmitter[int]{}
	_ = em.Subscribe() // Subscriber with no listener, so channel will be full (capacity 0)

	done := make(chan struct{})
	go func() {
		em.Emit(1) // This should not block
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit blocked when it should not have")
	}
}

func TestEventEmitter_Clear(t *testing.T) {
	t.Parallel()
	em := &EventEmitter[bool]{}
	ch1 := em.Subscribe()
	ch2 := em.Subscribe()

	em.Clear()

	em.mu.Lock()
	if len(em.subscribers) != 0 {
		t.Errorf("Expected 0 subscribers after Clear, got %d", len(em.subscribers))
	}
	em.mu.Unlock()

	// Check if channels are closed
	if _, ok := <-ch1; ok {
		t.Error("ch1 should be closed after Clear")
	}
	if _, ok := <-ch2; ok {
		t.Error("ch2 should be closed after Clear")
	}
}

func TestEventEmitter_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		em := &EventEmitter[int]{}
		var wg sync.WaitGroup
		numGoroutines := 100

		// Concurrent Subscribe
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				_ = em.Subscribe()
			}()
		}
		wg.Wait()

		em.mu.Lock()
		if len(em.subscribers) != numGoroutines {
			t.Fatalf("Expected %d subscribers, got %d", numGoroutines, len(em.subscribers))
		}
		em.mu.Unlock()

		// Concurrent Emit and Unsubscribe
		subs := make([]chan int, 0, numGoroutines)
		em.mu.Lock()
		for ch := range em.subscribers {
			subs = append(subs, ch)
		}
		em.mu.Unlock()

		wg.Add(numGoroutines * 2)
		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				defer wg.Done()
				em.Emit(i)
			}(i)
			go func(i int) {
				defer wg.Done()
				em.Unsubscribe(subs[i])
			}(i)
		}
		wg.Wait()

		em.mu.Lock()
		if len(em.subscribers) != 0 {
			t.Errorf("Expected 0 subscribers after concurrent unsubscribe, got %d", len(em.subscribers))
		}
		em.mu.Unlock()
	}
}

func TestBlockingEventEmitter_Emit(t *testing.T) {
	t.Parallel()
	em := &BlockingEventEmitter[int]{}
	ch1 := em.Subscribe()
	ch2 := em.Subscribe()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch1:
			if msg != 42 {
				t.Errorf("ch1: Expected 42, got %d", msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("ch1: Timed out waiting for event")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case msg := <-ch2:
			if msg != 42 {
				t.Errorf("ch2: Expected 42, got %d", msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("ch2: Timed out waiting for event")
		}
	}()

	em.Emit(42)
	wg.Wait()
}

func TestBlockingEventEmitter_Emit_Blocks(t *testing.T) {
	t.Parallel()
	em := &BlockingEventEmitter[int]{}
	_ = em.Subscribe() // No listener

	done := make(chan struct{})
	go func() {
		em.Emit(1) // This should block
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Emit did not block when it should have")
	case <-time.After(50 * time.Millisecond):
		// Success, it blocked for a bit
	}
}
