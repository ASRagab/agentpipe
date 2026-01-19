// Package manager provides the ConversationManager for orchestrating multi-agent AI conversations.
//
// The ConversationManager is the central component that coordinates agents, handles
// message routing, manages state, and provides features like auto-save, graceful
// degradation, and conversation resumption.
//
// # Quick Start
//
// Basic usage with two agents:
//
//	// Load configuration
//	cfg, err := config.LoadConfig("conversation.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Initialize agents
//	agents, err := cfg.InitializeAgents()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create manager with default config
//	managerCfg := manager.DefaultConfig()
//	managerCfg.Timeout = 30 * time.Second
//
//	mgr, err := manager.NewConversationManager(managerCfg, agents, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer mgr.Close()
//
//	// Start conversation
//	mgr.Start()
//
//	// Send user message and get responses from all agents
//	responses, err := mgr.SendUserMessage(ctx, "Hello, how are you?")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, msg := range responses {
//		fmt.Printf("%s: %s\n", msg.AgentName, msg.Content)
//	}
//
//	// Complete the conversation
//	mgr.Complete()
//
// # Configuration
//
// The Config struct controls manager behavior:
//
//	config := manager.Config{
//		// Maximum time to wait for agent responses
//		Timeout: 60 * time.Second,
//
//		// Persistence settings
//		Persistence: manager.PersistenceConfig{
//			Enabled:      true,
//			SaveDir:      "/path/to/conversations",
//			SaveInterval: 5 * time.Minute,  // Auto-save interval
//		},
//
//		// Graceful degradation (continues if some agents fail)
//		GracefulDegradation: manager.GracefulDegradationConfig{
//			Enabled:                  true,
//			PauseOnAllFailed:         true,   // Pause if ALL agents fail
//			RetryFailedOnNextMessage: true,   // Retry failed agents
//			EmitSystemMessages:       true,   // Show "Agent unavailable" messages
//		},
//	}
//
// Use DefaultConfig() for sensible defaults:
//
//	config := manager.DefaultConfig()
//	config.Timeout = 45 * time.Second
//	config.Persistence.Enabled = true
//
// # Event Subscriptions
//
// Subscribe to conversation events for reactive UI updates:
//
//	// Subscribe to agent typing (show indicator)
//	mgr.Subscribe(core.EventAgentTyping, func(event core.Event) {
//		data := event.Data.(core.AgentTypingData)
//		fmt.Printf("%s is typing...\n", data.AgentName)
//	})
//
//	// Subscribe to new messages
//	mgr.Subscribe(core.EventMessageCreated, func(event core.Event) {
//		msg := event.Data.(core.Message)
//		fmt.Printf("[%s] %s\n", msg.AgentName, msg.Content)
//	})
//
//	// Subscribe to all events for logging
//	mgr.SubscribeAll(func(event core.Event) {
//		log.Printf("Event: %s", event.Type)
//	})
//
// # Graceful Degradation
//
// When graceful degradation is enabled, the conversation continues even if some
// agents fail. This provides a better user experience:
//
//	config := manager.DefaultConfig()
//	config.GracefulDegradation = manager.GracefulDegradationConfig{
//		Enabled:                  true,
//		PauseOnAllFailed:         true,   // Only pause if ALL agents fail
//		RetryFailedOnNextMessage: true,   // Auto-retry failed agents
//		EmitSystemMessages:       true,   // Show "Agent unavailable"
//	}
//
//	// Check failed agents
//	failed := mgr.GetFailedAgents()
//	for _, info := range failed {
//		fmt.Printf("%s failed: %s (retries: %d)\n",
//			info.AgentName, info.LastError, info.RetryCount)
//	}
//
//	// Check if conversation is paused due to all agents failing
//	if mgr.IsPaused() {
//		// Resume with retry
//		mgr.ResumeConversation(false)  // false = keep failed agent list
//		mgr.ResetCircuitBreakers()      // Reset circuit breakers
//	}
//
// # Persistence
//
// Conversations can be saved and resumed:
//
//	// Manual save
//	filePath, err := mgr.Save()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Saved to: %s\n", filePath)
//
//	// Export to Markdown
//	err = mgr.ExportToMarkdown("/path/to/conversation.md")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Resume a conversation
//	err = mgr.Resume("latest")           // Resume most recent
//	err = mgr.Resume("abc123")           // Resume by ID prefix
//	err = mgr.Resume("/path/to/file.json") // Resume from file
//
//	// Check if conversation was resumed
//	if mgr.IsResumed() {
//		fmt.Println("Continuing previous conversation...")
//	}
//
// Auto-save happens automatically when enabled in config. It triggers:
//   - After each agent response
//   - On the configured interval (if SaveInterval > 0)
//
// # Parallel Execution
//
// By default, the manager executes all agents in parallel:
//
//	// All agents respond concurrently
//	responses, err := mgr.SendUserMessage(ctx, "What's your opinion?")
//
//	// responses contains messages from all agents
//	// Order may vary based on response times
//
// # Accessing State
//
// Query conversation state at any time:
//
//	// Get all messages
//	messages := mgr.GetMessages()
//
//	// Get conversation object
//	conv := mgr.GetConversation()
//
//	// Get participating agents
//	agents := mgr.GetAgents()
//
//	// Get agent statuses
//	statuses := mgr.GetAgentStatus()
//	for id, status := range statuses {
//		fmt.Printf("Agent %s: %s\n", id, status)
//	}
//
//	// Get summary statistics
//	summary := mgr.Summary()
//	fmt.Printf("Messages: %d, Tokens: %d, Cost: $%.4f\n",
//		summary.MessageCount, summary.TotalTokens, summary.TotalCost)
//
//	// Check available agent count (considering circuit breakers)
//	available := mgr.GetAvailableAgentCount()
//	fmt.Printf("%d agents available\n", available)
//
// # Lifecycle
//
// Proper lifecycle management:
//
//	// Create manager
//	mgr, err := manager.NewConversationManager(config, agents, eventBus)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer mgr.Close()  // Always close to cleanup resources
//
//	// Start conversation (emits ConversationStarted event)
//	mgr.Start()
//
//	// ... send messages ...
//
//	// Complete normally (emits ConversationCompleted event)
//	mgr.Complete()
//
// # Thread Safety
//
// The ConversationManager is thread-safe. Multiple goroutines can:
//   - Read state concurrently (GetMessages, GetAgents, etc.)
//   - Subscribe to events
//   - Query status
//
// However, SendUserMessage should be called sequentially (one at a time)
// to maintain conversation coherence.
//
// # Error Handling
//
// Two sentinel errors are defined:
//
//	manager.ErrAllAgentsFailed  // All agents failed (conversation paused)
//	manager.ErrConversationPaused  // Attempted to send while paused
//
// Example handling:
//
//	responses, err := mgr.SendUserMessage(ctx, "Hello")
//	if errors.Is(err, manager.ErrAllAgentsFailed) {
//		fmt.Println("All agents are unavailable. Try again later.")
//		mgr.ResumeConversation(true)
//		mgr.ResetCircuitBreakers()
//	} else if errors.Is(err, manager.ErrConversationPaused) {
//		fmt.Println("Conversation is paused.")
//		mgr.ResumeConversation(true)
//	} else if err != nil {
//		log.Fatal(err)
//	}
package manager
