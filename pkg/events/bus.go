// Package events provides the event bus system for publishing and subscribing
// to events in the v2 AgentPipe architecture.
package events

import (
	"sync"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/log"
)

// Handler is a function that handles an event.
type Handler func(event core.Event)

// EventBus defines the interface for publishing and subscribing to events.
type EventBus interface {
	// Publish sends an event to all subscribers of that event type.
	Publish(event core.Event)
	// Subscribe registers a handler for a specific event type.
	// Returns an unsubscribe function.
	Subscribe(eventType core.EventType, handler Handler) func()
	// SubscribeAll registers a handler for all event types.
	// Returns an unsubscribe function.
	SubscribeAll(handler Handler) func()
	// Close shuts down the event bus and waits for pending handlers to complete.
	Close()
}

// subscription holds a handler and its ID for unsubscription.
type subscription struct {
	id      int
	handler Handler
}

// Bus is a thread-safe implementation of EventBus.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[core.EventType][]subscription
	allHandlers []subscription
	nextID      int
	closed      bool
	wg          sync.WaitGroup
}

// NewBus creates a new event bus instance.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[core.EventType][]subscription),
		allHandlers: make([]subscription, 0),
	}
}

// Publish sends an event to all subscribers of that event type and all-event handlers.
// Events are dispatched asynchronously using goroutines.
func (b *Bus) Publish(event core.Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}

	// Get type-specific handlers
	typeHandlers := make([]Handler, 0)
	if subs, ok := b.subscribers[event.Type]; ok {
		for _, sub := range subs {
			typeHandlers = append(typeHandlers, sub.handler)
		}
	}

	// Get all-event handlers
	allHandlers := make([]Handler, len(b.allHandlers))
	for i, sub := range b.allHandlers {
		allHandlers[i] = sub.handler
	}
	b.mu.RUnlock()

	// Dispatch to type-specific handlers
	for _, handler := range typeHandlers {
		b.dispatchAsync(event, handler)
	}

	// Dispatch to all-event handlers
	for _, handler := range allHandlers {
		b.dispatchAsync(event, handler)
	}
}

// dispatchAsync runs a handler in a goroutine with panic recovery.
func (b *Bus) dispatchAsync(event core.Event, handler Handler) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(map[string]interface{}{
					"event_type": event.Type,
					"event_id":   event.ID,
					"panic":      r,
				}).Error("event handler panicked")
			}
		}()
		handler(event)
	}()
}

// Subscribe registers a handler for a specific event type.
// Returns an unsubscribe function that removes the handler.
func (b *Bus) Subscribe(eventType core.EventType, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return func() {}
	}

	id := b.nextID
	b.nextID++

	sub := subscription{
		id:      id,
		handler: handler,
	}

	b.subscribers[eventType] = append(b.subscribers[eventType], sub)

	// Return unsubscribe function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subs := b.subscribers[eventType]
		for i, s := range subs {
			if s.id == id {
				b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}

// SubscribeAll registers a handler for all event types.
// Returns an unsubscribe function that removes the handler.
func (b *Bus) SubscribeAll(handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return func() {}
	}

	id := b.nextID
	b.nextID++

	sub := subscription{
		id:      id,
		handler: handler,
	}

	b.allHandlers = append(b.allHandlers, sub)

	// Return unsubscribe function
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		for i, s := range b.allHandlers {
			if s.id == id {
				b.allHandlers = append(b.allHandlers[:i], b.allHandlers[i+1:]...)
				break
			}
		}
	}
}

// Close shuts down the event bus and waits for pending handlers to complete.
func (b *Bus) Close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()

	// Wait for all in-flight handlers to complete
	b.wg.Wait()
}

// IsClosed returns whether the event bus has been closed.
func (b *Bus) IsClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

// SubscriberCount returns the total number of subscribers.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := len(b.allHandlers)
	for _, subs := range b.subscribers {
		count += len(subs)
	}
	return count
}

// SubscriberCountForType returns the number of subscribers for a specific event type.
func (b *Bus) SubscriberCountForType(eventType core.EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers[eventType])
}
