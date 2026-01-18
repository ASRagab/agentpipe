// Package core provides the foundational domain types for the AgentPipe v2 architecture.
//
// This package defines the core data structures used throughout the v2 system:
// agents, messages, conversations, events, and error handling. These types form
// the backbone of multi-agent AI conversation orchestration.
//
// # Architecture Overview
//
// The v2 architecture is built around several key concepts:
//
//   - Agent: An AI participant in a conversation (e.g., Claude, GPT-4, Gemini)
//   - Message: A single message in a conversation from a user, agent, or system
//   - Conversation: A session containing messages and participating agents
//   - Event: A system event for pub/sub notification (message created, agent typing, etc.)
//
// # Agents
//
// An Agent represents an AI participant with configuration:
//
//	agent := core.NewAgent(
//		"claude-1",           // Unique ID
//		"openrouter",         // Provider type
//		"Claude",             // Display name
//		"anthropic/claude-3-sonnet-20240229",  // Model
//		"openrouter",         // Adapter name
//	).WithSystemPrompt("You are a helpful assistant.").
//	  WithTemperature(0.7).
//	  WithMaxTokens(4096)
//
// Agent states track runtime information:
//
//	state := core.NewAgentState(agent)
//	state.SetTyping()                    // Agent is generating
//	state.RecordMessage(1500, 0.015)     // Record token usage and cost
//	state.SetIdle()                      // Agent is ready
//
// # Messages
//
// Messages represent communication in a conversation:
//
//	// User message
//	userMsg := core.NewUserMessage("Hello, can you help me?")
//
//	// Agent response with metrics
//	metrics := &core.Metrics{
//		Duration:     2 * time.Second,
//		InputTokens:  50,
//		OutputTokens: 150,
//		TotalTokens:  200,
//		Model:        "claude-3-sonnet",
//		Cost:         0.003,
//	}
//	agentMsg := core.NewAgentMessage("claude-1", "Claude", "Of course! How can I help?", metrics)
//
//	// System message
//	sysMsg := core.NewSystemMessage("Claude is temporarily unavailable")
//
// Messages support streaming:
//
//	pending := core.NewPendingAgentMessage("claude-1", "Claude")
//	pending.SetStreaming()
//	pending.AppendContent("Hello")
//	pending.AppendContent(" there!")
//	pending.Complete("Hello there!", metrics)
//
// # Conversations
//
// Conversations manage the full session:
//
//	agents := []core.Agent{agent1, agent2}
//	conv := core.NewConversation(agents)
//
//	conv.AddMessage(userMsg)
//	conv.AddMessage(agentMsg)
//
//	// Query messages
//	all := conv.GetMessages()
//	last := conv.GetLastMessage()
//	byAgent := conv.GetMessagesByAgent("claude-1")
//
//	// Get statistics
//	summary := conv.Summary()
//	fmt.Printf("Total tokens: %d, Cost: $%.4f\n", summary.TotalTokens, summary.TotalCost)
//
//	// Lifecycle
//	conv.Pause()   // Temporarily halt
//	conv.Resume()  // Continue
//	conv.Complete() // Mark as finished
//
// # Events
//
// Events enable reactive programming patterns:
//
//	// Message events
//	event := core.NewMessageCreatedEvent(msg)
//	chunk := core.MessageChunk{
//		MessageID: msg.ID,
//		AgentID:   "claude-1",
//		Content:   "partial response",
//		Index:     0,
//	}
//	chunkEvent := core.NewMessageChunkEvent(chunk)
//
//	// Agent lifecycle events
//	typingEvent := core.NewAgentTypingEvent("claude-1", "Claude")
//	doneEvent := core.NewAgentDoneEvent("claude-1", "Claude", msg)
//	errorEvent := core.NewAgentErrorEvent("claude-1", "Claude", "timeout")
//
//	// Conversation events
//	startEvent := core.NewConversationStartedEvent(conv.ID, agents)
//	endEvent := core.NewConversationCompletedEvent(conv.ID, summary)
//
// # Error Handling
//
// Structured error types enable proper user feedback:
//
//	// Classify an error
//	errType := core.ClassifyError("connection refused")
//	// Returns: core.ErrorTypeNetwork
//
//	// Create error info for display
//	info := core.NewAgentErrorInfo(
//		core.ErrorTypeTimeout,
//		"request timed out after 30s",
//		"claude-1",
//		"Claude",
//	)
//
//	// Create error message for conversation
//	errMsg := core.NewErrorMessage(info)
//
// Error types include:
//   - ErrorTypeTimeout: Operation timed out
//   - ErrorTypeRateLimit: Rate limit exceeded
//   - ErrorTypeNetwork: Connection/DNS errors
//   - ErrorTypeAuthentication: API key issues
//   - ErrorTypeAPI: Server-side errors
//   - ErrorTypeInternal: Internal errors
//
// # Thread Safety
//
// Individual types in this package are NOT thread-safe. Thread safety is
// provided at higher levels (ConversationManager, Pool) through synchronization.
// When accessing types from multiple goroutines, use appropriate locking.
//
// # Serialization
//
// All types support JSON and YAML serialization for persistence and transport:
//
//	data, err := json.Marshal(conversation)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	var loaded core.Conversation
//	err = json.Unmarshal(data, &loaded)
//
// See the persistence package for higher-level save/load operations.
package core
