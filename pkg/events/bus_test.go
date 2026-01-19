package events

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()

	if bus == nil {
		t.Fatal("NewBus returned nil")
	}
	if bus.subscribers == nil {
		t.Error("subscribers map should be initialized")
	}
	if bus.allHandlers == nil {
		t.Error("allHandlers slice should be initialized")
	}
	if bus.closed {
		t.Error("bus should not be closed initially")
	}
}

func TestPublishSubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var received atomic.Bool
	var receivedEvent core.Event

	var mu sync.Mutex

	unsub := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		received.Store(true)
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
	})
	defer unsub()

	event := core.NewEvent(core.EventMessageCreated, "test data")
	bus.Publish(event)

	// Wait for async handler
	time.Sleep(50 * time.Millisecond)

	if !received.Load() {
		t.Error("handler should have received the event")
	}

	mu.Lock()
	if receivedEvent.ID != event.ID {
		t.Errorf("event ID mismatch: expected %q, got %q", event.ID, receivedEvent.ID)
	}
	mu.Unlock()
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count atomic.Int32

	unsub1 := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		count.Add(1)
	})
	defer unsub1()

	unsub2 := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		count.Add(1)
	})
	defer unsub2()

	unsub3 := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		count.Add(1)
	})
	defer unsub3()

	event := core.NewEvent(core.EventMessageCreated, nil)
	bus.Publish(event)

	// Wait for async handlers
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 3 {
		t.Errorf("expected 3 handlers called, got %d", count.Load())
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count atomic.Int32

	unsub := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		count.Add(1)
	})

	// First publish - should receive
	event1 := core.NewEvent(core.EventMessageCreated, nil)
	bus.Publish(event1)
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("expected 1 call, got %d", count.Load())
	}

	// Unsubscribe
	unsub()

	// Second publish - should NOT receive
	event2 := core.NewEvent(core.EventMessageCreated, nil)
	bus.Publish(event2)
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("expected still 1 call after unsubscribe, got %d", count.Load())
	}
}

func TestSubscribeAll(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var receivedTypes []core.EventType
	var mu sync.Mutex

	unsub := bus.SubscribeAll(func(event core.Event) {
		mu.Lock()
		receivedTypes = append(receivedTypes, event.Type)
		mu.Unlock()
	})
	defer unsub()

	// Publish different event types
	bus.Publish(core.NewEvent(core.EventMessageCreated, nil))
	bus.Publish(core.NewEvent(core.EventAgentTyping, nil))
	bus.Publish(core.NewEvent(core.EventConversationStarted, nil))

	// Wait for async handlers
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(receivedTypes) != 3 {
		t.Errorf("expected 3 events, got %d", len(receivedTypes))
	}
	mu.Unlock()
}

func TestSubscribeAllUnsubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count atomic.Int32

	unsub := bus.SubscribeAll(func(event core.Event) {
		count.Add(1)
	})

	bus.Publish(core.NewEvent(core.EventMessageCreated, nil))
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("expected 1 call, got %d", count.Load())
	}

	unsub()

	bus.Publish(core.NewEvent(core.EventAgentTyping, nil))
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("expected still 1 call, got %d", count.Load())
	}
}

func TestPanicRecovery(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var handler1Called atomic.Bool
	var handler2Called atomic.Bool

	// Handler that panics
	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		handler1Called.Store(true)
		panic("test panic")
	})

	// Handler that should still be called
	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		handler2Called.Store(true)
	})

	event := core.NewEvent(core.EventMessageCreated, nil)
	bus.Publish(event)

	// Wait for async handlers
	time.Sleep(100 * time.Millisecond)

	if !handler1Called.Load() {
		t.Error("panicking handler should have been called")
	}
	if !handler2Called.Load() {
		t.Error("second handler should still be called despite panic in first")
	}
}

func TestConcurrentPublish(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var count atomic.Int32

	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		count.Add(1)
	})

	// Publish from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := core.NewEvent(core.EventMessageCreated, nil)
			bus.Publish(event)
		}()
	}

	wg.Wait()
	// Wait for handlers to complete
	time.Sleep(200 * time.Millisecond)

	if count.Load() != 100 {
		t.Errorf("expected 100 events received, got %d", count.Load())
	}
}

func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var wg sync.WaitGroup

	// Concurrent subscriptions
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unsub := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {})
			time.Sleep(10 * time.Millisecond)
			unsub()
		}()
	}

	// Concurrent publishes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(core.NewEvent(core.EventMessageCreated, nil))
		}()
	}

	wg.Wait()
	// Test completes without race conditions - verified with -race flag
}

func TestClose(t *testing.T) {
	bus := NewBus()

	var handlerDone atomic.Bool

	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		time.Sleep(100 * time.Millisecond)
		handlerDone.Store(true)
	})

	bus.Publish(core.NewEvent(core.EventMessageCreated, nil))

	// Close should wait for handler
	start := time.Now()
	bus.Close()
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("Close should wait for handlers, elapsed: %v", elapsed)
	}
	if !handlerDone.Load() {
		t.Error("handler should have completed")
	}
}

func TestClosePreventsFurtherPublish(t *testing.T) {
	bus := NewBus()

	var count atomic.Int32

	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		count.Add(1)
	})

	bus.Close()

	// Publish after close should be ignored
	bus.Publish(core.NewEvent(core.EventMessageCreated, nil))
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 0 {
		t.Errorf("expected 0 events after close, got %d", count.Load())
	}
}

func TestSubscribeAfterClose(t *testing.T) {
	bus := NewBus()
	bus.Close()

	// Subscribe after close should return no-op unsub
	unsub := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {})
	unsub() // Should not panic
}

func TestSubscribeAllAfterClose(t *testing.T) {
	bus := NewBus()
	bus.Close()

	unsub := bus.SubscribeAll(func(event core.Event) {})
	unsub() // Should not panic
}

func TestIsClosed(t *testing.T) {
	bus := NewBus()

	if bus.IsClosed() {
		t.Error("bus should not be closed initially")
	}

	bus.Close()

	if !bus.IsClosed() {
		t.Error("bus should be closed after Close()")
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}

	unsub1 := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {})
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	unsub2 := bus.Subscribe(core.EventAgentTyping, func(event core.Event) {})
	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}

	unsub3 := bus.SubscribeAll(func(event core.Event) {})
	if bus.SubscriberCount() != 3 {
		t.Errorf("expected 3 subscribers, got %d", bus.SubscriberCount())
	}

	unsub1()
	unsub2()
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber after unsubs, got %d", bus.SubscriberCount())
	}

	unsub3()
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestSubscriberCountForType(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	if bus.SubscriberCountForType(core.EventMessageCreated) != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCountForType(core.EventMessageCreated))
	}

	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {})
	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {})
	bus.Subscribe(core.EventAgentTyping, func(event core.Event) {})

	if bus.SubscriberCountForType(core.EventMessageCreated) != 2 {
		t.Errorf("expected 2 subscribers for MessageCreated, got %d", bus.SubscriberCountForType(core.EventMessageCreated))
	}
	if bus.SubscriberCountForType(core.EventAgentTyping) != 1 {
		t.Errorf("expected 1 subscriber for AgentTyping, got %d", bus.SubscriberCountForType(core.EventAgentTyping))
	}
}

func TestDifferentEventTypes(t *testing.T) {
	bus := NewBus()
	defer bus.Close()

	var messageCreatedCount atomic.Int32
	var agentTypingCount atomic.Int32

	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		messageCreatedCount.Add(1)
	})

	bus.Subscribe(core.EventAgentTyping, func(event core.Event) {
		agentTypingCount.Add(1)
	})

	bus.Publish(core.NewEvent(core.EventMessageCreated, nil))
	bus.Publish(core.NewEvent(core.EventMessageCreated, nil))
	bus.Publish(core.NewEvent(core.EventAgentTyping, nil))

	time.Sleep(100 * time.Millisecond)

	if messageCreatedCount.Load() != 2 {
		t.Errorf("expected 2 MessageCreated events, got %d", messageCreatedCount.Load())
	}
	if agentTypingCount.Load() != 1 {
		t.Errorf("expected 1 AgentTyping event, got %d", agentTypingCount.Load())
	}
}

func TestImplementsEventBusInterface(t *testing.T) {
	var _ EventBus = (*Bus)(nil)
}
