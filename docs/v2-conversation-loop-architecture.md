# AgentPipe v2: Conversation Loop Architecture

## Overview

AgentPipe v2 reimagines multi-agent orchestration as a **group chat primitive** where users and AI agents collaborate naturally in real-time. The core architecture enables fluid, context-aware conversations where agents can respond to users, react to each other, and initiate sub-conversations.

## Design Principles

1. **Group Chat First**: Model conversations like Slack/Discord, not sequential pipelines
2. **Real-time Feedback**: Show typing indicators, thinking states, and live responses
3. **Natural Turn-Taking**: Support free-form, managed, and hybrid conversation flows
4. **Agent Awareness**: Every agent has full conversation context and can reference other agents
5. **Concurrency by Default**: Agents can respond in parallel when appropriate
6. **Event-Driven**: All actions emit events for real-time UI updates and logging

---

## 1. Conversation Loop State Machine

### States

```
┌─────────────┐
│   IDLE      │ ← Initial state, waiting for input
└──────┬──────┘
       │
       │ User sends message / Agent initiates
       ▼
┌─────────────┐
│ USER_INPUT  │ ← Processing user input, determining targets
└──────┬──────┘
       │
       │ Routing complete
       ▼
┌─────────────┐
│   ROUTING   │ ← Selecting which agents should respond
└──────┬──────┘
       │
       │ Agents selected
       ▼
┌─────────────┐
│  THINKING   │ ← Agents processing, preparing responses
└──────┬──────┘
       │
       │ Responses ready
       ▼
┌─────────────┐
│ RESPONDING  │ ← Agents streaming/delivering responses
└──────┬──────┘
       │
       │ Responses complete
       ▼
┌─────────────┐
│  FEEDBACK   │ ← Other agents react, user can continue
└──────┬──────┘
       │
       │ Loop or terminate
       ▼
┌─────────────┐
│   IDLE      │
└─────────────┘
```

### State Transitions

| From | To | Trigger | Notes |
|------|-----|---------|-------|
| IDLE | USER_INPUT | User message, agent initiation | Entry point |
| USER_INPUT | ROUTING | Input parsed | Determine targets |
| ROUTING | THINKING | Agents selected | May be 0, 1, or N agents |
| THINKING | RESPONDING | First agent ready | Can be concurrent |
| RESPONDING | FEEDBACK | All agents responded | Or timeout |
| FEEDBACK | IDLE | No follow-up | Conversation pauses |
| FEEDBACK | ROUTING | Agent/user follow-up | Loop continues |

### Error Handling

```
ANY STATE → ERROR → IDLE
         (timeout, crash, user cancel)
```

---

## 2. Message Flow Architecture

### Message Types

```go
type MessageType string

const (
    UserMessage      MessageType = "user"      // User → Agents
    AgentResponse    MessageType = "agent"     // Agent → User/Agents
    SystemMessage    MessageType = "system"    // System notifications
    ThinkingIndicator MessageType = "thinking" // Agent is processing
    TypingIndicator  MessageType = "typing"    // User/Agent is typing
)

type Message struct {
    ID          string      `json:"id"`          // Unique message ID
    Type        MessageType `json:"type"`
    From        string      `json:"from"`        // User ID or Agent ID
    To          []string    `json:"to"`          // Target IDs (empty = broadcast)
    Content     string      `json:"content"`
    ReplyTo     string      `json:"reply_to"`    // Message ID being replied to
    Mentions    []string    `json:"mentions"`    // @agent references
    Timestamp   time.Time   `json:"timestamp"`
    Metadata    Metadata    `json:"metadata"`    // Tokens, cost, duration, etc.
}

type Metadata struct {
    Tokens      *TokenUsage `json:"tokens,omitempty"`
    Cost        float64     `json:"cost,omitempty"`
    Duration    int64       `json:"duration_ms,omitempty"`
    Model       string      `json:"model,omitempty"`
    StreamState string      `json:"stream_state,omitempty"` // "thinking", "responding", "complete"
}
```

### Routing Strategies

#### 1. Broadcast (Default)
User message goes to all agents unless explicitly targeted.

```
User: "What's the weather?"
  ↓
[Claude] [Gemini] [Qwen] [GPT-4] ← All receive
```

#### 2. Targeted (@mentions)
User explicitly mentions agents.

```
User: "@claude @gemini What do you think?"
  ↓
[Claude] [Gemini] ← Only these receive
```

#### 3. Reply Threading
Agents/users reply to specific messages.

```
User: "Explain recursion"
  ↓ (msg_1)
Claude: "Recursion is when a function calls itself..."
  ↓ (msg_2, reply_to: msg_1)
User: "@claude Can you show an example?"
  ↓ (msg_3, reply_to: msg_2)
Claude: [example code]
```

#### 4. Agent-to-Agent
Agents can message each other directly.

```
Claude: "@gemini Do you agree with this approach?"
  ↓
Gemini: "@claude Yes, but I'd optimize by..."
  ↓
GPT-4: "I suggest a different pattern..." (unsolicited reaction)
```

### Message Router

```go
type Router interface {
    // Route determines which agents should receive a message
    Route(msg Message, agents []Agent) ([]Agent, error)

    // CanAgentRespond checks if agent should respond based on rules
    CanAgentRespond(agent Agent, msg Message, context ConversationContext) bool
}

type RoutingRule struct {
    Type     RoutingType
    Agents   []string       // Specific agents for targeted routing
    Filter   MessageFilter  // Conditions for routing
    Priority int            // Higher priority rules override lower
}

type RoutingType string

const (
    RouteAll       RoutingType = "all"        // Broadcast to all
    RouteTargeted  RoutingType = "targeted"   // Only @mentioned or "to" field
    RouteReplies   RoutingType = "replies"    // Only agents in reply chain
    RouteReactive  RoutingType = "reactive"   // Agents react based on content relevance
)
```

---

## 3. Turn-Taking Models

### Model Comparison

| Model | Use Case | Pros | Cons |
|-------|----------|------|------|
| **Free-Form** | Brainstorming, debate | Natural, dynamic | Can be chaotic |
| **Managed** | Sequential tasks, pipelines | Predictable, ordered | Rigid, slow |
| **Hybrid** | Most conversations | Balanced control | Complex to tune |

### 1. Free-Form (Group Chat)

Agents respond whenever they have something to say. No turn enforcement.

```go
type FreeFormMode struct {
    MaxConcurrent int           // Max agents responding simultaneously
    Timeout       time.Duration // How long to wait for responses
    MinWaitTime   time.Duration // Prevent rapid-fire spam
}

// Example flow:
// User: "Design a REST API"
// ↓ (broadcast)
// [Claude thinking...] [Gemini thinking...] [GPT-4 thinking...]
// ↓ (Claude finishes first)
// Claude: "I suggest starting with OpenAPI spec..."
// ↓ (Gemini finishes)
// Gemini: "I'd add rate limiting from the start..."
// ↓ (GPT-4 finishes)
// GPT-4: "@claude Good point on OpenAPI. We should also..."
```

**Implementation:**

```go
func (c *Conversation) RunFreeForm(msg Message) error {
    // 1. Route to eligible agents
    agents, _ := c.router.Route(msg, c.agents)

    // 2. Start all agents concurrently
    responses := make(chan AgentResponse, len(agents))
    for _, agent := range agents {
        go func(a Agent) {
            c.emitEvent(ThinkingEvent{AgentID: a.ID})
            resp, _ := a.Respond(msg, c.context)
            responses <- resp
        }(agent)
    }

    // 3. Collect responses as they arrive
    for i := 0; i < len(agents); i++ {
        select {
        case resp := <-responses:
            c.addMessage(resp.Message)
            c.emitEvent(MessageEvent{Message: resp.Message})
        case <-time.After(c.mode.Timeout):
            return ErrResponseTimeout
        }
    }

    return nil
}
```

### 2. Managed (Round-Robin)

Agents respond in strict order. Current v1 behavior.

```go
type ManagedMode struct {
    Order      []string      // Agent IDs in response order
    TurnLimit  int           // Max turns before stopping
    TurnDelay  time.Duration // Pause between turns
}

// Example flow:
// User: "Explain quantum computing"
// ↓
// Claude: [responds] ← Turn 1
// ↓ (wait TurnDelay)
// Gemini: [responds] ← Turn 2
// ↓ (wait TurnDelay)
// GPT-4: [responds] ← Turn 3
// ↓ (cycle repeats or stops at TurnLimit)
```

**Implementation:**

```go
func (c *Conversation) RunManaged(msg Message) error {
    turn := 0
    for turn < c.mode.TurnLimit {
        for _, agentID := range c.mode.Order {
            agent := c.getAgent(agentID)

            c.emitEvent(ThinkingEvent{AgentID: agentID})
            resp, _ := agent.Respond(msg, c.context)

            c.addMessage(resp.Message)
            c.emitEvent(MessageEvent{Message: resp.Message})

            time.Sleep(c.mode.TurnDelay)
        }
        turn++
    }
    return nil
}
```

### 3. Hybrid (Recommended for v2)

Combines structure with flexibility. Agents can respond freely within time windows or react to specific triggers.

```go
type HybridMode struct {
    InitialResponders []string      // Agents that respond first (round-robin)
    ReactiveWindow    time.Duration // Time window for other agents to react
    MaxReactions      int           // Max reactive responses per window
    AllowSpontaneous  bool          // Can agents start new threads?
}

// Example flow:
// User: "Design a caching strategy"
// ↓
// [Round 1: Initial Responders]
// Claude: [architectural design] ← Primary
// Gemini: [performance analysis] ← Primary
// ↓ (open ReactiveWindow = 30s)
// GPT-4: "@claude What about Redis vs Memcached?" ← Reaction
// Qwen: "I'd add TTL-based eviction..." ← Reaction
// ↓ (window closes after 30s or 2 reactions)
// [Round 2: Continue or finish]
```

**Implementation:**

```go
func (c *Conversation) RunHybrid(msg Message) error {
    // Phase 1: Initial responders (sequential)
    for _, agentID := range c.mode.InitialResponders {
        agent := c.getAgent(agentID)
        resp, _ := agent.Respond(msg, c.context)
        c.addMessage(resp.Message)
        c.emitEvent(MessageEvent{Message: resp.Message})
    }

    // Phase 2: Reactive window (concurrent)
    reactionCtx, cancel := context.WithTimeout(
        context.Background(),
        c.mode.ReactiveWindow,
    )
    defer cancel()

    reactions := c.collectReactions(reactionCtx, msg)

    // Phase 3: Process reactions
    for _, reaction := range reactions {
        c.addMessage(reaction)
        c.emitEvent(MessageEvent{Message: reaction})
    }

    return nil
}

func (c *Conversation) collectReactions(
    ctx context.Context,
    triggerMsg Message,
) []Message {
    reactions := make([]Message, 0)
    reactionChan := make(chan Message, len(c.agents))

    // All agents can react concurrently
    for _, agent := range c.agents {
        go func(a Agent) {
            if c.shouldReact(a, triggerMsg) {
                resp, _ := a.Respond(triggerMsg, c.context)
                reactionChan <- resp.Message
            }
        }(agent)
    }

    // Collect up to MaxReactions within timeout
    for i := 0; i < c.mode.MaxReactions; i++ {
        select {
        case msg := <-reactionChan:
            reactions = append(reactions, msg)
        case <-ctx.Done():
            return reactions
        }
    }

    return reactions
}
```

---

## 4. Agent Awareness & Context

### Conversation Context

Every agent receives full conversation history with rich metadata:

```go
type ConversationContext struct {
    ID           string       `json:"id"`
    Messages     []Message    `json:"messages"`      // Full history
    Participants []Participant `json:"participants"` // All agents + user
    Mode         ModeConfig   `json:"mode"`          // Current turn-taking mode
    Metadata     map[string]interface{} `json:"metadata"`
}

type Participant struct {
    ID      string      `json:"id"`
    Type    string      `json:"type"`   // "user", "agent"
    Name    string      `json:"name"`
    Model   string      `json:"model"`  // For agents
    Status  string      `json:"status"` // "active", "idle", "thinking"
}

// Example context passed to agents:
{
  "id": "conv_abc123",
  "messages": [
    {
      "id": "msg_1",
      "from": "user",
      "content": "Design a caching layer",
      "timestamp": "2025-01-18T10:00:00Z"
    },
    {
      "id": "msg_2",
      "from": "claude",
      "content": "I recommend Redis with TTL-based eviction...",
      "reply_to": "msg_1",
      "timestamp": "2025-01-18T10:00:15Z"
    }
  ],
  "participants": [
    {"id": "user", "type": "user", "name": "Ahmad"},
    {"id": "claude", "type": "agent", "model": "claude-3-7-sonnet", "status": "idle"},
    {"id": "gemini", "type": "agent", "model": "gemini-2.0-flash-exp", "status": "thinking"}
  ],
  "mode": {
    "type": "hybrid",
    "initial_responders": ["claude", "gemini"]
  }
}
```

### Agent Decision Framework

Agents use context to decide:

1. **Should I respond?**
   - Am I mentioned? (`@claude`)
   - Is this a broadcast and relevant to my expertise?
   - Am I in a reply chain?
   - Has someone asked a question I can answer?

2. **What should I say?**
   - Who else has responded? (avoid redundancy)
   - What's been said already? (build on prior responses)
   - Are there disagreements? (offer tie-breaking perspective)
   - Can I add value? (new insight, different angle)

3. **Who should I address?**
   - Reply to original user?
   - Direct response to another agent? (`@gemini I disagree...`)
   - Broadcast to everyone?

**Example Agent Prompt Construction:**

```go
func (a *Agent) buildPrompt(msg Message, ctx ConversationContext) string {
    prompt := fmt.Sprintf(`You are %s (%s) in a group conversation.

PARTICIPANTS:
%s

CONVERSATION HISTORY:
%s

CURRENT MESSAGE:
From: %s
Content: %s

INSTRUCTIONS:
- You can respond to the user, reply to other agents, or stay silent
- Use @mentions to address specific agents (e.g., "@claude I agree")
- Only respond if you have something valuable to add
- Build on what others have said, don't repeat
- Be concise and collaborative

Your response:`,
        a.Name,
        a.Model,
        formatParticipants(ctx.Participants),
        formatHistory(ctx.Messages),
        msg.From,
        msg.Content,
    )

    return prompt
}
```

---

## 5. Concurrency Model

### Parallel Response Collection

```go
type ResponseCollector struct {
    timeout      time.Duration
    maxConcurrent int
    semaphore    chan struct{} // Limit concurrent agents
}

func (rc *ResponseCollector) Collect(
    agents []Agent,
    msg Message,
    ctx ConversationContext,
) ([]AgentResponse, error) {
    responses := make([]AgentResponse, 0, len(agents))
    responseChan := make(chan AgentResponse, len(agents))
    errorChan := make(chan error, len(agents))

    // Start all agents concurrently (with limit)
    for _, agent := range agents {
        rc.semaphore <- struct{}{} // Acquire slot

        go func(a Agent) {
            defer func() { <-rc.semaphore }() // Release slot

            resp, err := a.Respond(msg, ctx)
            if err != nil {
                errorChan <- err
                return
            }
            responseChan <- resp
        }(agent)
    }

    // Collect responses with timeout
    deadline := time.After(rc.timeout)
    for i := 0; i < len(agents); i++ {
        select {
        case resp := <-responseChan:
            responses = append(responses, resp)
        case err := <-errorChan:
            // Log error but continue collecting
            log.Printf("Agent error: %v", err)
        case <-deadline:
            return responses, ErrCollectionTimeout
        }
    }

    return responses, nil
}
```

### Race Condition Handling

```go
type ConversationState struct {
    mu       sync.RWMutex
    messages []Message
    status   ConversationStatus
}

func (cs *ConversationState) AddMessage(msg Message) {
    cs.mu.Lock()
    defer cs.mu.Unlock()

    cs.messages = append(cs.messages, msg)
}

func (cs *ConversationState) GetMessages() []Message {
    cs.mu.RLock()
    defer cs.mu.RUnlock()

    // Return copy to prevent mutation
    msgs := make([]Message, len(cs.messages))
    copy(msgs, cs.messages)
    return msgs
}
```

---

## 6. Event-Driven Architecture

### Event Types

```go
type Event interface {
    Type() EventType
    Timestamp() time.Time
    ConversationID() string
}

type EventType string

const (
    EventConversationStarted EventType = "conversation.started"
    EventMessageCreated      EventType = "message.created"
    EventAgentThinking       EventType = "agent.thinking"
    EventAgentTyping         EventType = "agent.typing"
    EventAgentResponded      EventType = "agent.responded"
    EventConversationPaused  EventType = "conversation.paused"
    EventConversationEnded   EventType = "conversation.ended"
    EventError               EventType = "error"
)

// Concrete event types
type MessageCreatedEvent struct {
    ConvID    string    `json:"conversation_id"`
    Message   Message   `json:"message"`
    Time      time.Time `json:"timestamp"`
}

type AgentThinkingEvent struct {
    ConvID    string    `json:"conversation_id"`
    AgentID   string    `json:"agent_id"`
    AgentName string    `json:"agent_name"`
    Time      time.Time `json:"timestamp"`
}

type AgentRespondedEvent struct {
    ConvID    string       `json:"conversation_id"`
    AgentID   string       `json:"agent_id"`
    Message   Message      `json:"message"`
    Duration  int64        `json:"duration_ms"`
    Metadata  Metadata     `json:"metadata"`
    Time      time.Time    `json:"timestamp"`
}
```

### Event Bus

```go
type EventBus interface {
    Subscribe(eventType EventType, handler EventHandler) Subscription
    Publish(event Event)
    Unsubscribe(sub Subscription)
}

type EventHandler func(event Event)

type Subscription struct {
    ID        string
    EventType EventType
    Handler   EventHandler
}

// In-memory implementation
type InMemoryEventBus struct {
    mu           sync.RWMutex
    subscribers  map[EventType][]Subscription
    eventBuffer  chan Event
}

func (bus *InMemoryEventBus) Publish(event Event) {
    bus.eventBuffer <- event
}

func (bus *InMemoryEventBus) run() {
    for event := range bus.eventBuffer {
        bus.dispatch(event)
    }
}

func (bus *InMemoryEventBus) dispatch(event Event) {
    bus.mu.RLock()
    subs := bus.subscribers[event.Type()]
    bus.mu.RUnlock()

    for _, sub := range subs {
        go sub.Handler(event) // Non-blocking handlers
    }
}
```

### Event Subscribers

**TUI Updates:**

```go
func (tui *TUI) setupEventListeners(bus EventBus) {
    bus.Subscribe(EventAgentThinking, func(e Event) {
        evt := e.(AgentThinkingEvent)
        tui.showThinkingIndicator(evt.AgentID)
    })

    bus.Subscribe(EventMessageCreated, func(e Event) {
        evt := e.(MessageCreatedEvent)
        tui.appendMessage(evt.Message)
        tui.scrollToBottom()
    })

    bus.Subscribe(EventAgentResponded, func(e Event) {
        evt := e.(AgentRespondedEvent)
        tui.hideThinkingIndicator(evt.AgentID)
        tui.updateMetrics(evt.Metadata)
    })
}
```

**Streaming Bridge:**

```go
func (bridge *StreamingBridge) setupEventListeners(bus EventBus) {
    bus.Subscribe(EventMessageCreated, func(e Event) {
        evt := e.(MessageCreatedEvent)
        bridge.emitMessageCreated(evt.Message)
    })

    bus.Subscribe(EventConversationEnded, func(e Event) {
        evt := e.(ConversationEndedEvent)
        bridge.emitConversationCompleted(evt.Summary)
    })
}
```

**Logging:**

```go
func (logger *ConversationLogger) setupEventListeners(bus EventBus) {
    bus.Subscribe(EventMessageCreated, func(e Event) {
        evt := e.(MessageCreatedEvent)
        logger.logMessage(evt.Message)
    })
}
```

---

## 7. Core Loop Implementation

### Main Conversation Loop

```go
type ConversationLoop struct {
    id          string
    state       *ConversationState
    agents      []Agent
    router      Router
    mode        TurnTakingMode
    eventBus    EventBus
    collector   *ResponseCollector
}

func (loop *ConversationLoop) Start() error {
    loop.eventBus.Publish(ConversationStartedEvent{
        ConvID:    loop.id,
        Agents:    loop.getAgentInfo(),
        Mode:      loop.mode.Type(),
        Timestamp: time.Now(),
    })

    // Enter IDLE state
    loop.state.SetStatus(StatusIdle)

    // Wait for input
    for {
        select {
        case msg := <-loop.inputChan:
            if err := loop.processMessage(msg); err != nil {
                loop.handleError(err)
            }

        case <-loop.stopChan:
            return loop.shutdown()
        }
    }
}

func (loop *ConversationLoop) processMessage(msg Message) error {
    // USER_INPUT state
    loop.state.SetStatus(StatusUserInput)
    loop.state.AddMessage(msg)
    loop.eventBus.Publish(MessageCreatedEvent{
        ConvID:  loop.id,
        Message: msg,
        Time:    time.Now(),
    })

    // ROUTING state
    loop.state.SetStatus(StatusRouting)
    agents, err := loop.router.Route(msg, loop.agents)
    if err != nil {
        return err
    }

    if len(agents) == 0 {
        // No agents to respond, back to IDLE
        loop.state.SetStatus(StatusIdle)
        return nil
    }

    // THINKING state
    loop.state.SetStatus(StatusThinking)
    for _, agent := range agents {
        loop.eventBus.Publish(AgentThinkingEvent{
            ConvID:    loop.id,
            AgentID:   agent.ID,
            AgentName: agent.Name,
            Time:      time.Now(),
        })
    }

    // RESPONDING state
    loop.state.SetStatus(StatusResponding)
    ctx := loop.state.GetContext()
    responses, err := loop.collector.Collect(agents, msg, ctx)
    if err != nil {
        return err
    }

    // Add responses to state
    for _, resp := range responses {
        loop.state.AddMessage(resp.Message)
        loop.eventBus.Publish(AgentRespondedEvent{
            ConvID:   loop.id,
            AgentID:  resp.AgentID,
            Message:  resp.Message,
            Duration: resp.Duration,
            Metadata: resp.Metadata,
            Time:     time.Now(),
        })
    }

    // FEEDBACK state
    loop.state.SetStatus(StatusFeedback)

    // Check for follow-up (agent reactions, user input)
    if loop.mode.AllowReactions() {
        loop.handleReactions(responses)
    }

    // Back to IDLE
    loop.state.SetStatus(StatusIdle)
    return nil
}

func (loop *ConversationLoop) handleReactions(
    triggerResponses []AgentResponse,
) {
    // Give agents a window to react to each other's responses
    reactionCtx, cancel := context.WithTimeout(
        context.Background(),
        loop.mode.ReactionWindow(),
    )
    defer cancel()

    reactionChan := make(chan AgentResponse, len(loop.agents))

    for _, agent := range loop.agents {
        go func(a Agent) {
            // Agent decides if it wants to react
            if reaction := a.React(triggerResponses, loop.state.GetContext()); reaction != nil {
                reactionChan <- *reaction
            }
        }(agent)
    }

    // Collect reactions
    for {
        select {
        case reaction := <-reactionChan:
            loop.state.AddMessage(reaction.Message)
            loop.eventBus.Publish(AgentRespondedEvent{
                ConvID:   loop.id,
                AgentID:  reaction.AgentID,
                Message:  reaction.Message,
                Duration: reaction.Duration,
                Metadata: reaction.Metadata,
                Time:     time.Now(),
            })

        case <-reactionCtx.Done():
            return // Reaction window closed
        }
    }
}
```

---

## 8. Code Examples

### Example 1: Simple Free-Form Conversation

```go
package main

import (
    "github.com/ASRagab/agentpipe/pkg/conversation"
    "github.com/ASRagab/agentpipe/pkg/adapters"
)

func main() {
    // Setup agents
    claude := adapters.NewClaude("claude", "claude-3-7-sonnet")
    gemini := adapters.NewGemini("gemini", "gemini-2.0-flash-exp")

    // Create conversation with free-form mode
    conv := conversation.New(
        conversation.WithMode(conversation.FreeFormMode{
            MaxConcurrent: 3,
            Timeout:       30 * time.Second,
        }),
        conversation.WithAgents(claude, gemini),
    )

    // Start conversation
    conv.Start()

    // Send user message (broadcast)
    conv.SendMessage(conversation.Message{
        From:    "user",
        Content: "What's the best way to implement caching?",
    })

    // Agents respond in parallel
    // Claude: "I recommend Redis..."
    // Gemini: "Consider TTL-based eviction..."

    // User can follow up
    conv.SendMessage(conversation.Message{
        From:    "user",
        To:      []string{"claude"},
        Content: "@claude What about cache invalidation?",
    })

    conv.Wait()
}
```

### Example 2: Hybrid Mode with Reactions

```go
func main() {
    claude := adapters.NewClaude("claude", "claude-3-7-sonnet")
    gemini := adapters.NewGemini("gemini", "gemini-2.0-flash-exp")
    gpt4 := adapters.NewOpenRouter("gpt4", "openai/gpt-4")

    // Hybrid: Claude and Gemini respond first, then others can react
    conv := conversation.New(
        conversation.WithMode(conversation.HybridMode{
            InitialResponders: []string{"claude", "gemini"},
            ReactiveWindow:    30 * time.Second,
            MaxReactions:      5,
            AllowSpontaneous:  true,
        }),
        conversation.WithAgents(claude, gemini, gpt4),
    )

    conv.Start()

    // User asks question
    conv.SendMessage(conversation.Message{
        From:    "user",
        Content: "Design a microservices architecture for an e-commerce platform",
    })

    // Flow:
    // 1. Claude responds (initial)
    // 2. Gemini responds (initial)
    // 3. Reactive window opens (30s)
    // 4. GPT-4 can react: "@claude I'd add an API gateway..."
    // 5. Claude can react back: "@gpt4 Good point, also consider..."
    // 6. Window closes or max reactions reached

    conv.Wait()
}
```

### Example 3: Event-Driven TUI Updates

```go
func (tui *TUI) Run(conv *conversation.Conversation) {
    // Subscribe to events
    conv.EventBus().Subscribe(conversation.EventAgentThinking, func(e conversation.Event) {
        evt := e.(conversation.AgentThinkingEvent)

        tui.mu.Lock()
        tui.thinkingIndicators[evt.AgentID] = true
        tui.mu.Unlock()

        tui.render() // Update UI
    })

    conv.EventBus().Subscribe(conversation.EventMessageCreated, func(e conversation.Event) {
        evt := e.(conversation.MessageCreatedEvent)

        tui.mu.Lock()
        tui.messages = append(tui.messages, evt.Message)
        tui.mu.Unlock()

        tui.render()
    })

    conv.EventBus().Subscribe(conversation.EventAgentResponded, func(e conversation.Event) {
        evt := e.(conversation.AgentRespondedEvent)

        tui.mu.Lock()
        delete(tui.thinkingIndicators, evt.AgentID)
        tui.metrics[evt.AgentID] = evt.Metadata
        tui.mu.Unlock()

        tui.render()
    })

    // Start conversation in background
    go conv.Start()

    // Handle user input
    for {
        input := tui.readInput()
        conv.SendMessage(conversation.Message{
            From:    "user",
            Content: input,
        })
    }
}
```

---

## 9. Migration Path from v1

### v1 Architecture (Current)

```
User Input → Orchestrator → Agent 1 → Agent 2 → Agent 3 → ...
             (Round-robin)
```

- Sequential, turn-based
- No agent-to-agent communication
- Limited concurrency

### v2 Architecture (Proposed)

```
                    ┌─────────────┐
                    │  Event Bus  │
                    └─────┬───────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
    ┌────────┐      ┌──────────┐     ┌────────┐
    │  TUI   │      │ Streaming│     │ Logger │
    │ Updates│      │  Bridge  │     │        │
    └────────┘      └──────────┘     └────────┘

              ┌──────────────────┐
              │ Conversation Loop│
              └────────┬─────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
         ▼             ▼             ▼
    ┌────────┐   ┌─────────┐   ┌────────┐
    │ Claude │   │ Gemini  │   │  GPT-4 │
    └────────┘   └─────────┘   └────────┘
         │             │             │
         └─────────────┴─────────────┘
                       │
              (Can message each other)
```

### Backwards Compatibility

v2 will support v1 config files with automatic mode detection:

```go
func LoadConfig(path string) (*Config, error) {
    cfg := &Config{}
    // ... load YAML

    // Auto-detect mode from v1 config
    if cfg.Orchestrator.Mode == "round-robin" {
        cfg.ConversationMode = ManagedMode{
            Order:     getAgentIDs(cfg.Agents),
            TurnLimit: cfg.Orchestrator.MaxTurns,
        }
    }

    return cfg, nil
}
```

---

## 10. Performance Considerations

### Concurrency Limits

```go
type PerformanceConfig struct {
    MaxConcurrentAgents   int           // Limit parallel agents (default: 5)
    ResponseTimeout       time.Duration // Per-agent timeout (default: 30s)
    MessageBufferSize     int           // Event buffer size (default: 100)
    ReactionWindowMax     time.Duration // Max reaction window (default: 60s)
}
```

### Memory Management

- **Message history pruning**: Keep last N messages in context (configurable)
- **Event buffer**: Bounded channel to prevent memory leaks
- **Agent pool**: Reuse agent instances (no spawn/destroy per message)

### Latency Optimization

- **Parallel thinking**: All agents start processing simultaneously
- **Stream responses**: Don't wait for full completion
- **Early termination**: Stop collecting if timeout or max reactions reached

---

## 11. Security & Safety

### Input Validation

```go
func (loop *ConversationLoop) validateMessage(msg Message) error {
    if len(msg.Content) > MaxMessageLength {
        return ErrMessageTooLong
    }

    if len(msg.To) > MaxTargets {
        return ErrTooManyTargets
    }

    if !isValidMessageType(msg.Type) {
        return ErrInvalidMessageType
    }

    return nil
}
```

### Rate Limiting

```go
type RateLimiter struct {
    userLimits  map[string]*rate.Limiter // Per-user rate limits
    agentLimits map[string]*rate.Limiter // Per-agent rate limits
}

func (rl *RateLimiter) AllowMessage(from string) bool {
    limiter := rl.getLimiter(from)
    return limiter.Allow()
}
```

### Agent Isolation

- Agents cannot execute arbitrary code
- File system access restricted
- API calls sandboxed

---

## 12. Testing Strategy

### Unit Tests

```go
func TestFreeFormMode(t *testing.T) {
    // Mock agents
    claude := &MockAgent{ID: "claude", ResponseTime: 100 * time.Millisecond}
    gemini := &MockAgent{ID: "gemini", ResponseTime: 200 * time.Millisecond}

    conv := conversation.New(
        conversation.WithMode(conversation.FreeFormMode{
            MaxConcurrent: 2,
            Timeout:       1 * time.Second,
        }),
        conversation.WithAgents(claude, gemini),
    )

    msg := conversation.Message{From: "user", Content: "Test"}
    err := conv.SendMessage(msg)

    assert.NoError(t, err)
    assert.Len(t, conv.GetMessages(), 3) // User + Claude + Gemini
}
```

### Integration Tests

```go
func TestHybridConversation(t *testing.T) {
    // Real agents (requires API keys)
    if os.Getenv("ANTHROPIC_API_KEY") == "" {
        t.Skip("Missing ANTHROPIC_API_KEY")
    }

    claude := adapters.NewClaude("claude", "claude-3-7-sonnet")
    gemini := adapters.NewGemini("gemini", "gemini-2.0-flash-exp")

    conv := conversation.New(
        conversation.WithMode(conversation.HybridMode{
            InitialResponders: []string{"claude"},
            ReactiveWindow:    10 * time.Second,
        }),
        conversation.WithAgents(claude, gemini),
    )

    conv.SendMessage(conversation.Message{
        From:    "user",
        Content: "What is 2+2?",
    })

    time.Sleep(15 * time.Second) // Wait for responses + reactions

    messages := conv.GetMessages()
    assert.GreaterOrEqual(t, len(messages), 2) // At least user + claude
}
```

---

## 13. Next Steps

### Phase 1: Core Loop (Week 1-2)
- Implement state machine
- Message routing
- Event bus
- Basic free-form mode

### Phase 2: Turn-Taking Modes (Week 3)
- Managed mode (v1 parity)
- Hybrid mode
- Configuration system

### Phase 3: Agent Awareness (Week 4)
- Context building
- Agent-to-agent messaging
- Reaction system

### Phase 4: TUI Integration (Week 5)
- Event-driven UI updates
- Real-time indicators
- User input handling

### Phase 5: Polish & Testing (Week 6)
- Performance tuning
- Integration tests
- Documentation

---

## Appendix: Glossary

- **Broadcast**: Message sent to all agents
- **Targeted**: Message sent to specific agents (@mentions or "to" field)
- **Reaction**: Agent response triggered by another agent's message
- **Turn**: One complete cycle of user input → agent responses → feedback
- **Context**: Full conversation history and metadata available to agents
- **Routing**: Process of determining which agents should respond to a message
- **Mode**: Turn-taking strategy (free-form, managed, hybrid)
- **Event**: Notification of state change or action in the conversation loop
