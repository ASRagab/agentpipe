# Multi-Agent Conversation Systems: Research & Best Practices

**Research Date:** January 18, 2026
**Project:** AgentPipe v2 Architecture
**Purpose:** Research proven patterns for multi-agent systems and group chat architectures
**Status:** Reference Documentation

---

## Executive Summary

This document synthesizes research on multi-agent conversation systems, group chat architectures, and real-time collaborative platforms to inform AgentPipe v2's conversation management design. The research focuses on proven patterns from production systems, academic research on turn-taking, and event-driven architectures that scale.

**Key Findings:**
- **Event-driven architecture** is essential for multi-party conversations at scale
- **Turn-taking protocols** prevent chaos through coordination strategies
- **State management** patterns from collaborative systems apply directly to multi-agent orchestration
- **Real-time streaming** requires careful backpressure and buffering strategies
- **Message ordering guarantees** are critical for conversation coherence

---

## Table of Contents

1. [Multi-Agent AI Systems](#1-multi-agent-ai-systems)
2. [Group Chat Protocols](#2-group-chat-protocols)
3. [Real-Time Collaborative Systems](#3-real-time-collaborative-systems)
4. [Turn-Taking in Multi-Party Conversations](#4-turn-taking-in-multi-party-conversations)
5. [Event-Driven Architectures](#5-event-driven-architectures)
6. [TUI Group Chat Implementations](#6-tui-group-chat-implementations)
7. [Proven Patterns for AgentPipe](#7-proven-patterns-for-agentpipe)
8. [Anti-Patterns to Avoid](#8-anti-patterns-to-avoid)
9. [Implementation Recommendations](#9-implementation-recommendations)

---

## 1. Multi-Agent AI Systems

### 1.1 LangGraph (LangChain)

**Architecture Pattern:**
```
State Graph Pattern
├── Nodes: Processing steps (agents, tools, decision points)
├── Edges: Conditional routing between nodes
├── State: Shared state passed through graph
└── Checkpointing: Save/resume at any node
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **State Checkpointing**: Save conversation state at turn boundaries
2. **Conditional Routing**: Agent selection based on state conditions
3. **Graph Visualization**: Export conversation flow as diagrams
4. **Message Filtering**: Filter history by relevance to current agent

**❌ What to Avoid:**
1. **Graph Complexity**: Don't require users to define complex graphs
2. **Heavy Abstractions**: Keep mental model simple (turns, not nodes)
3. **Framework Lock-in**: Maintain vendor neutrality

**Proven Pattern:**
```go
// State checkpoint at turn boundaries
type TurnCheckpoint struct {
    TurnNumber    int
    ConversationID ConversationID
    State         ConversationState
    Timestamp     time.Time
}

// Allow resumption from any checkpoint
func (o *Orchestrator) ResumeFromCheckpoint(checkpoint TurnCheckpoint) error {
    o.state = checkpoint.State
    o.currentTurn = checkpoint.TurnNumber
    return o.Continue()
}
```

### 1.2 AutoGen (Microsoft)

**Architecture Pattern:**
```
Conversational Pattern
├── GroupChat: Multi-agent conversation manager
├── Speaker Selection: Round-robin, auto, manual
├── Human-in-the-loop: User approval for actions
└── Code Execution: Sandboxed code interpreter
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Speaker Selection Strategies**: Multiple modes (round-robin, auto, manual)
2. **Human-in-the-Loop**: User can inject messages at any time (already implemented)
3. **Termination Conditions**: Configurable end conditions (max turns, keywords)
4. **Agent Roles**: Agents can have specialized roles (coder, reviewer, planner)

**❌ What to Avoid:**
1. **Synchronous Blocking**: Don't block entire conversation on one agent
2. **Complex State Machines**: Keep turn logic simple and predictable
3. **Hidden Magic**: Make agent selection transparent and debuggable

**Proven Pattern:**
```go
// Multiple speaker selection strategies
type SpeakerSelectionStrategy interface {
    SelectNext(ctx context.Context, state State) (AgentID, error)
    Name() string
}

// Round-robin strategy (already implemented in v1)
type RoundRobinStrategy struct {
    currentIndex int
    agents       []AgentID
}

// Auto strategy: agents self-select based on context
type AutoStrategy struct {
    selectPrompt string
}

func (s *AutoStrategy) SelectNext(ctx context.Context, state State) (AgentID, error) {
    // Ask a meta-agent which agent should respond next
    // Based on conversation context and agent capabilities
    return s.aiPoweredSelection(state)
}
```

### 1.3 CrewAI

**Architecture Pattern:**
```
Role-Based Pattern
├── Crew: Team of agents with roles
├── Tasks: Sequential or parallel execution
├── Processes: Sequential, hierarchical, consensus
└── Tools: Shared capabilities agents can use
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Agent Roles**: Assign roles like "researcher", "coder", "reviewer"
2. **Task Hierarchies**: Support sequential and parallel task execution
3. **Shared Context**: Agents share access to conversation artifacts
4. **Process Types**: Different orchestration modes for different use cases

**❌ What to Avoid:**
1. **Heavyweight Task System**: Keep tasks lightweight (a turn is a task)
2. **Complex Dependencies**: Avoid complex task dependency graphs
3. **Role Rigidity**: Allow agents to be flexible, not locked into roles

**Proven Pattern:**
```go
// Agent roles as metadata
type AgentRole string

const (
    RoleResearcher AgentRole = "researcher"
    RoleCoder      AgentRole = "coder"
    RoleReviewer   AgentRole = "reviewer"
    RolePlanner    AgentRole = "planner"
    RoleGeneral    AgentRole = "general"
)

type Agent struct {
    ID   AgentID
    Type AgentType
    Role AgentRole  // New in v2
    // ... other fields
}

// Role-based selection strategy
type RoleBasedStrategy struct {
    requiredRole AgentRole
}

func (s *RoleBasedStrategy) SelectNext(ctx context.Context, state State) (AgentID, error) {
    // Select first available agent with required role
    for _, agent := range state.AvailableAgents() {
        if agent.Role == s.requiredRole {
            return agent.ID, nil
        }
    }
    return "", ErrNoAgentWithRole
}
```

### 1.4 Common Patterns Across Systems

**Pattern 1: Message History Management**
```
All systems implement some form of:
├── Full History: Keep all messages
├── Sliding Window: Last N messages
├── Relevance Filtering: Filter by semantic relevance
└── Summarization: Compress old messages
```

**Pattern 2: Agent Coordination**
```
Common coordination patterns:
├── Centralized: One orchestrator controls all
├── Decentralized: Agents coordinate peer-to-peer
├── Hierarchical: Leader agents coordinate workers
└── Market-based: Agents "bid" to respond
```

**Pattern 3: Error Handling**
```
Resilience patterns:
├── Retry with backoff (already in AgentPipe v1)
├── Skip failed agents and continue
├── Fallback to simpler agent
└── Human escalation
```

---

## 2. Group Chat Protocols

### 2.1 XMPP (Extensible Messaging and Presence Protocol)

**Architecture Pattern:**
```
Federated Client-Server Model
├── Client ←→ Server: XML streams
├── Server ←→ Server: Federation
├── Multi-User Chat (XEP-0045): Group chat extension
└── Presence: Real-time availability
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Presence System**: Track agent availability/health in real-time
2. **Stanza Types**: Different message types (message, presence, iq)
3. **Addressing**: Unique identifiers for agents (already implemented)
4. **Room Moderation**: Conversation-level controls

**❌ What to Avoid:**
1. **XML Overhead**: Use efficient serialization (JSON, Protocol Buffers)
2. **Federation Complexity**: Start with single-instance, add later if needed
3. **Synchronous Stanzas**: Use async event-driven model

**Proven Pattern:**
```go
// Agent presence tracking
type AgentPresence struct {
    AgentID       AgentID
    Status        PresenceStatus  // available, away, busy, offline
    Since         time.Time
    LastHeartbeat time.Time
}

type PresenceStatus string

const (
    PresenceAvailable PresenceStatus = "available"
    PresenceAway      PresenceStatus = "away"
    PresenceBusy      PresenceStatus = "busy"
    PresenceOffline   PresenceStatus = "offline"
)

// Presence updates via event bus
type PresenceUpdated struct {
    AgentID   AgentID
    Previous  PresenceStatus
    Current   PresenceStatus
    Timestamp time.Time
}
```

### 2.2 Matrix Protocol

**Architecture Pattern:**
```
Decentralized Event Graph
├── Events: Immutable event DAG
├── State Resolution: Merge conflicting states
├── Rooms: Shared conversation spaces
└── Federation: Server-to-server sync
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Event Sourcing**: Store conversation as immutable events
2. **DAG Structure**: Support branching conversations (multi-window TUI)
3. **State Snapshots**: Periodic state snapshots for performance
4. **Room Concept**: Conversations as first-class entities

**❌ What to Avoid:**
1. **Complex Conflict Resolution**: Single orchestrator = no conflicts
2. **Federation Overhead**: Not needed for single-instance
3. **Cryptographic Signing**: Overkill for dev tool (maybe later for audit)

**Proven Pattern:**
```go
// Event-sourced conversation
type ConversationEvent interface {
    EventID() EventID
    Timestamp() time.Time
    AgentID() AgentID
}

type MessageCreatedEvent struct {
    ID        EventID
    Time      time.Time
    Agent     AgentID
    Content   string
    ParentID  EventID  // For threading/branching
}

type ConversationState struct {
    events []ConversationEvent
}

// Rebuild state by replaying events
func (s *ConversationState) Replay() State {
    state := NewEmptyState()
    for _, event := range s.events {
        state.Apply(event)
    }
    return state
}
```

### 2.3 Slack Architecture (Public Knowledge)

**Architecture Pattern:**
```
Channel-Based Model
├── Channels: Topic-based conversations
├── Threads: Nested reply chains
├── Real-time Gateway: WebSocket for live updates
└── Message Formatting: Rich text, attachments, blocks
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Threading**: Support threaded conversations (multi-window TUI idea)
2. **Rich Formatting**: Markdown, code blocks, artifacts (already in v1)
3. **Real-time Updates**: WebSocket/SSE for TUI (streaming bridge in v1)
4. **Reactions**: Emoji reactions to messages (nice-to-have)

**❌ What to Avoid:**
1. **Channel Proliferation**: One conversation at a time is simpler
2. **Complex Formatting**: Keep formatting simple for AI output
3. **Synchronous API**: Use async events for scalability

**Proven Pattern:**
```go
// Message threading for nested conversations
type Message struct {
    ID        MessageID
    ParentID  *MessageID  // nil for top-level, set for replies
    Content   string
    AgentID   AgentID
    Timestamp time.Time
}

// Thread-aware message retrieval
func (c *Conversation) GetThread(parentID MessageID) []Message {
    var thread []Message
    for _, msg := range c.messages {
        if msg.ParentID != nil && *msg.ParentID == parentID {
            thread = append(thread, msg)
        }
    }
    return thread
}
```

---

## 3. Real-Time Collaborative Systems

### 3.1 Figma (Multiplayer Graphics)

**Architecture Pattern:**
```
Operational Transformation (OT)
├── Client-side prediction: Immediate local updates
├── Server reconciliation: Merge concurrent edits
├── Conflict-free data types: CRDTs
└── Cursor broadcasting: Show other users' cursors
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Optimistic Updates**: Show agent responses immediately in TUI
2. **Presence Indicators**: Show which agent is "typing" (processing)
3. **Cursor/Focus**: Highlight active agent in TUI
4. **Undo/Redo**: Support conversation rollback (time-travel debugging)

**❌ What to Avoid:**
1. **OT Complexity**: Not needed for sequential conversations
2. **CRDTs**: Overkill for single-orchestrator model
3. **Real-time Sync**: Agents don't edit in parallel (sequential turns)

**Proven Pattern:**
```go
// Agent activity indicators
type AgentActivity struct {
    AgentID     AgentID
    Status      ActivityStatus
    StartedAt   time.Time
}

type ActivityStatus string

const (
    ActivityIdle       ActivityStatus = "idle"
    ActivityProcessing ActivityStatus = "processing"
    ActivityTyping     ActivityStatus = "typing"  // For streaming
)

// Event for TUI updates
type AgentActivityChanged struct {
    AgentID  AgentID
    Previous ActivityStatus
    Current  ActivityStatus
}
```

### 3.2 Google Docs (Collaborative Editing)

**Architecture Pattern:**
```
OT-based Real-time Editing
├── Document snapshots: Periodic checkpoints
├── Delta updates: Send only changes
├── Cursor positions: Track all users
└── Suggestion mode: Non-destructive edits
```

**Key Insights for AgentPipe:**

**✅ What to Adopt:**
1. **Delta Updates**: Send incremental message chunks (streaming)
2. **Snapshots**: Periodic conversation checkpoints
3. **Suggestion Mode**: Agent can suggest edits to prior messages
4. **Version History**: Full conversation replay from events

**❌ What to Avoid:**
1. **Character-level OT**: Messages are atomic units, not characters
2. **Real-time Merging**: Sequential turns = no merge conflicts
3. **Cursor Sync**: Not applicable to agent conversations

**Proven Pattern:**
```go
// Streaming message chunks (like typing indicator)
type MessageChunk struct {
    MessageID MessageID
    AgentID   AgentID
    Chunk     string
    Index     int
    IsFinal   bool
}

// Stream message as agent generates it
func (a *Agent) StreamMessage(ctx context.Context, req MessageRequest) (<-chan MessageChunk, error) {
    chunks := make(chan MessageChunk)
    go func() {
        defer close(chunks)
        // Stream response in chunks
        for chunk := range a.generateResponse(req) {
            chunks <- chunk
        }
    }()
    return chunks, nil
}
```

---

## 4. Turn-Taking in Multi-Party Conversations

### 4.1 Academic Research Findings

**Research Source:** Conversation Analysis (Sacks, Schegloff, Jefferson)

**Key Findings:**
1. **Turn-Taking Rules**: Natural conversations follow implicit rules
2. **Turn Allocation**: Speaker selection vs. self-selection
3. **Overlap Avoidance**: Mechanisms to prevent simultaneous talking
4. **Repair Mechanisms**: How conversations handle errors

**Conversation Turn-Taking Rules:**
```
Rule 1: Current speaker selects next (directed question)
Rule 2: Next speaker self-selects (volunteer)
Rule 3: Current speaker continues (no one volunteers)
```

**AgentPipe Mapping:**
```go
// Rule 1: Directed turn (agent mentions another agent)
type DirectedTurn struct {
    FromAgent AgentID
    ToAgent   AgentID
}

// Rule 2: Self-selection (reactive mode in v1)
type SelfSelectionTurn struct {
    AvailableAgents []AgentID
    SelectionMethod string  // random, priority, first-available
}

// Rule 3: Continuation (agent keeps turn)
type ContinuationTurn struct {
    CurrentAgent AgentID
    Reason       string  // "multi-part response", "follow-up"
}
```

### 4.2 Floor Control Mechanisms

**Pattern:** Token-based floor control (who has permission to speak)

```
Token Passing Protocol:
1. Only agent with "token" can send messages
2. Token passed explicitly (directed) or implicitly (rules)
3. Orchestrator enforces token rules
4. Timeout recovers from stuck tokens
```

**Implementation:**
```go
// Floor token (who can speak)
type FloorToken struct {
    Holder      AgentID
    AcquiredAt  time.Time
    MaxDuration time.Duration
}

// Orchestrator manages floor
type FloorManager struct {
    token   *FloorToken
    waiting []AgentID  // Queue of agents wanting to speak
}

func (f *FloorManager) AcquireFloor(agentID AgentID, timeout time.Duration) error {
    if f.token != nil && f.token.Holder != agentID {
        // Token held by another agent
        if time.Since(f.token.AcquiredAt) > f.token.MaxDuration {
            // Token expired, reclaim it
            f.token = nil
        } else {
            // Add to waiting queue
            f.waiting = append(f.waiting, agentID)
            return ErrFloorBusy
        }
    }

    // Grant floor
    f.token = &FloorToken{
        Holder:      agentID,
        AcquiredAt:  time.Now(),
        MaxDuration: timeout,
    }
    return nil
}
```

### 4.3 Overlap Prevention

**Problem:** Multiple agents responding simultaneously creates chaos

**Solution:** Centralized coordination (already in AgentPipe v1)

```go
// Prevent concurrent agent responses
type TurnLock struct {
    mu            sync.Mutex
    activeAgent   AgentID
    activeSince   time.Time
}

func (l *TurnLock) AcquireTurn(agentID AgentID) error {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.activeAgent != "" {
        return ErrTurnInProgress
    }

    l.activeAgent = agentID
    l.activeSince = time.Now()
    return nil
}

func (l *TurnLock) ReleaseTurn(agentID AgentID) error {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.activeAgent != agentID {
        return ErrNotTurnHolder
    }

    l.activeAgent = ""
    return nil
}
```

---

## 5. Event-Driven Architectures

### 5.1 CQRS (Command Query Responsibility Segregation)

**Pattern:**
```
Separate Read and Write Models
├── Commands: Mutate state (AddMessage, StartConversation)
├── Queries: Read state (GetMessages, GetAgentStatus)
├── Events: State change notifications
└── Projections: Optimized read models
```

**AgentPipe Application:**
```go
// Commands (write side)
type Command interface {
    Execute(ctx context.Context) error
}

type StartConversationCommand struct {
    ConversationID ConversationID
    Participants   []AgentID
    InitialPrompt  string
}

type AddMessageCommand struct {
    ConversationID ConversationID
    Message        Message
}

// Queries (read side)
type Query interface {
    Execute(ctx context.Context) (interface{}, error)
}

type GetConversationQuery struct {
    ConversationID ConversationID
}

type GetMessagesQuery struct {
    ConversationID ConversationID
    Filter         MessageFilter
}

// Event-driven updates
// Write side publishes events, read side updates projections
type MessageAddedEvent struct {
    ConversationID ConversationID
    Message        Message
    Timestamp      time.Time
}

// Projection for fast queries
type ConversationProjection struct {
    messages      []Message
    participants  map[AgentID]bool
    totalTokens   int
    totalCost     float64
}

func (p *ConversationProjection) Apply(event Event) {
    switch e := event.(type) {
    case *MessageAddedEvent:
        p.messages = append(p.messages, e.Message)
        p.totalTokens += e.Message.Metrics.TotalTokens
        p.totalCost += e.Message.Metrics.Cost
    }
}
```

### 5.2 Event Sourcing

**Pattern:** Store all changes as immutable events

```
Event Store (Append-Only Log)
├── Event 1: ConversationStarted
├── Event 2: MessageCreated
├── Event 3: MessageCreated
├── Event 4: TurnCompleted
└── Event N: ConversationCompleted

Current State = Replay All Events
```

**Benefits:**
- **Complete Audit Trail**: Never lose history
- **Time Travel**: Replay to any point
- **Debugging**: See exactly what happened
- **Derived Views**: Multiple projections from same events

**Implementation:**
```go
// Event store interface
type EventStore interface {
    Append(ctx context.Context, event Event) error
    Load(ctx context.Context, conversationID ConversationID) ([]Event, error)
    LoadSince(ctx context.Context, conversationID ConversationID, timestamp time.Time) ([]Event, error)
}

// Rebuild conversation state from events
func RebuildState(events []Event) *ConversationState {
    state := &ConversationState{}
    for _, event := range events {
        state.Apply(event)
    }
    return state
}

// Event store with snapshots for performance
type SnapshotEventStore struct {
    events    EventStore
    snapshots SnapshotStore
}

func (s *SnapshotEventStore) Load(ctx context.Context, conversationID ConversationID) ([]Event, error) {
    // Load latest snapshot
    snapshot, err := s.snapshots.GetLatest(ctx, conversationID)
    if err != nil {
        // No snapshot, load all events
        return s.events.Load(ctx, conversationID)
    }

    // Load events since snapshot
    events, err := s.events.LoadSince(ctx, conversationID, snapshot.Timestamp)
    if err != nil {
        return nil, err
    }

    return append(snapshot.Events, events...), nil
}
```

### 5.3 Pub/Sub Event Bus

**Pattern:** Decoupled components via event bus

```
        Publisher                  Event Bus                    Subscribers
┌───────────────────┐         ┌───────────────┐         ┌────────────────────┐
│  Orchestrator     │─publish→│   Event       │─notify→ │ Bridge Emitter     │
│                   │         │   Router      │         ├────────────────────┤
│                   │         │               │─notify→ │ Metrics Collector  │
└───────────────────┘         └───────────────┘         ├────────────────────┤
                                                        │ Logger             │
                                                        ├────────────────────┤
                                                        │ TUI Updater        │
                                                        └────────────────────┘
```

**Implementation:**
```go
// Event bus with filtering
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(filter EventFilter, handler EventHandler) (Subscription, error)
    Unsubscribe(subscription Subscription) error
}

type EventFilter func(Event) bool

type EventHandler func(context.Context, Event) error

// Example: Subscribe to only message events
bus.Subscribe(
    func(e Event) bool {
        _, ok := e.(*MessageCreatedEvent)
        return ok
    },
    func(ctx context.Context, e Event) error {
        msg := e.(*MessageCreatedEvent)
        return updateTUI(msg)
    },
)

// Example: Subscribe to all events for specific conversation
bus.Subscribe(
    func(e Event) bool {
        return e.ConversationID() == targetConversationID
    },
    handler,
)
```

**Backpressure Handling:**
```go
// Buffered event bus with backpressure
type BufferedEventBus struct {
    events chan Event
    done   chan struct{}
}

func NewBufferedEventBus(bufferSize int) *BufferedEventBus {
    bus := &BufferedEventBus{
        events: make(chan Event, bufferSize),
        done:   make(chan struct{}),
    }
    go bus.dispatch()
    return bus
}

func (b *BufferedEventBus) Publish(ctx context.Context, event Event) error {
    select {
    case b.events <- event:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(5 * time.Second):
        return ErrEventBusFull  // Backpressure: buffer full
    }
}
```

---

## 6. TUI Group Chat Implementations

### 6.1 Existing Go TUI Chat Apps

**Bubble Tea Patterns:**

```go
// Model-Update-View pattern (Elm architecture)
type Model struct {
    messages    []Message
    agents      []Agent
    viewport    viewport.Model
    input       textarea.Model
    activeAgent AgentID
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        textarea.Blink,
        listenForMessages(),
    )
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case MessageReceived:
        m.messages = append(m.messages, msg.Message)
        m.viewport.SetContent(m.renderMessages())
        return m, nil

    case AgentActivity:
        m.activeAgent = msg.AgentID
        return m, nil
    }
    return m, nil
}

func (m Model) View() string {
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.renderHeader(),
        m.viewport.View(),
        m.renderInput(),
        m.renderStatus(),
    )
}
```

**Multi-Panel Layout:**
```go
// Three-panel layout (agents | conversation | input)
func (m Model) View() string {
    agentsPanel := m.renderAgentsPanel()    // Left: agent list with status
    conversationPanel := m.viewport.View()  // Center: message history
    inputPanel := m.renderInputPanel()      // Bottom: user input

    // Horizontal split: agents | conversation
    mainContent := lipgloss.JoinHorizontal(
        lipgloss.Top,
        agentsPanel,
        conversationPanel,
    )

    // Vertical split: main | input
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.renderTitle(),
        mainContent,
        inputPanel,
        m.renderStatusBar(),
    )
}
```

### 6.2 Real-Time Update Patterns

**Pattern 1: Channel-based updates**
```go
// Background goroutine sends messages to TUI
func listenForMessages() tea.Cmd {
    return func() tea.Msg {
        msg := <-messageChan
        return MessageReceived{Message: msg}
    }
}

// Recursive listening
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case MessageReceived:
        m.messages = append(m.messages, msg.Message)
        return m, listenForMessages()  // Keep listening
    }
    return m, nil
}
```

**Pattern 2: Ticker-based polling**
```go
// Poll for updates every 100ms
func tickEvery(d time.Duration) tea.Cmd {
    return tea.Tick(d, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case TickMsg:
        // Check for new messages
        newMessages := m.orchestrator.GetNewMessages()
        if len(newMessages) > 0 {
            m.messages = append(m.messages, newMessages...)
        }
        return m, tickEvery(100 * time.Millisecond)
    }
    return m, nil
}
```

**Pattern 3: Event bus integration**
```go
// Subscribe to event bus for live updates
func subscribeToEvents(bus EventBus) tea.Cmd {
    ch := make(chan Event)
    bus.Subscribe(allEventsFilter, func(ctx context.Context, e Event) error {
        ch <- e
        return nil
    })

    return func() tea.Msg {
        event := <-ch
        switch e := event.(type) {
        case *MessageCreatedEvent:
            return MessageReceived{Message: e.Message}
        case *AgentActivityChanged:
            return AgentActivityUpdate{Activity: e}
        }
        return nil
    }
}
```

---

## 7. Proven Patterns for AgentPipe

### 7.1 Conversation Flow Architecture

**Recommended Pattern:** Event-sourced orchestration with CQRS

```
┌──────────────────────────────────────────────────────────┐
│                  Conversation Flow                       │
└──────────────────────────────────────────────────────────┘

Commands                   Events                  Projections
────────                   ──────                  ───────────
StartConversation    →   ConversationStarted   →   ConversationState
                                                    ├─ participants
                                                    ├─ messages: []
                                                    └─ status: active

ExecuteTurn          →   TurnStarted           →   (update state)
                    →   MessageCreated        →   ├─ messages: +1
                    →   TurnCompleted         →   └─ currentTurn: +1

AddUserMessage       →   UserMessageAdded      →   (update state)

EndConversation      →   ConversationCompleted →   ├─ status: completed
                                                    └─ summary: ...
```

**Benefits:**
- **Audit Trail**: Every state change is logged
- **Time Travel**: Replay to any point for debugging
- **Projections**: Multiple views (TUI, API, export) from same events
- **Scalability**: Events can be distributed across systems

### 7.2 Turn Coordination Pattern

**Recommended Pattern:** Token-based floor control with timeout

```go
type TurnCoordinator struct {
    conversationID ConversationID
    currentHolder  *AgentID
    heldSince      time.Time
    maxDuration    time.Duration
    waitQueue      []AgentID
    mu             sync.RWMutex
}

func (tc *TurnCoordinator) RequestTurn(agentID AgentID) error {
    tc.mu.Lock()
    defer tc.mu.Unlock()

    // Check if turn is available
    if tc.currentHolder == nil {
        tc.grantTurn(agentID)
        return nil
    }

    // Check if turn expired
    if time.Since(tc.heldSince) > tc.maxDuration {
        tc.releaseTurn(*tc.currentHolder)
        tc.grantTurn(agentID)
        return nil
    }

    // Add to queue
    tc.waitQueue = append(tc.waitQueue, agentID)
    return ErrTurnNotAvailable
}

func (tc *TurnCoordinator) ReleaseTurn(agentID AgentID) error {
    tc.mu.Lock()
    defer tc.mu.Unlock()

    if tc.currentHolder == nil || *tc.currentHolder != agentID {
        return ErrNotTurnHolder
    }

    tc.releaseTurn(agentID)

    // Grant turn to next in queue
    if len(tc.waitQueue) > 0 {
        next := tc.waitQueue[0]
        tc.waitQueue = tc.waitQueue[1:]
        tc.grantTurn(next)
    }

    return nil
}

func (tc *TurnCoordinator) grantTurn(agentID AgentID) {
    tc.currentHolder = &agentID
    tc.heldSince = time.Now()

    // Publish event
    tc.publishEvent(&TurnGranted{
        ConversationID: tc.conversationID,
        AgentID:        agentID,
        Timestamp:      time.Now(),
    })
}
```

### 7.3 Message Streaming Pattern

**Recommended Pattern:** Incremental updates with buffering

```go
// Stream message chunks to TUI in real-time
type StreamingMessage struct {
    MessageID MessageID
    AgentID   AgentID
    chunks    []string
    mu        sync.RWMutex
}

func (sm *StreamingMessage) AppendChunk(chunk string) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.chunks = append(sm.chunks, chunk)
}

func (sm *StreamingMessage) Content() string {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return strings.Join(sm.chunks, "")
}

// Agent streams response
func (a *Agent) StreamMessage(ctx context.Context, req MessageRequest) (<-chan string, error) {
    chunks := make(chan string, 10)  // Buffered to prevent blocking

    go func() {
        defer close(chunks)

        // Generate response in chunks
        response, err := a.cli.StreamResponse(req)
        if err != nil {
            return
        }

        for chunk := range response {
            select {
            case chunks <- chunk:
            case <-ctx.Done():
                return
            }
        }
    }()

    return chunks, nil
}

// Orchestrator handles streaming
func (o *Orchestrator) handleStreamingResponse(agentID AgentID, chunks <-chan string) {
    msg := &StreamingMessage{
        MessageID: generateMessageID(),
        AgentID:   agentID,
    }

    // Publish initial event
    o.eventBus.Publish(ctx, &MessageStarted{
        MessageID: msg.MessageID,
        AgentID:   agentID,
    })

    // Stream chunks
    for chunk := range chunks {
        msg.AppendChunk(chunk)

        // Publish chunk event for TUI update
        o.eventBus.Publish(ctx, &MessageChunkReceived{
            MessageID: msg.MessageID,
            Chunk:     chunk,
        })
    }

    // Publish completion event
    o.eventBus.Publish(ctx, &MessageCompleted{
        MessageID: msg.MessageID,
        Content:   msg.Content(),
    })
}
```

### 7.4 State Management Pattern

**Recommended Pattern:** Snapshot + Event Log

```go
// Hybrid: Snapshots for performance, events for audit trail
type ConversationStateManager struct {
    eventStore    EventStore
    snapshotStore SnapshotStore
    snapshotEvery int  // Snapshot every N events
}

func (sm *ConversationStateManager) SaveEvent(ctx context.Context, event Event) error {
    // Append event to store
    if err := sm.eventStore.Append(ctx, event); err != nil {
        return err
    }

    // Check if snapshot needed
    count, _ := sm.eventStore.Count(ctx, event.ConversationID())
    if count%sm.snapshotEvery == 0 {
        // Create snapshot
        state := sm.rebuildState(ctx, event.ConversationID())
        if err := sm.snapshotStore.Save(ctx, state); err != nil {
            log.Warn("failed to save snapshot", "error", err)
        }
    }

    return nil
}

func (sm *ConversationStateManager) LoadState(ctx context.Context, conversationID ConversationID) (*ConversationState, error) {
    // Try to load latest snapshot
    snapshot, err := sm.snapshotStore.GetLatest(ctx, conversationID)
    if err == nil {
        // Load events since snapshot
        events, err := sm.eventStore.LoadSince(ctx, conversationID, snapshot.Timestamp)
        if err != nil {
            return nil, err
        }

        // Apply events to snapshot
        state := snapshot.State
        for _, event := range events {
            state.Apply(event)
        }
        return state, nil
    }

    // No snapshot, rebuild from all events
    return sm.rebuildState(ctx, conversationID)
}
```

### 7.5 Backpressure Handling Pattern

**Recommended Pattern:** Buffered channels with timeout

```go
// Prevent overwhelming consumers with too many events
type BackpressureEventBus struct {
    buffer     chan Event
    bufferSize int
    timeout    time.Duration
}

func (b *BackpressureEventBus) Publish(ctx context.Context, event Event) error {
    select {
    case b.buffer <- event:
        return nil

    case <-ctx.Done():
        return ctx.Err()

    case <-time.After(b.timeout):
        // Buffer full, drop event or return error
        metrics.RecordDroppedEvent(event.Type())
        return ErrEventBufferFull
    }
}

// Consumer with rate limiting
type RateLimitedConsumer struct {
    limiter *rate.Limiter
    handler EventHandler
}

func (c *RateLimitedConsumer) Consume(ctx context.Context, events <-chan Event) {
    for event := range events {
        // Wait for rate limiter
        if err := c.limiter.Wait(ctx); err != nil {
            return
        }

        // Handle event
        if err := c.handler(ctx, event); err != nil {
            log.Error("event handler failed", "error", err, "event", event)
        }
    }
}
```

---

## 8. Anti-Patterns to Avoid

### 8.1 Synchronous Blocking

**❌ Anti-Pattern:**
```go
// BAD: Blocking the entire conversation on one agent
func (o *Orchestrator) runTurn() {
    for _, agent := range o.agents {
        response, _ := agent.SendMessage(ctx, messages)  // BLOCKS all others
        o.messages = append(o.messages, response)
    }
}
```

**✅ Better Pattern:**
```go
// GOOD: Concurrent execution with coordination
func (o *Orchestrator) runTurn() {
    // Execute agents concurrently
    results := make(chan MessageResult, len(o.agents))
    for _, agent := range o.agents {
        go func(a Agent) {
            response, err := a.SendMessage(ctx, messages)
            results <- MessageResult{Agent: a, Response: response, Error: err}
        }(agent)
    }

    // Collect results with timeout
    timeout := time.After(o.config.TurnTimeout)
    for i := 0; i < len(o.agents); i++ {
        select {
        case result := <-results:
            if result.Error == nil {
                o.handleResponse(result)
            }
        case <-timeout:
            return ErrTurnTimeout
        }
    }
}
```

### 8.2 Unbounded Message History

**❌ Anti-Pattern:**
```go
// BAD: Keeping all messages forever
type Conversation struct {
    messages []Message  // Grows without bound
}

func (c *Conversation) AddMessage(msg Message) {
    c.messages = append(c.messages, msg)  // Memory leak
}
```

**✅ Better Pattern:**
```go
// GOOD: Bounded history with sliding window or summarization
type Conversation struct {
    messages       []Message
    maxMessages    int
    summarizedMsgs []Message  // Compressed old messages
}

func (c *Conversation) AddMessage(msg Message) {
    c.messages = append(c.messages, msg)

    // Check if we need to summarize old messages
    if len(c.messages) > c.maxMessages {
        // Move oldest messages to summarized set
        toSummarize := c.messages[:c.maxMessages/2]
        summary := c.summarize(toSummarize)
        c.summarizedMsgs = append(c.summarizedMsgs, summary)

        // Keep only recent messages
        c.messages = c.messages[c.maxMessages/2:]
    }
}
```

### 8.3 Global Mutable State

**❌ Anti-Pattern:**
```go
// BAD: Global state shared across goroutines
var currentAgent AgentID
var messages []Message

func processMessage(msg Message) {
    messages = append(messages, msg)  // Race condition
}
```

**✅ Better Pattern:**
```go
// GOOD: Encapsulated state with synchronization
type ConversationState struct {
    currentAgent AgentID
    messages     []Message
    mu           sync.RWMutex
}

func (s *ConversationState) AddMessage(msg Message) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.messages = append(s.messages, msg)
}

func (s *ConversationState) GetMessages() []Message {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Return a copy to prevent mutation
    msgs := make([]Message, len(s.messages))
    copy(msgs, s.messages)
    return msgs
}
```

### 8.4 Tight Coupling to TUI

**❌ Anti-Pattern:**
```go
// BAD: Orchestrator directly updates TUI
func (o *Orchestrator) sendMessage(agent Agent) {
    response, _ := agent.SendMessage(ctx, messages)
    o.tui.UpdateMessage(response)  // Tight coupling
}
```

**✅ Better Pattern:**
```go
// GOOD: Event-driven decoupling
func (o *Orchestrator) sendMessage(agent Agent) {
    response, _ := agent.SendMessage(ctx, messages)

    // Publish event (TUI subscribes)
    o.eventBus.Publish(ctx, &MessageCreatedEvent{
        Message: response,
    })
}

// TUI subscribes to events
func (tui *TUI) Init() {
    eventBus.Subscribe(isMessageEvent, func(ctx context.Context, e Event) error {
        msg := e.(*MessageCreatedEvent)
        return tui.UpdateMessage(msg.Message)
    })
}
```

### 8.5 No Error Recovery

**❌ Anti-Pattern:**
```go
// BAD: Single failure kills entire conversation
func (o *Orchestrator) runConversation() error {
    for turn := 0; turn < maxTurns; turn++ {
        agent := o.selectAgent()
        response, err := agent.SendMessage(ctx, messages)
        if err != nil {
            return err  // Entire conversation fails
        }
    }
}
```

**✅ Better Pattern:**
```go
// GOOD: Resilient error handling with retry and skip
func (o *Orchestrator) runConversation() error {
    for turn := 0; turn < maxTurns; turn++ {
        agent := o.selectAgent()

        // Retry with backoff
        response, err := o.sendWithRetry(agent, messages)
        if err != nil {
            // Log error and continue with other agents
            log.Error("agent failed", "agent", agent.ID(), "error", err)
            o.publishError(err)
            continue
        }

        o.handleResponse(response)
    }
    return nil
}
```

---

## 9. Implementation Recommendations

### 9.1 Architecture Priorities for AgentPipe v2

**Priority 1: Event-Driven Core** ⭐⭐⭐⭐⭐
- Decouple components via event bus
- Enable real-time updates for TUI and Bridge
- Support multiple subscribers (TUI, metrics, logging, bridge)
- Foundation for future features (webhooks, plugins)

**Priority 2: State Management** ⭐⭐⭐⭐
- Event sourcing for audit trail
- Snapshots for performance
- Enable time-travel debugging
- Support conversation resume

**Priority 3: Turn Coordination** ⭐⭐⭐⭐
- Token-based floor control
- Timeout recovery
- Agent queue management
- Prevent overlap chaos

**Priority 4: Message Streaming** ⭐⭐⭐
- Real-time chunk updates
- Backpressure handling
- Progress indicators in TUI
- Better UX for slow agents

**Priority 5: Multi-Window TUI** ⭐⭐
- Split conversations (threads)
- Side-by-side agent views
- Parallel conversation tracking
- Advanced visualization

### 9.2 Implementation Phases

**Phase 1: Event Bus Foundation (Week 1-2)**
```
Tasks:
├─ Implement in-memory event bus
├─ Define core event types
├─ Migrate orchestrator to publish events
├─ Update TUI to subscribe to events
└─ Add event-driven logging/metrics
```

**Phase 2: State Management (Week 3-4)**
```
Tasks:
├─ Implement event store (file-based)
├─ Add snapshot system
├─ Conversation state rebuilding
├─ Save/resume functionality
└─ Migration from v1 state format
```

**Phase 3: Turn Coordination (Week 5-6)**
```
Tasks:
├─ Implement floor manager
├─ Add timeout recovery
├─ Agent queue system
├─ Overlap prevention
└─ Turn metrics tracking
```

**Phase 4: Message Streaming (Week 7-8)**
```
Tasks:
├─ Streaming agent interface
├─ Chunked message handling
├─ TUI real-time updates
├─ Backpressure management
└─ Progress indicators
```

**Phase 5: Advanced Features (Week 9+)**
```
Tasks:
├─ Multi-window TUI
├─ Conversation threading
├─ Plugin system integration
├─ Performance optimization
└─ Documentation
```

### 9.3 Code Structure Recommendations

**Recommended Package Layout:**
```
pkg/
├── core/
│   ├── conversation/
│   │   ├── conversation.go       # Core conversation entity
│   │   ├── message.go            # Message types
│   │   ├── state.go              # State management
│   │   └── repository.go         # Persistence interface
│   │
│   ├── orchestration/
│   │   ├── orchestrator.go       # Main orchestrator
│   │   ├── strategy.go           # Selection strategies
│   │   ├── floor.go              # Floor control
│   │   └── coordinator.go        # Turn coordination
│   │
│   └── event/
│       ├── bus.go                # Event bus interface
│       ├── types.go              # Event type definitions
│       └── store.go              # Event store interface
│
├── infrastructure/
│   ├── eventbus/
│   │   ├── memory.go             # In-memory implementation
│   │   ├── nats.go               # NATS implementation (future)
│   │   └── redis.go              # Redis implementation (future)
│   │
│   ├── eventstore/
│   │   ├── file.go               # File-based store
│   │   ├── postgres.go           # PostgreSQL store (future)
│   │   └── snapshot.go           # Snapshot management
│   │
│   └── adapters/
│       └── (existing agent adapters)
│
└── interfaces/
    ├── tui/
    │   ├── tui.go                # Main TUI
    │   ├── events.go             # Event subscription
    │   └── rendering.go          # View rendering
    │
    └── cli/
        └── (existing CLI commands)
```

### 9.4 Testing Strategy

**Unit Tests:**
```go
// Event bus tests
func TestEventBus_PublishSubscribe(t *testing.T) {
    bus := eventbus.NewMemory()
    received := make(chan Event)

    // Subscribe
    bus.Subscribe(allEventsFilter, func(ctx context.Context, e Event) error {
        received <- e
        return nil
    })

    // Publish
    event := &MessageCreatedEvent{Content: "test"}
    bus.Publish(context.Background(), event)

    // Verify
    select {
    case e := <-received:
        assert.Equal(t, event, e)
    case <-time.After(1 * time.Second):
        t.Fatal("event not received")
    }
}

// Floor control tests
func TestFloorManager_TurnCoordination(t *testing.T) {
    fm := orchestration.NewFloorManager()

    // Agent 1 acquires floor
    err := fm.AcquireFloor("agent-1", 5*time.Second)
    assert.NoError(t, err)

    // Agent 2 tries to acquire (should queue)
    err = fm.AcquireFloor("agent-2", 5*time.Second)
    assert.ErrorIs(t, err, ErrFloorBusy)

    // Agent 1 releases
    err = fm.ReleaseFloor("agent-1")
    assert.NoError(t, err)

    // Agent 2 should now have floor
    assert.Equal(t, "agent-2", fm.CurrentHolder())
}
```

**Integration Tests:**
```go
// End-to-end conversation flow
func TestConversation_EventDrivenFlow(t *testing.T) {
    // Setup
    bus := eventbus.NewMemory()
    store := eventstore.NewFileStore(t.TempDir())
    orch := orchestration.NewOrchestrator(bus, store)

    // Track events
    var events []Event
    bus.Subscribe(allEventsFilter, func(ctx context.Context, e Event) error {
        events = append(events, e)
        return nil
    })

    // Run conversation
    orch.Start(context.Background())

    // Verify event sequence
    assert.IsType(t, &ConversationStartedEvent{}, events[0])
    assert.IsType(t, &TurnStartedEvent{}, events[1])
    assert.IsType(t, &MessageCreatedEvent{}, events[2])
    // ... etc
}
```

### 9.5 Performance Considerations

**Metrics to Track:**
- Event bus throughput (events/second)
- Event latency (time from publish to consume)
- State rebuild time (from events)
- Snapshot creation time
- Memory usage (event buffer, message history)

**Optimization Strategies:**
1. **Buffered Channels**: Prevent blocking on event publish
2. **Event Batching**: Group related events for efficiency
3. **Snapshot Frequency**: Balance performance vs. storage
4. **Message Pruning**: Limit in-memory message history
5. **Lazy Loading**: Load conversation state on-demand

**Benchmarks:**
```go
func BenchmarkEventBus_Publish(b *testing.B) {
    bus := eventbus.NewMemory()
    event := &MessageCreatedEvent{Content: "test"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bus.Publish(context.Background(), event)
    }
}

func BenchmarkStateManager_Rebuild(b *testing.B) {
    sm := statemanager.New(eventstore, snapshotstore)
    events := generateTestEvents(1000)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sm.RebuildFromEvents(events)
    }
}
```

---

## Conclusion

This research identifies proven patterns from production systems that can be applied to AgentPipe v2:

**Top Recommendations:**
1. **Event-Driven Architecture**: Decouple components, enable real-time updates
2. **Turn Coordination**: Token-based floor control prevents chaos
3. **State Management**: Event sourcing + snapshots for audit and performance
4. **Message Streaming**: Real-time chunk updates for better UX
5. **Backpressure Handling**: Prevent overwhelming consumers

**Key Takeaways:**
- Multi-agent systems require **explicit coordination** to prevent chaos
- **Event-driven patterns** scale better than synchronous blocking
- **State management** is critical for conversation resume and debugging
- **Real-time updates** improve user experience significantly
- **Error recovery** must be built-in from the start

**Next Steps:**
1. Validate event bus design with proof-of-concept
2. Design concrete event types for AgentPipe domain
3. Implement floor control for turn coordination
4. Add streaming support to agent interface
5. Build TUI event subscription layer

---

**References:**
- LangGraph Documentation: https://langchain-ai.github.io/langgraph/
- AutoGen Paper: https://arxiv.org/abs/2308.08155
- CrewAI GitHub: https://github.com/joaomdmoura/crewAI
- XMPP XEP-0045: Multi-User Chat
- Matrix Specification: https://spec.matrix.org/
- Bubble Tea Examples: https://github.com/charmbracelet/bubbletea
- Event Sourcing Pattern: Martin Fowler's blog
- CQRS Pattern: Microsoft Azure Architecture Center

**Document Version:** 1.0
**Last Updated:** 2026-01-18
**Author:** Research & Analysis Agent
