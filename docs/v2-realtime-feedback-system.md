# AgentPipe v2: Real-Time Feedback and Event System

**Version:** 2.0.0-alpha
**Date:** 2026-01-18
**Status:** Design Proposal
**Authors:** System Architecture Team

---

## Executive Summary

This document defines the real-time feedback and event system for AgentPipe v2, enabling users to observe agent conversations as they unfold with full visibility into agent states, message streaming, and conversation dynamics. The system is designed with performance, smooth UX, and rich observability as primary goals.

### Key Design Goals

- **Complete Visibility**: Users see everything happening in real-time (like watching a group chat)
- **Non-Blocking**: Event emission never blocks agent operations
- **Performant**: Smart buffering and throttling for smooth rendering
- **Extensible**: Event types easily extended for new features
- **Replayable**: Full conversation playback with timing information

---

## Table of Contents

1. [Event Type Definitions](#1-event-type-definitions)
2. [Event Bus Architecture](#2-event-bus-architecture)
3. [TUI Real-Time Rendering Strategy](#3-tui-real-time-rendering-strategy)
4. [Performance Considerations](#4-performance-considerations)
5. [Historical Playback](#5-historical-playback)
6. [Export Formats](#6-export-formats)
7. [Example Event Sequences](#7-example-event-sequences)
8. [Implementation Approach](#8-implementation-approach)
9. [Testing Strategy](#9-testing-strategy)
10. [Migration from v1](#10-migration-from-v1)

---

## 1. Event Type Definitions

### 1.1 Event Hierarchy

All events implement a base `Event` interface:

```go
// Base event interface
type Event interface {
    ID() string              // Unique event ID
    Type() EventType         // Event type enum
    Timestamp() time.Time    // When event occurred
    ConversationID() string  // Associated conversation
    Metadata() map[string]any // Optional metadata
}

// Event type enumeration
type EventType string

const (
    // Agent lifecycle events
    EventAgentRegistered   EventType = "agent.registered"
    EventAgentInitialized  EventType = "agent.initialized"
    EventAgentHealthCheck  EventType = "agent.health_check"
    EventAgentError        EventType = "agent.error"
    EventAgentTerminated   EventType = "agent.terminated"

    // Agent state events
    EventAgentIdle         EventType = "agent.idle"
    EventAgentThinking     EventType = "agent.thinking"
    EventAgentTyping       EventType = "agent.typing"
    EventAgentResponding   EventType = "agent.responding"
    EventAgentWaiting      EventType = "agent.waiting"
    EventAgentDone         EventType = "agent.done"

    // Message events
    EventMessageStarted    EventType = "message.started"
    EventMessageChunk      EventType = "message.chunk"
    EventMessageComplete   EventType = "message.complete"
    EventMessageError      EventType = "message.error"
    EventMessageRetry      EventType = "message.retry"

    // Turn events
    EventTurnStarted       EventType = "turn.started"
    EventTurnAgentSelected EventType = "turn.agent_selected"
    EventTurnComplete      EventType = "turn.complete"
    EventTurnSkipped       EventType = "turn.skipped"

    // Conversation events
    EventConversationStarted   EventType = "conversation.started"
    EventConversationPaused    EventType = "conversation.paused"
    EventConversationResumed   EventType = "conversation.resumed"
    EventConversationComplete  EventType = "conversation.complete"
    EventConversationError     EventType = "conversation.error"

    // Reaction events (agent-to-agent)
    EventAgentReaction     EventType = "agent.reaction"
    EventAgentAcknowledge  EventType = "agent.acknowledge"
    EventAgentInterrupt    EventType = "agent.interrupt"

    // Progress events
    EventProgressStarted   EventType = "progress.started"
    EventProgressUpdate    EventType = "progress.update"
    EventProgressComplete  EventType = "progress.complete"

    // Artifact events
    EventArtifactCreated   EventType = "artifact.created"
    EventArtifactUpdated   EventType = "artifact.updated"
    EventArtifactShared    EventType = "artifact.shared"

    // System events
    EventSystemMetrics     EventType = "system.metrics"
    EventSystemWarning     EventType = "system.warning"
)
```

### 1.2 Core Event Structures

#### Agent State Events

```go
// AgentStateEvent represents agent status changes
type AgentStateEvent struct {
    BaseEvent
    AgentID     string            `json:"agent_id"`
    AgentName   string            `json:"agent_name"`
    AgentType   string            `json:"agent_type"`
    PrevState   AgentState        `json:"prev_state"`
    NewState    AgentState        `json:"new_state"`
    Reason      string            `json:"reason,omitempty"`
    Progress    *ProgressInfo     `json:"progress,omitempty"` // For long operations
}

type AgentState string

const (
    StateIdle       AgentState = "idle"
    StateThinking   AgentState = "thinking"   // Processing prompt
    StateTyping     AgentState = "typing"     // Generating response
    StateResponding AgentState = "responding" // Sending response
    StateWaiting    AgentState = "waiting"    // Waiting for turn
    StateDone       AgentState = "done"       // Completed turn
    StateError      AgentState = "error"      // Error occurred
)

type ProgressInfo struct {
    Current     int     `json:"current"`
    Total       int     `json:"total"`
    Percentage  float64 `json:"percentage"`
    Message     string  `json:"message,omitempty"`
    EstimatedMS int64   `json:"estimated_ms,omitempty"` // Est. time remaining
}
```

#### Message Events

```go
// MessageStartedEvent fires when agent begins generating a message
type MessageStartedEvent struct {
    BaseEvent
    MessageID   string    `json:"message_id"`
    AgentID     string    `json:"agent_id"`
    AgentName   string    `json:"agent_name"`
    TurnNumber  int       `json:"turn_number"`
    IsStreaming bool      `json:"is_streaming"`
}

// MessageChunkEvent fires for each chunk of streaming content
type MessageChunkEvent struct {
    BaseEvent
    MessageID   string `json:"message_id"`
    AgentID     string `json:"agent_id"`
    ChunkIndex  int    `json:"chunk_index"`
    Content     string `json:"content"`
    Delta       string `json:"delta"` // New content only (for efficiency)
    ChunkType   ChunkType `json:"chunk_type"`
}

type ChunkType string

const (
    ChunkText      ChunkType = "text"       // Regular text content
    ChunkThinking  ChunkType = "thinking"   // Thinking/reasoning text
    ChunkCode      ChunkType = "code"       // Code block
    ChunkToolUse   ChunkType = "tool_use"   // Tool invocation
    ChunkToolResult ChunkType = "tool_result" // Tool output
)

// MessageCompleteEvent fires when message generation finishes
type MessageCompleteEvent struct {
    BaseEvent
    MessageID       string        `json:"message_id"`
    AgentID         string        `json:"agent_id"`
    Content         string        `json:"content"`
    TokensInput     int           `json:"tokens_input"`
    TokensOutput    int           `json:"tokens_output"`
    TokensTotal     int           `json:"tokens_total"`
    Cost            float64       `json:"cost"`
    DurationMS      int64         `json:"duration_ms"`
    Model           string        `json:"model"`
    FinishReason    string        `json:"finish_reason"`
    Artifacts       []ArtifactRef `json:"artifacts,omitempty"`
}
```

#### Turn Events

```go
// TurnStartedEvent fires at the beginning of a conversation turn
type TurnStartedEvent struct {
    BaseEvent
    TurnNumber      int      `json:"turn_number"`
    Mode            string   `json:"mode"` // round-robin, reactive, free-form
    EligibleAgents  []string `json:"eligible_agents"`
    MaxDurationMS   int64    `json:"max_duration_ms,omitempty"`
}

// TurnAgentSelectedEvent fires when next agent is chosen
type TurnAgentSelectedEvent struct {
    BaseEvent
    TurnNumber      int    `json:"turn_number"`
    AgentID         string `json:"agent_id"`
    AgentName       string `json:"agent_name"`
    SelectionReason string `json:"selection_reason"`
    IsAutoSelected  bool   `json:"is_auto_selected"`
}

// TurnCompleteEvent fires when a turn finishes
type TurnCompleteEvent struct {
    BaseEvent
    TurnNumber      int             `json:"turn_number"`
    MessagesCount   int             `json:"messages_count"`
    TotalTokens     int             `json:"total_tokens"`
    TotalCost       float64         `json:"total_cost"`
    DurationMS      int64           `json:"duration_ms"`
    Participants    []AgentSummary  `json:"participants"`
}
```

#### Reaction Events

```go
// AgentReactionEvent fires when an agent reacts to another's message
type AgentReactionEvent struct {
    BaseEvent
    ReactingAgentID string       `json:"reacting_agent_id"`
    TargetMessageID string       `json:"target_message_id"`
    TargetAgentID   string       `json:"target_agent_id"`
    ReactionType    ReactionType `json:"reaction_type"`
    Comment         string       `json:"comment,omitempty"`
}

type ReactionType string

const (
    ReactionAgree     ReactionType = "agree"
    ReactionDisagree  ReactionType = "disagree"
    ReactionQuestion  ReactionType = "question"
    ReactionClarify   ReactionType = "clarify"
    ReactionApprove   ReactionType = "approve"
    ReactionObject    ReactionType = "object"
)

// AgentAcknowledgeEvent fires when agent acknowledges receipt
type AgentAcknowledgeEvent struct {
    BaseEvent
    AgentID         string `json:"agent_id"`
    MessageID       string `json:"message_id"`
    AckType         string `json:"ack_type"` // read, understood, processing
}
```

### 1.3 Event Metadata

All events can carry optional metadata for extensibility:

```go
type EventMetadata map[string]any

// Common metadata keys
const (
    MetaKeyUserID      = "user_id"
    MetaKeySessionID   = "session_id"
    MetaKeyEnvironment = "environment"
    MetaKeyVersion     = "version"
    MetaKeyTags        = "tags"
    MetaKeyCorrelation = "correlation_id"
    MetaKeyParentEvent = "parent_event_id"
    MetaKeySource      = "source_component"
)
```

---

## 2. Event Bus Architecture

### 2.1 Event Bus Design

```go
// EventBus is the central event distribution hub
type EventBus interface {
    // Publish emits an event to all subscribers
    Publish(ctx context.Context, event Event) error

    // Subscribe registers a handler for specific event types
    Subscribe(types []EventType, handler EventHandler) Subscription

    // SubscribeAll registers a handler for all events
    SubscribeAll(handler EventHandler) Subscription

    // SubscribePattern registers a handler using pattern matching
    SubscribePattern(pattern string, handler EventHandler) Subscription

    // Close shuts down the event bus gracefully
    Close(ctx context.Context) error

    // Metrics returns event bus statistics
    Metrics() EventBusMetrics
}

// EventHandler processes events
type EventHandler func(ctx context.Context, event Event) error

// Subscription allows unsubscribing
type Subscription interface {
    Unsubscribe() error
    ID() string
    EventTypes() []EventType
}

// EventBusMetrics provides observability
type EventBusMetrics struct {
    EventsPublished     int64
    EventsDropped       int64
    SubscriberCount     int
    AvgLatencyMS        float64
    BufferUtilization   float64
    ErrorRate           float64
}
```

### 2.2 Implementation: Buffered Channel-Based Bus

```go
// bufferSize determines channel capacity
type BufferedEventBus struct {
    subscribers   map[string]*subscriber
    subscribersMu sync.RWMutex

    eventChan     chan Event
    bufferSize    int

    ctx           context.Context
    cancel        context.CancelFunc
    wg            sync.WaitGroup

    metrics       *eventBusMetrics
    logger        Logger

    // Configuration
    dropOnFull    bool  // Drop events if buffer full
    blockOnFull   bool  // Block publisher if buffer full
    warnThreshold float64 // Warn when buffer > this %
}

type subscriber struct {
    id          string
    types       []EventType
    pattern     string // For pattern matching
    handler     EventHandler
    deliverChan chan Event
    bufferSize  int
}

// Key methods
func (b *BufferedEventBus) Publish(ctx context.Context, event Event) error {
    select {
    case b.eventChan <- event:
        atomic.AddInt64(&b.metrics.eventsPublished, 1)
        return nil
    case <-ctx.Done():
        return ctx.Err()
    default:
        if b.dropOnFull {
            atomic.AddInt64(&b.metrics.eventsDropped, 1)
            b.logger.Warn("Event dropped due to full buffer",
                "type", event.Type(),
                "buffer_util", b.BufferUtilization())
            return nil
        }
        // Block until space available
        select {
        case b.eventChan <- event:
            return nil
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

func (b *BufferedEventBus) dispatch() {
    defer b.wg.Done()

    for {
        select {
        case event := <-b.eventChan:
            b.deliverToSubscribers(event)
        case <-b.ctx.Done():
            return
        }
    }
}

func (b *BufferedEventBus) deliverToSubscribers(event Event) {
    b.subscribersMu.RLock()
    defer b.subscribersMu.RUnlock()

    start := time.Now()

    for _, sub := range b.subscribers {
        if sub.matches(event) {
            // Non-blocking delivery to subscriber
            select {
            case sub.deliverChan <- event:
                // Delivered successfully
            default:
                // Subscriber buffer full, log warning
                b.logger.Warn("Subscriber buffer full, dropping event",
                    "subscriber", sub.id,
                    "event_type", event.Type())
                atomic.AddInt64(&b.metrics.eventsDropped, 1)
            }
        }
    }

    // Update latency metrics
    latency := time.Since(start).Milliseconds()
    b.metrics.recordLatency(latency)
}
```

### 2.3 Subscriber Patterns

**Type-Based Subscription**
```go
// Subscribe to specific event types
sub := bus.Subscribe([]EventType{
    EventAgentThinking,
    EventAgentTyping,
    EventAgentResponding,
}, func(ctx context.Context, event Event) error {
    // Handle agent state changes
    return nil
})
defer sub.Unsubscribe()
```

**Pattern-Based Subscription**
```go
// Subscribe to all message events
sub := bus.SubscribePattern("message.*", func(ctx context.Context, event Event) error {
    // Handle all message events
    return nil
})

// Subscribe to all agent events
sub := bus.SubscribePattern("agent.*", handleAgentEvents)
```

**Fan-Out Pattern**
```go
// Multiple subscribers for same event type
bus.Subscribe([]EventType{EventMessageChunk}, renderToTUI)
bus.Subscribe([]EventType{EventMessageChunk}, writeToLog)
bus.Subscribe([]EventType{EventMessageChunk}, sendToWebSocket)
```

### 2.4 Event Bus Configuration

```go
type EventBusConfig struct {
    BufferSize      int           // Main event buffer size (default: 1000)
    SubscriberBuffer int          // Per-subscriber buffer (default: 100)
    DropOnFull      bool          // Drop events if buffer full (default: false)
    WarnThreshold   float64       // Warn threshold 0-1 (default: 0.8)
    MaxSubscribers  int           // Max concurrent subscribers (default: 100)
    EnableMetrics   bool          // Collect metrics (default: true)
    MetricsInterval time.Duration // Metrics collection interval (default: 1s)
}

// Recommended configurations
var (
    // High-throughput config (for many events)
    HighThroughputConfig = EventBusConfig{
        BufferSize:       10000,
        SubscriberBuffer: 1000,
        DropOnFull:       true, // Prevent blocking
        WarnThreshold:    0.9,
    }

    // Reliable config (no event loss)
    ReliableConfig = EventBusConfig{
        BufferSize:       5000,
        SubscriberBuffer: 500,
        DropOnFull:       false, // Block if necessary
        WarnThreshold:    0.7,
    }

    // Lightweight config (for testing/dev)
    LightweightConfig = EventBusConfig{
        BufferSize:       100,
        SubscriberBuffer: 50,
        DropOnFull:       true,
        EnableMetrics:    false,
    }
)
```

---

## 3. TUI Real-Time Rendering Strategy

### 3.1 Component Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        TUI LAYOUT                            │
├──────────────────────────────────────────────────────────────┤
│  ┌────────────────┐  ┌─────────────────────────────────────┐ │
│  │  Agent Panel   │  │     Conversation Panel              │ │
│  │                │  │                                     │ │
│  │  Agent 1 [🟢]  │  │  [Turn 1]                          │ │
│  │    Idle        │  │  Agent1: Hello there!              │ │
│  │                │  │  ├─ 150 tokens | 0.5s | $0.0002    │ │
│  │  Agent 2 [🟡]  │  │                                     │ │
│  │    Thinking... │  │  [Turn 2]                          │ │
│  │    ▓▓▓▓▓░░░░   │  │  Agent2: ▓ Let me analyze...       │ │
│  │                │  │  ├─ streaming... ⏱️ 1.2s            │ │
│  │  Agent 3 [⚪]  │  │                                     │ │
│  │    Waiting     │  │  [Next: Agent 3]                   │ │
│  │                │  │                                     │ │
│  └────────────────┘  └─────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Input Panel                                            │ │
│  │  > Type message or press 'u' to participate...         │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│  Status: Turn 2/10 | Tokens: 1,523 | Cost: $0.0024 | ⏱️ 12s  │
└──────────────────────────────────────────────────────────────┘
```

### 3.2 Event-Driven Rendering

```go
// TUIRenderer subscribes to events and updates UI
type TUIRenderer struct {
    app           *tview.Application
    agentPanel    *AgentListView
    chatPanel     *ConversationView
    inputPanel    *InputView
    statusBar     *StatusBar

    eventBus      EventBus
    subscriptions []Subscription

    updateChan    chan UIUpdate
    throttler     *EventThrottler

    state         *UIState
    stateMu       sync.RWMutex
}

// UIUpdate represents a UI state change
type UIUpdate struct {
    Component UpdateComponent
    Action    UpdateAction
    Data      any
}

type UpdateComponent string
const (
    ComponentAgentList UpdateComponent = "agent_list"
    ComponentChat      UpdateComponent = "chat"
    ComponentInput     UpdateComponent = "input"
    ComponentStatus    UpdateComponent = "status"
)

type UpdateAction string
const (
    ActionAdd      UpdateAction = "add"
    ActionUpdate   UpdateAction = "update"
    ActionRemove   UpdateAction = "remove"
    ActionClear    UpdateAction = "clear"
    ActionHighlight UpdateAction = "highlight"
)

func (r *TUIRenderer) Start(ctx context.Context) error {
    // Subscribe to relevant events
    r.subscriptions = []Subscription{
        r.eventBus.SubscribePattern("agent.*", r.handleAgentEvent),
        r.eventBus.SubscribePattern("message.*", r.handleMessageEvent),
        r.eventBus.SubscribePattern("turn.*", r.handleTurnEvent),
        r.eventBus.Subscribe([]EventType{
            EventConversationStarted,
            EventConversationComplete,
        }, r.handleConversationEvent),
    }

    // Start update loop
    go r.updateLoop(ctx)

    // Run TUI app
    return r.app.Run()
}

func (r *TUIRenderer) handleAgentEvent(ctx context.Context, event Event) error {
    switch e := event.(type) {
    case *AgentStateEvent:
        r.updateChan <- UIUpdate{
            Component: ComponentAgentList,
            Action:    ActionUpdate,
            Data: AgentStatusUpdate{
                AgentID:   e.AgentID,
                State:     e.NewState,
                Progress:  e.Progress,
            },
        }
    }
    return nil
}

func (r *TUIRenderer) handleMessageEvent(ctx context.Context, event Event) error {
    switch e := event.(type) {
    case *MessageStartedEvent:
        r.updateChan <- UIUpdate{
            Component: ComponentChat,
            Action:    ActionAdd,
            Data: MessagePlaceholder{
                MessageID: e.MessageID,
                AgentName: e.AgentName,
            },
        }

    case *MessageChunkEvent:
        r.updateChan <- UIUpdate{
            Component: ComponentChat,
            Action:    ActionUpdate,
            Data: MessageContentUpdate{
                MessageID: e.MessageID,
                Delta:     e.Delta, // Only new content
            },
        }

    case *MessageCompleteEvent:
        r.updateChan <- UIUpdate{
            Component: ComponentChat,
            Action:    ActionUpdate,
            Data: MessageFinalUpdate{
                MessageID:    e.MessageID,
                Metrics: MessageMetrics{
                    Tokens:   e.TokensTotal,
                    Cost:     e.Cost,
                    Duration: e.DurationMS,
                },
            },
        }
    }
    return nil
}
```

### 3.3 Progressive Rendering Strategies

**Character-by-Character Streaming**
```go
// For smooth typing effect
type CharacterStreamRenderer struct {
    buffer      *strings.Builder
    renderRate  time.Duration // e.g., 50ms per char
    lastRender  time.Time
    pendingChars []rune
    mu          sync.Mutex
}

func (r *CharacterStreamRenderer) AddChunk(chunk string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.pendingChars = append(r.pendingChars, []rune(chunk)...)
}

func (r *CharacterStreamRenderer) Render() string {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    if now.Sub(r.lastRender) < r.renderRate {
        return r.buffer.String()
    }

    // Render next character
    if len(r.pendingChars) > 0 {
        r.buffer.WriteRune(r.pendingChars[0])
        r.pendingChars = r.pendingChars[1:]
        r.lastRender = now
    }

    return r.buffer.String()
}
```

**Sentence-by-Sentence Streaming**
```go
// For better readability
type SentenceStreamRenderer struct {
    buffer         *strings.Builder
    pendingContent string
    sentenceRegex  *regexp.Regexp
    mu             sync.Mutex
}

func (r *SentenceStreamRenderer) AddChunk(chunk string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.pendingContent += chunk

    // Check for complete sentences
    sentences := r.sentenceRegex.FindAllString(r.pendingContent, -1)
    if len(sentences) > 0 {
        for _, sentence := range sentences {
            r.buffer.WriteString(sentence)
            r.pendingContent = strings.TrimPrefix(r.pendingContent, sentence)
        }
    }
}
```

**Word-by-Word Streaming** (Recommended)
```go
// Balance between smoothness and readability
type WordStreamRenderer struct {
    buffer      *strings.Builder
    pendingWords []string
    renderRate  time.Duration // e.g., 100ms per word
    lastRender  time.Time
    mu          sync.Mutex
}

func (r *WordStreamRenderer) AddChunk(chunk string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    words := strings.Fields(chunk)
    r.pendingWords = append(r.pendingWords, words...)
}

func (r *WordStreamRenderer) Render() string {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    if now.Sub(r.lastRender) < r.renderRate || len(r.pendingWords) == 0 {
        return r.buffer.String()
    }

    // Render next word
    r.buffer.WriteString(r.pendingWords[0])
    r.buffer.WriteString(" ")
    r.pendingWords = r.pendingWords[1:]
    r.lastRender = now

    return r.buffer.String()
}
```

### 3.4 Agent Status Indicators

```go
// Visual representation of agent states
var AgentStateIndicators = map[AgentState]string{
    StateIdle:       "🟢", // Green circle
    StateThinking:   "🟡", // Yellow circle
    StateTyping:     "🔵", // Blue circle
    StateResponding: "🟣", // Purple circle
    StateWaiting:    "⚪", // White circle
    StateDone:       "✅", // Checkmark
    StateError:      "🔴", // Red circle
}

// Progress bar for long operations
func RenderProgressBar(progress *ProgressInfo, width int) string {
    if progress == nil {
        return ""
    }

    filled := int(float64(width) * (progress.Percentage / 100.0))
    empty := width - filled

    bar := strings.Repeat("▓", filled) + strings.Repeat("░", empty)

    if progress.Message != "" {
        return fmt.Sprintf("%s %s (%.0f%%)", bar, progress.Message, progress.Percentage)
    }

    return fmt.Sprintf("%s %.0f%%", bar, progress.Percentage)
}
```

---

## 4. Performance Considerations

### 4.1 Event Throttling

```go
// EventThrottler prevents UI from being overwhelmed
type EventThrottler struct {
    minInterval   time.Duration  // Min time between events
    lastEmit      map[EventType]time.Time
    mu            sync.RWMutex

    // Aggregation settings
    aggregateTypes map[EventType]bool
    aggregateWindow time.Duration
    aggregated     map[EventType][]Event
}

func (t *EventThrottler) ShouldEmit(event Event) bool {
    t.mu.RLock()
    defer t.mu.RUnlock()

    eventType := event.Type()
    lastEmit, exists := t.lastEmit[eventType]

    if !exists || time.Since(lastEmit) >= t.minInterval {
        return true
    }

    return false
}

func (t *EventThrottler) RecordEmit(event Event) {
    t.mu.Lock()
    defer t.mu.Unlock()

    t.lastEmit[event.Type()] = time.Now()
}

// For high-frequency events like MessageChunk
func (t *EventThrottler) Aggregate(event Event) ([]Event, bool) {
    t.mu.Lock()
    defer t.mu.Unlock()

    eventType := event.Type()
    if !t.aggregateTypes[eventType] {
        return nil, false
    }

    t.aggregated[eventType] = append(t.aggregated[eventType], event)

    // Check if window expired
    if time.Since(t.lastEmit[eventType]) >= t.aggregateWindow {
        events := t.aggregated[eventType]
        t.aggregated[eventType] = nil
        t.lastEmit[eventType] = time.Now()
        return events, true
    }

    return nil, false
}
```

### 4.2 Buffering Strategy

```go
// MessageBuffer aggregates chunks for efficient rendering
type MessageBuffer struct {
    messageID   string
    chunks      []string
    chunkCount  int
    lastFlush   time.Time
    flushInterval time.Duration
    maxChunks   int
    mu          sync.Mutex
}

func (b *MessageBuffer) AddChunk(chunk string) (string, bool) {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.chunks = append(b.chunks, chunk)
    b.chunkCount++

    // Flush if conditions met
    shouldFlush := b.chunkCount >= b.maxChunks ||
                   time.Since(b.lastFlush) >= b.flushInterval

    if shouldFlush {
        content := strings.Join(b.chunks, "")
        b.chunks = nil
        b.chunkCount = 0
        b.lastFlush = time.Now()
        return content, true
    }

    return "", false
}
```

### 4.3 Rendering Optimization

```go
// RenderQueue batches UI updates
type RenderQueue struct {
    updates     []UIUpdate
    maxBatch    int
    batchWindow time.Duration
    lastRender  time.Time
    mu          sync.Mutex

    renderFunc  func([]UIUpdate) error
}

func (q *RenderQueue) Enqueue(update UIUpdate) error {
    q.mu.Lock()
    defer q.mu.Unlock()

    q.updates = append(q.updates, update)

    // Flush if needed
    if len(q.updates) >= q.maxBatch ||
       time.Since(q.lastRender) >= q.batchWindow {
        return q.flush()
    }

    return nil
}

func (q *RenderQueue) flush() error {
    if len(q.updates) == 0 {
        return nil
    }

    updates := q.updates
    q.updates = nil
    q.lastRender = time.Now()

    return q.renderFunc(updates)
}

// Periodic flush
func (q *RenderQueue) StartPeriodicFlush(ctx context.Context) {
    ticker := time.NewTicker(q.batchWindow)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            q.mu.Lock()
            q.flush()
            q.mu.Unlock()
        case <-ctx.Done():
            return
        }
    }
}
```

### 4.4 Memory Management

```go
// EventHistory with size limits
type EventHistory struct {
    events     []Event
    maxSize    int
    maxAge     time.Duration
    mu         sync.RWMutex

    // Optional persistence
    persistence EventPersistence
}

func (h *EventHistory) Add(event Event) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.events = append(h.events, event)

    // Prune old events
    h.pruneOldEvents()

    // Enforce size limit
    if len(h.events) > h.maxSize {
        // Archive old events to persistence
        if h.persistence != nil {
            archived := h.events[:len(h.events)-h.maxSize]
            h.persistence.Archive(archived)
        }
        h.events = h.events[len(h.events)-h.maxSize:]
    }
}

func (h *EventHistory) pruneOldEvents() {
    if h.maxAge == 0 {
        return
    }

    cutoff := time.Now().Add(-h.maxAge)

    for i, event := range h.events {
        if event.Timestamp().After(cutoff) {
            h.events = h.events[i:]
            return
        }
    }
}
```

### 4.5 Performance Metrics

```go
// Key performance indicators
type PerformanceMetrics struct {
    // Event bus metrics
    EventsPerSecond     float64
    AvgEventLatencyMS   float64
    BufferUtilization   float64

    // Rendering metrics
    FramesPerSecond     float64
    AvgRenderTimeMS     float64
    DroppedFrames       int64

    // Memory metrics
    EventHistorySize    int
    TotalMemoryMB       float64

    // User experience metrics
    InputLagMS          float64
    TimeToFirstByte     float64
}

// Recommended thresholds
const (
    TargetFPS           = 30.0  // 30 FPS for smooth rendering
    MaxRenderTimeMS     = 33.0  // 33ms per frame (30 FPS)
    MaxInputLagMS       = 100.0 // 100ms input response time
    MaxBufferUtil       = 0.8   // 80% buffer utilization
)
```

---

## 5. Historical Playback

### 5.1 Event Recording

```go
// EventRecorder captures all events for playback
type EventRecorder struct {
    events      []TimedEvent
    recording   bool
    startTime   time.Time
    mu          sync.RWMutex

    // Output configuration
    outputFormat RecordingFormat
    compression  bool
}

type TimedEvent struct {
    Event        Event         `json:"event"`
    RelativeTime time.Duration `json:"relative_time"` // Time since conversation start
    WallTime     time.Time     `json:"wall_time"`
}

type RecordingFormat string

const (
    FormatJSON     RecordingFormat = "json"
    FormatJSONL    RecordingFormat = "jsonl"    // JSON Lines
    FormatProtobuf RecordingFormat = "protobuf"
    FormatParquet  RecordingFormat = "parquet"
)

func (r *EventRecorder) Start() {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.recording = true
    r.startTime = time.Now()
    r.events = nil
}

func (r *EventRecorder) Record(event Event) {
    r.mu.RLock()
    if !r.recording {
        r.mu.RUnlock()
        return
    }
    r.mu.RUnlock()

    r.mu.Lock()
    defer r.mu.Unlock()

    r.events = append(r.events, TimedEvent{
        Event:        event,
        RelativeTime: time.Since(r.startTime),
        WallTime:     time.Now(),
    })
}

func (r *EventRecorder) Stop() Recording {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.recording = false

    return Recording{
        Events:    r.events,
        Duration:  time.Since(r.startTime),
        StartTime: r.startTime,
        EndTime:   time.Now(),
    }
}
```

### 5.2 Playback Engine

```go
// PlaybackEngine replays recorded events
type PlaybackEngine struct {
    recording    Recording
    eventBus     EventBus

    currentIndex int
    currentTime  time.Duration

    speed        float64  // Playback speed multiplier
    paused       bool

    mu           sync.RWMutex
}

func (p *PlaybackEngine) Play(ctx context.Context) error {
    p.mu.Lock()
    p.paused = false
    p.mu.Unlock()

    startTime := time.Now()

    for p.currentIndex < len(p.recording.Events) {
        p.mu.RLock()
        if p.paused {
            p.mu.RUnlock()
            time.Sleep(100 * time.Millisecond)
            continue
        }
        speed := p.speed
        p.mu.RUnlock()

        timedEvent := p.recording.Events[p.currentIndex]

        // Calculate when to emit next event
        targetTime := time.Duration(float64(timedEvent.RelativeTime) / speed)
        elapsed := time.Since(startTime)

        if targetTime > elapsed {
            select {
            case <-time.After(targetTime - elapsed):
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        // Emit event
        if err := p.eventBus.Publish(ctx, timedEvent.Event); err != nil {
            return fmt.Errorf("failed to emit event: %w", err)
        }

        p.mu.Lock()
        p.currentIndex++
        p.currentTime = timedEvent.RelativeTime
        p.mu.Unlock()
    }

    return nil
}

func (p *PlaybackEngine) Pause() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.paused = true
}

func (p *PlaybackEngine) Resume() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.paused = false
}

func (p *PlaybackEngine) SetSpeed(speed float64) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.speed = speed
}

func (p *PlaybackEngine) Seek(position time.Duration) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Binary search for event at position
    for i, event := range p.recording.Events {
        if event.RelativeTime >= position {
            p.currentIndex = i
            p.currentTime = position
            return
        }
    }
}
```

### 5.3 Playback Controls

```go
// Playback UI controls
type PlaybackControls struct {
    engine    *PlaybackEngine

    playBtn   *tview.Button
    pauseBtn  *tview.Button
    stopBtn   *tview.Button
    seekBar   *tview.ProgressBar
    speedCtrl *tview.Dropdown

    timeline  *TimelineView
}

// Timeline visualization
type TimelineView struct {
    recording Recording

    // Event markers
    markers   []TimelineMarker
}

type TimelineMarker struct {
    Time      time.Duration
    EventType EventType
    Label     string
    Color     tcell.Color
}

func (t *TimelineView) Render() string {
    // Create visual timeline with markers
    duration := t.recording.Duration
    width := 100 // Terminal width

    timeline := make([]rune, width)
    for i := range timeline {
        timeline[i] = '─'
    }

    // Add markers
    for _, marker := range t.markers {
        pos := int(float64(marker.Time) / float64(duration) * float64(width))
        if pos >= 0 && pos < width {
            timeline[pos] = '●'
        }
    }

    return string(timeline)
}
```

---

## 6. Export Formats

### 6.1 JSON Export

```go
// ConversationExport represents complete conversation data
type ConversationExport struct {
    Metadata ConversationMetadata `json:"metadata"`
    Events   []TimedEvent         `json:"events"`
    Messages []MessageExport      `json:"messages"`
    Agents   []AgentExport        `json:"agents"`
    Metrics  ConversationMetrics  `json:"metrics"`
}

type ConversationMetadata struct {
    ConversationID string    `json:"conversation_id"`
    StartTime      time.Time `json:"start_time"`
    EndTime        time.Time `json:"end_time"`
    Duration       string    `json:"duration"`
    Mode           string    `json:"mode"`
    TotalTurns     int       `json:"total_turns"`
    Version        string    `json:"version"` // AgentPipe version
}

type MessageExport struct {
    MessageID    string    `json:"message_id"`
    AgentID      string    `json:"agent_id"`
    AgentName    string    `json:"agent_name"`
    AgentType    string    `json:"agent_type"`
    TurnNumber   int       `json:"turn_number"`
    Content      string    `json:"content"`
    Timestamp    time.Time `json:"timestamp"`

    // Metrics
    TokensInput  int     `json:"tokens_input"`
    TokensOutput int     `json:"tokens_output"`
    TokensTotal  int     `json:"tokens_total"`
    Cost         float64 `json:"cost"`
    DurationMS   int64   `json:"duration_ms"`

    // Artifacts
    Artifacts    []ArtifactExport `json:"artifacts,omitempty"`
}

type AgentExport struct {
    AgentID      string   `json:"agent_id"`
    AgentName    string   `json:"agent_name"`
    AgentType    string   `json:"agent_type"`
    Model        string   `json:"model"`
    Version      string   `json:"version"`
    MessageCount int      `json:"message_count"`
    TotalTokens  int      `json:"total_tokens"`
    TotalCost    float64  `json:"total_cost"`
}

type ConversationMetrics struct {
    TotalMessages   int     `json:"total_messages"`
    TotalTokens     int     `json:"total_tokens"`
    TotalCost       float64 `json:"total_cost"`
    AvgTokensPerMsg int     `json:"avg_tokens_per_message"`
    AvgCostPerMsg   float64 `json:"avg_cost_per_message"`
    AvgDurationMS   int64   `json:"avg_duration_ms"`
}

func ExportToJSON(conversation *Conversation, includeEvents bool) ([]byte, error) {
    export := ConversationExport{
        Metadata: extractMetadata(conversation),
        Messages: extractMessages(conversation),
        Agents:   extractAgents(conversation),
        Metrics:  calculateMetrics(conversation),
    }

    if includeEvents {
        export.Events = conversation.Events
    }

    return json.MarshalIndent(export, "", "  ")
}
```

### 6.2 Markdown Export

```go
func ExportToMarkdown(conversation *Conversation) (string, error) {
    var buf strings.Builder

    // Header
    buf.WriteString("# AgentPipe Conversation\n\n")
    buf.WriteString(fmt.Sprintf("**ID:** %s\n", conversation.ID))
    buf.WriteString(fmt.Sprintf("**Started:** %s\n", conversation.StartTime.Format(time.RFC3339)))
    buf.WriteString(fmt.Sprintf("**Duration:** %s\n", conversation.Duration))
    buf.WriteString(fmt.Sprintf("**Mode:** %s\n", conversation.Mode))
    buf.WriteString("\n---\n\n")

    // Participants
    buf.WriteString("## Participants\n\n")
    for _, agent := range conversation.Agents {
        buf.WriteString(fmt.Sprintf("- **%s** (%s) - %s\n",
            agent.Name, agent.Type, agent.Model))
    }
    buf.WriteString("\n---\n\n")

    // Conversation
    buf.WriteString("## Conversation\n\n")
    for i, msg := range conversation.Messages {
        buf.WriteString(fmt.Sprintf("### Turn %d: %s\n\n", i+1, msg.AgentName))
        buf.WriteString(msg.Content)
        buf.WriteString("\n\n")

        // Metrics
        buf.WriteString(fmt.Sprintf("*Tokens: %d | Duration: %dms | Cost: $%.4f*\n\n",
            msg.TokensTotal, msg.DurationMS, msg.Cost))

        // Artifacts
        if len(msg.Artifacts) > 0 {
            buf.WriteString("**Artifacts:**\n")
            for _, artifact := range msg.Artifacts {
                buf.WriteString(fmt.Sprintf("- %s: `%s`\n", artifact.Type, artifact.Name))
            }
            buf.WriteString("\n")
        }

        buf.WriteString("---\n\n")
    }

    // Summary
    buf.WriteString("## Summary\n\n")
    buf.WriteString(fmt.Sprintf("- **Total Messages:** %d\n", len(conversation.Messages)))
    buf.WriteString(fmt.Sprintf("- **Total Tokens:** %d\n", conversation.TotalTokens))
    buf.WriteString(fmt.Sprintf("- **Total Cost:** $%.4f\n", conversation.TotalCost))
    buf.WriteString(fmt.Sprintf("- **Avg Duration:** %dms\n", conversation.AvgDurationMS))

    return buf.String(), nil
}
```

### 6.3 HTML Export (Interactive)

```go
// HTML export with embedded JavaScript for playback
func ExportToHTML(conversation *Conversation, includePlayback bool) (string, error) {
    tmpl := template.Must(template.New("conversation").Parse(htmlTemplate))

    data := struct {
        Conversation ConversationExport
        IncludePlayback bool
        CSS string
        JS  string
    }{
        Conversation: exportConversation(conversation),
        IncludePlayback: includePlayback,
        CSS: embedCSS(),
        JS:  embedJS(),
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", err
    }

    return buf.String(), nil
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>AgentPipe Conversation - {{.Conversation.Metadata.ConversationID}}</title>
    <style>{{.CSS}}</style>
</head>
<body>
    <div class="container">
        <header>
            <h1>AgentPipe Conversation</h1>
            <div class="metadata">
                <span>ID: {{.Conversation.Metadata.ConversationID}}</span>
                <span>Duration: {{.Conversation.Metadata.Duration}}</span>
                <span>Mode: {{.Conversation.Metadata.Mode}}</span>
            </div>
        </header>

        {{if .IncludePlayback}}
        <div class="playback-controls">
            <button id="playBtn">▶ Play</button>
            <button id="pauseBtn">⏸ Pause</button>
            <input type="range" id="seekBar" min="0" max="100" value="0">
            <select id="speedCtrl">
                <option value="0.5">0.5x</option>
                <option value="1.0" selected>1.0x</option>
                <option value="2.0">2.0x</option>
                <option value="4.0">4.0x</option>
            </select>
        </div>
        {{end}}

        <div class="conversation">
            {{range .Conversation.Messages}}
            <div class="message" data-agent="{{.AgentName}}" data-turn="{{.TurnNumber}}">
                <div class="message-header">
                    <span class="agent-name">{{.AgentName}}</span>
                    <span class="timestamp">{{.Timestamp}}</span>
                </div>
                <div class="message-content">{{.Content}}</div>
                <div class="message-metrics">
                    <span>Tokens: {{.TokensTotal}}</span>
                    <span>Duration: {{.DurationMS}}ms</span>
                    <span>Cost: ${{printf "%.4f" .Cost}}</span>
                </div>
            </div>
            {{end}}
        </div>
    </div>

    {{if .IncludePlayback}}
    <script>
        const events = {{.Conversation.Events}};
        {{.JS}}
    </script>
    {{end}}
</body>
</html>
`
```

---

## 7. Example Event Sequences

### 7.1 Simple Round-Robin Conversation

```
Timeline: [Agent1] → [Agent2] → [Agent3]

Events:
1. ConversationStartedEvent (t=0ms)
   - conversation_id: "conv-123"
   - mode: "round-robin"
   - agents: ["agent1", "agent2", "agent3"]

2. TurnStartedEvent (t=10ms)
   - turn_number: 1
   - eligible_agents: ["agent1"]

3. TurnAgentSelectedEvent (t=15ms)
   - agent_id: "agent1"
   - agent_name: "Claude"
   - selection_reason: "round-robin-first"

4. AgentStateEvent (t=20ms)
   - agent_id: "agent1"
   - new_state: "thinking"

5. MessageStartedEvent (t=100ms)
   - message_id: "msg-1"
   - agent_id: "agent1"
   - is_streaming: true

6. AgentStateEvent (t=110ms)
   - agent_id: "agent1"
   - new_state: "typing"

7. MessageChunkEvent (t=200ms)
   - message_id: "msg-1"
   - chunk_index: 0
   - delta: "Hello there! "

8. MessageChunkEvent (t=300ms)
   - message_id: "msg-1"
   - chunk_index: 1
   - delta: "Let me analyze this... "

9. MessageCompleteEvent (t=1500ms)
   - message_id: "msg-1"
   - tokens_total: 150
   - cost: 0.0002
   - duration_ms: 1400

10. AgentStateEvent (t=1510ms)
    - agent_id: "agent1"
    - new_state: "done"

11. TurnCompleteEvent (t=1520ms)
    - turn_number: 1
    - messages_count: 1

12. TurnStartedEvent (t=1530ms)
    - turn_number: 2
    - eligible_agents: ["agent2"]

... [repeat for agent2, agent3]
```

### 7.2 Reactive Mode with Interruption

```
Timeline: [Agent1] → [Agent2 interrupts] → [Agent3 reacts]

Events:
1. ConversationStartedEvent (t=0ms)
   - mode: "reactive"

2. TurnStartedEvent (t=10ms)
   - turn_number: 1

3. MessageStartedEvent (t=100ms)
   - agent_id: "agent1"
   - is_streaming: true

4. MessageChunkEvent (t=200ms)
   - agent_id: "agent1"
   - delta: "I think we should..."

5. AgentReactionEvent (t=250ms) ⚠️ INTERRUPT
   - reacting_agent_id: "agent2"
   - target_message_id: "msg-1"
   - reaction_type: "interrupt"
   - comment: "Wait, I have a concern about that approach"

6. AgentStateEvent (t=260ms)
   - agent_id: "agent1"
   - new_state: "waiting"

7. AgentStateEvent (t=270ms)
   - agent_id: "agent2"
   - new_state: "thinking"

8. MessageStartedEvent (t=300ms)
   - agent_id: "agent2"
   - message_id: "msg-2"

... [agent2 completes message]

9. AgentReactionEvent (t=2000ms)
   - reacting_agent_id: "agent3"
   - target_message_id: "msg-2"
   - reaction_type: "agree"
   - comment: "Good point, let me elaborate"

10. MessageStartedEvent (t=2100ms)
    - agent_id: "agent3"
    - message_id: "msg-3"
```

### 7.3 Long Operation with Progress

```
Timeline: [Agent] performing complex analysis

Events:
1. MessageStartedEvent (t=0ms)
   - agent_id: "agent1"
   - message_id: "msg-1"

2. AgentStateEvent (t=10ms)
   - agent_id: "agent1"
   - new_state: "thinking"
   - progress: {
       current: 0,
       total: 100,
       percentage: 0,
       message: "Initializing analysis..."
     }

3. ProgressStartedEvent (t=100ms)
   - operation: "code_analysis"
   - total_steps: 5

4. AgentStateEvent (t=1000ms)
   - new_state: "thinking"
   - progress: {
       current: 20,
       total: 100,
       percentage: 20,
       message: "Analyzing dependencies...",
       estimated_ms: 4000
     }

5. ProgressUpdateEvent (t=2000ms)
   - step: 2
   - message: "Checking security vulnerabilities..."

6. AgentStateEvent (t=3000ms)
   - progress: {
       percentage: 60,
       message: "Generating recommendations..."
     }

7. ProgressCompleteEvent (t=5000ms)
   - operation: "code_analysis"
   - duration_ms: 4900

8. AgentStateEvent (t=5100ms)
   - new_state: "typing"

9. MessageChunkEvent (t=5200ms)
   - delta: "Based on my analysis..."

10. MessageCompleteEvent (t=7000ms)
    - duration_ms: 7000
```

---

## 8. Implementation Approach

### 8.1 Phase 1: Core Event System (Week 1-2)

**Deliverables:**
- `pkg/events/` package with event type definitions
- `pkg/eventbus/` package with buffered channel implementation
- Unit tests for event bus (>90% coverage)
- Benchmarks for event throughput

**Tasks:**
1. Define all event types and interfaces
2. Implement BufferedEventBus with pub/sub
3. Add subscription pattern matching
4. Implement metrics collection
5. Write comprehensive tests

### 8.2 Phase 2: TUI Integration (Week 3-4)

**Deliverables:**
- Event-driven TUI rendering
- Real-time message streaming
- Agent status indicators
- Performance optimizations

**Tasks:**
1. Refactor TUI to subscribe to events
2. Implement streaming renderers (word-by-word)
3. Add agent status panel with live updates
4. Implement throttling and buffering
5. Performance testing and optimization

### 8.3 Phase 3: Historical Playback (Week 5)

**Deliverables:**
- Event recording system
- Playback engine
- Export formats (JSON, Markdown, HTML)

**Tasks:**
1. Implement EventRecorder
2. Build PlaybackEngine with controls
3. Create export templates
4. Add playback UI to TUI

### 8.4 Phase 4: Advanced Features (Week 6+)

**Deliverables:**
- Agent reactions system
- Progress indicators for long operations
- Artifact event integration
- Enhanced metrics and observability

**Tasks:**
1. Implement agent-to-agent reactions
2. Add progress tracking for complex operations
3. Integrate with artifact collection system
4. Build comprehensive observability dashboard

### 8.5 Testing Strategy

**Unit Tests:**
- Event bus publish/subscribe
- Event serialization/deserialization
- Throttling and buffering logic
- Playback engine

**Integration Tests:**
- TUI rendering with mock events
- Multi-agent conversations with events
- Event recording and playback
- Export format validation

**Performance Tests:**
- Event throughput benchmarks (target: 10,000 events/sec)
- Rendering frame rate (target: 30 FPS)
- Memory usage under load
- Latency measurements

**Benchmarks:**
```go
func BenchmarkEventBus_Publish(b *testing.B) {
    bus := NewBufferedEventBus(HighThroughputConfig)
    event := &MessageChunkEvent{
        MessageID: "msg-1",
        Content: "test",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bus.Publish(context.Background(), event)
    }
}

func BenchmarkTUI_RenderMessage(b *testing.B) {
    renderer := NewWordStreamRenderer(100 * time.Millisecond)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        renderer.AddChunk("This is a test chunk")
        renderer.Render()
    }
}
```

---

## 9. Testing Strategy

### 9.1 Event Generation for Testing

```go
// EventGenerator creates realistic event sequences for testing
type EventGenerator struct {
    conversationID string
    agentIDs       []string
    messageCounter int
    turnCounter    int
}

func (g *EventGenerator) GenerateConversation(turns int) []Event {
    var events []Event

    // Conversation started
    events = append(events, &ConversationStartedEvent{
        ConversationID: g.conversationID,
        Agents: g.agentIDs,
        Mode: "round-robin",
    })

    // Generate turns
    for t := 0; t < turns; t++ {
        events = append(events, g.generateTurn()...)
    }

    // Conversation complete
    events = append(events, &ConversationCompleteEvent{
        ConversationID: g.conversationID,
        TotalTurns: turns,
    })

    return events
}

func (g *EventGenerator) generateTurn() []Event {
    g.turnCounter++
    agentID := g.agentIDs[g.turnCounter % len(g.agentIDs)]

    return []Event{
        &TurnStartedEvent{TurnNumber: g.turnCounter},
        &AgentStateEvent{AgentID: agentID, NewState: StateThinking},
        &MessageStartedEvent{MessageID: fmt.Sprintf("msg-%d", g.messageCounter)},
        // ... more events
    }
}
```

### 9.2 Mock Event Bus

```go
// MockEventBus for testing subscribers
type MockEventBus struct {
    publishedEvents []Event
    subscribers     map[string]EventHandler
    mu              sync.Mutex
}

func (m *MockEventBus) Publish(ctx context.Context, event Event) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.publishedEvents = append(m.publishedEvents, event)

    // Immediately deliver to subscribers (synchronous for testing)
    for _, handler := range m.subscribers {
        handler(ctx, event)
    }

    return nil
}

func (m *MockEventBus) AssertEventPublished(t *testing.T, eventType EventType) {
    m.mu.Lock()
    defer m.mu.Unlock()

    for _, event := range m.publishedEvents {
        if event.Type() == eventType {
            return
        }
    }

    t.Errorf("Expected event type %s was not published", eventType)
}
```

---

## 10. Migration from v1

### 10.1 Backward Compatibility

```go
// v1 Orchestrator emits events for v2 compatibility
func (o *OrchestratorV1) SendMessage(ctx context.Context, agent Agent, message string) error {
    // v2 event emission
    if o.eventBus != nil {
        o.eventBus.Publish(ctx, &MessageStartedEvent{
            MessageID: generateID(),
            AgentID:   agent.ID(),
            AgentName: agent.Name(),
        })
    }

    // v1 logic
    response, err := agent.SendMessage(ctx, message)

    if o.eventBus != nil {
        if err != nil {
            o.eventBus.Publish(ctx, &MessageErrorEvent{
                Error: err.Error(),
            })
        } else {
            o.eventBus.Publish(ctx, &MessageCompleteEvent{
                Content: response,
            })
        }
    }

    return err
}
```

### 10.2 Incremental Migration

**Step 1:** Add event bus to existing orchestrator
**Step 2:** Emit events alongside existing logging
**Step 3:** Update TUI to subscribe to events
**Step 4:** Deprecate old logging-based updates
**Step 5:** Remove old code

---

## Appendices

### A. Event Type Quick Reference

| Category | Event Types | Count |
|----------|-------------|-------|
| Agent Lifecycle | registered, initialized, health_check, error, terminated | 5 |
| Agent State | idle, thinking, typing, responding, waiting, done | 6 |
| Message | started, chunk, complete, error, retry | 5 |
| Turn | started, agent_selected, complete, skipped | 4 |
| Conversation | started, paused, resumed, complete, error | 5 |
| Reaction | reaction, acknowledge, interrupt | 3 |
| Progress | started, update, complete | 3 |
| Artifact | created, updated, shared | 3 |
| System | metrics, warning | 2 |
| **Total** | | **36** |

### B. Configuration Examples

**High Performance Config**
```yaml
event_bus:
  buffer_size: 10000
  subscriber_buffer: 1000
  drop_on_full: true
  warn_threshold: 0.9

rendering:
  streaming_mode: word
  render_rate: 100ms
  max_batch: 50
  batch_window: 16ms  # 60 FPS
```

**Reliable Config**
```yaml
event_bus:
  buffer_size: 5000
  subscriber_buffer: 500
  drop_on_full: false
  warn_threshold: 0.7

rendering:
  streaming_mode: sentence
  render_rate: 200ms
  max_batch: 20
  batch_window: 33ms  # 30 FPS
```

### C. Performance Benchmarks

Expected performance targets:

| Metric | Target | Measured |
|--------|--------|----------|
| Event Throughput | 10,000 events/sec | TBD |
| Event Latency | <5ms p99 | TBD |
| Render FPS | 30 FPS | TBD |
| Render Latency | <33ms | TBD |
| Memory (10K events) | <50MB | TBD |
| Buffer Utilization | <80% | TBD |

---

## Summary

This real-time feedback and event system provides AgentPipe v2 with:

✅ **Complete Visibility** - Users see everything as it happens
✅ **Smooth Performance** - Smart buffering and throttling for 30+ FPS
✅ **Rich Events** - 36 event types covering all conversation aspects
✅ **Extensible Architecture** - Easy to add new event types
✅ **Historical Playback** - Full conversation replay with timing
✅ **Multiple Export Formats** - JSON, Markdown, HTML with playback
✅ **Production-Ready** - Comprehensive testing and benchmarks

The system is designed to be non-blocking, performant, and provide an exceptional user experience watching agent conversations unfold in real-time.
