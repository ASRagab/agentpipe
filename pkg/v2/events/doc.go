// Package events provides a thread-safe publish/subscribe event bus for the v2 architecture.
//
// The event bus enables decoupled, reactive programming patterns throughout AgentPipe.
// Components can publish events when state changes occur, and other components can
// subscribe to those events without direct coupling.
//
// # Architecture
//
// The event system consists of:
//
//   - Event types defined in the core package (core.Event, core.EventType)
//   - The EventBus interface defining pub/sub operations
//   - The Bus implementation providing thread-safe event dispatch
//   - Handler functions that process events
//
// # Creating an Event Bus
//
//	bus := events.NewBus()
//	defer bus.Close()  // Always close when done
//
// # Subscribing to Events
//
// Subscribe to specific event types:
//
//	// Subscribe to message created events
//	unsubscribe := bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
//		if msg, ok := event.Data.(core.Message); ok {
//			fmt.Printf("New message from %s: %s\n", msg.AgentName, msg.Content)
//		}
//	})
//
//	// Later, unsubscribe when no longer needed
//	defer unsubscribe()
//
// Subscribe to all events:
//
//	unsubscribe := bus.SubscribeAll(func(event core.Event) {
//		fmt.Printf("Event: %s at %s\n", event.Type, event.Timestamp)
//	})
//
// # Publishing Events
//
// Publish events using the core event constructors:
//
//	// Message events
//	bus.Publish(core.NewMessageCreatedEvent(message))
//	bus.Publish(core.NewMessageChunkEvent(chunk))
//
//	// Agent lifecycle events
//	bus.Publish(core.NewAgentTypingEvent("agent-1", "Claude"))
//	bus.Publish(core.NewAgentDoneEvent("agent-1", "Claude", message))
//	bus.Publish(core.NewAgentErrorEvent("agent-1", "Claude", "timeout"))
//	bus.Publish(core.NewAgentCancelledEvent("agent-1", "Claude", "user cancelled"))
//
//	// Conversation events
//	bus.Publish(core.NewConversationStartedEvent(convID, agents))
//	bus.Publish(core.NewConversationSavedEvent(convID, "/path/to/file.json"))
//	bus.Publish(core.NewConversationCompletedEvent(convID, summary))
//	bus.Publish(core.NewConversationErrorEvent(convID, "all agents failed"))
//
//	// Health check events
//	bus.Publish(core.NewAgentHealthyEvent("agent-1", "Claude", 500*time.Millisecond))
//	bus.Publish(core.NewAgentUnhealthyEvent("agent-1", "Claude", "connection refused"))
//	bus.Publish(core.NewPreflightCompletedEvent(true, 3, 0, 2*time.Second, nil))
//
// # Event Types
//
// The following event types are defined in core:
//
//	// Message events
//	core.EventMessageCreated   // New message added to conversation
//	core.EventMessageChunk     // Streaming chunk received
//
//	// Agent lifecycle events
//	core.EventAgentTyping      // Agent started generating
//	core.EventAgentDone        // Agent finished generating
//	core.EventAgentError       // Agent encountered an error
//	core.EventAgentCancelled   // Agent request was cancelled
//
//	// Conversation events
//	core.EventConversationStarted   // Conversation began
//	core.EventConversationSaved     // Conversation was persisted
//	core.EventConversationCompleted // Conversation ended normally
//	core.EventConversationError     // Conversation ended with error
//
//	// Health check events
//	core.EventAgentHealthy        // Agent passed health check
//	core.EventAgentUnhealthy      // Agent failed health check
//	core.EventPreflightCompleted  // Preflight checks finished
//
// # Event Data Types
//
// Each event type has a corresponding data structure:
//
//	// Access typed event data
//	bus.Subscribe(core.EventAgentDone, func(event core.Event) {
//		if data, ok := event.Data.(core.AgentDoneData); ok {
//			fmt.Printf("%s responded with %d tokens\n",
//				data.AgentName,
//				data.Message.Metrics.TotalTokens)
//		}
//	})
//
//	// Streaming chunks
//	bus.Subscribe(core.EventMessageChunk, func(event core.Event) {
//		if chunk, ok := event.Data.(core.MessageChunk); ok {
//			fmt.Printf("[%s] chunk %d: %s\n",
//				chunk.AgentName,
//				chunk.Index,
//				chunk.Content)
//		}
//	})
//
// # Thread Safety
//
// The Bus implementation is fully thread-safe:
//   - Multiple goroutines can safely publish and subscribe concurrently
//   - Handlers are dispatched asynchronously in separate goroutines
//   - Handlers include panic recovery to prevent cascading failures
//   - Close() waits for all in-flight handlers to complete
//
// # Error Handling in Handlers
//
// Handler panics are recovered and logged, preventing one bad handler from
// affecting others:
//
//	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
//		// Even if this panics, other handlers continue
//		panic("something went wrong")
//	})
//
// However, it's best practice to handle errors properly:
//
//	bus.Subscribe(core.EventMessageCreated, func(event core.Event) {
//		msg, ok := event.Data.(core.Message)
//		if !ok {
//			log.Printf("unexpected event data type: %T", event.Data)
//			return
//		}
//		// Process message...
//	})
//
// # Integration with ConversationManager
//
// The ConversationManager uses an event bus internally and exposes subscription:
//
//	manager, _ := manager.NewConversationManager(config, agents, nil)
//
//	// Subscribe to specific events
//	manager.Subscribe(core.EventMessageCreated, func(event core.Event) {
//		// Handle new messages
//	})
//
//	// Subscribe to all events
//	manager.SubscribeAll(func(event core.Event) {
//		// Handle all events
//	})
//
// # TUI Integration
//
// The TUI uses events to update the display reactively:
//
//	bus.Subscribe(core.EventAgentTyping, func(event core.Event) {
//		if data, ok := event.Data.(core.AgentTypingData); ok {
//			tui.ShowTypingIndicator(data.AgentName)
//		}
//	})
//
//	bus.Subscribe(core.EventMessageChunk, func(event core.Event) {
//		if chunk, ok := event.Data.(core.MessageChunk); ok {
//			tui.AppendToMessage(chunk.MessageID, chunk.Content)
//		}
//	})
//
// # Monitoring
//
// Check bus state for debugging:
//
//	count := bus.SubscriberCount()
//	fmt.Printf("Total subscribers: %d\n", count)
//
//	countForType := bus.SubscriberCountForType(core.EventMessageCreated)
//	fmt.Printf("Message subscribers: %d\n", countForType)
//
//	if bus.IsClosed() {
//		fmt.Println("Bus has been closed")
//	}
package events
