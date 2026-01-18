# AgentPipe v2 Architecture Diagrams

## 1. High-Level System Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                         TUI Layer                             │
│                      (bubbletea UI)                           │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │ Agent List  │  │ Conversation │  │ User Input   │        │
│  │  - Claude   │  │   Messages   │  │   Textarea   │        │
│  │  - GPT-4    │  │   Streaming  │  │              │        │
│  │  - Gemini   │  │   Metrics    │  │   [Send]     │        │
│  └─────────────┘  └──────────────┘  └──────────────┘        │
└─────────────────────────┬─────────────────────────────────────┘
                          │ Events (subscribe)
                          │ Commands (send message)
                          ▼
┌───────────────────────────────────────────────────────────────┐
│                   Conversation Manager                        │
│  - Maintains conversation state                               │
│  - Coordinates agent pool                                     │
│  - Publishes events to event bus                              │
│  - Handles user messages                                      │
└─────────────────────────┬─────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
        ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Agent Pool  │  │  Event Bus   │  │ Persistence  │
│              │  │              │  │              │
│ - Parallel   │  │ - Pub/Sub    │  │ - Save       │
│   execution  │  │ - Events     │  │ - Resume     │
│ - Streaming  │  │ - Async      │  │ - Export     │
└──────┬───────┘  └──────────────┘  └──────────────┘
       │
       │ Executes
       ▼
┌───────────────────────────────────────────────────────────────┐
│                     Agent Adapters                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ API Adapter  │  │ API Adapter  │  │ CLI Adapter  │       │
│  │  OpenRouter  │  │  Claude      │  │  Gemini      │       │
│  │              │  │              │  │              │       │
│  │ HTTP Client  │  │ HTTP Client  │  │ exec.Command │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
└─────────┼──────────────────┼──────────────────┼──────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
    ┌──────────┐       ┌──────────┐       ┌──────────┐
    │ OpenRouter│       │ Claude   │       │ Gemini   │
    │   API     │       │   API    │       │   CLI    │
    └──────────┘       └──────────┘       └──────────┘
```

---

## 2. Message Flow Diagram

### User Sends Message → All Agents Respond

```
User                TUI          ConversationManager      EventBus       AgentPool        Agent Adapters
 │                   │                    │                  │              │                    │
 │ Types message     │                    │                  │              │                    │
 │──────────────────>│                    │                  │              │                    │
 │                   │ SendUserMessage()  │                  │              │                    │
 │                   │───────────────────>│                  │              │                    │
 │                   │                    │ Publish(         │              │                    │
 │                   │                    │  "message.user") │              │                    │
 │                   │                    │─────────────────>│              │                    │
 │                   │<────────────────────────────Event─────│              │                    │
 │                   │ [Update UI:        │                  │              │                    │
 │                   │  Show user msg]    │                  │              │                    │
 │                   │                    │ ExecuteParallel()│              │                    │
 │                   │                    │─────────────────────────────────>│                    │
 │                   │                    │                  │              │ SendMessage()      │
 │                   │                    │                  │              ├───────────────────>│
 │                   │                    │                  │              │ (Claude)           │
 │                   │                    │                  │              │                    │
 │                   │                    │                  │              ├───────────────────>│
 │                   │                    │                  │              │ (GPT-4)            │
 │                   │                    │                  │              │                    │
 │                   │                    │                  │              ├───────────────────>│
 │                   │                    │                  │              │ (Gemini)           │
 │                   │                    │                  │              │                    │
 │                   │                    │ Publish(         │              │                    │
 │                   │                    │  "agent.typing") │              │                    │
 │                   │                    │─────────────────>│              │                    │
 │                   │<────────────────────────────Event─────│              │                    │
 │                   │ [Update UI:        │                  │              │                    │
 │                   │  Show typing...]   │                  │              │                    │
 │                   │                    │                  │              │<── Response chunk  │
 │                   │                    │ Publish(         │              │                    │
 │                   │                    │  "message.chunk")│              │                    │
 │                   │                    │─────────────────>│              │                    │
 │                   │<────────────────────────────Event─────│              │                    │
 │                   │ [Update UI:        │                  │              │                    │
 │                   │  Stream text...]   │                  │              │                    │
 │                   │                    │                  │              │<── Complete        │
 │                   │                    │ Publish(         │              │                    │
 │                   │                    │  "agent.done")   │              │                    │
 │                   │                    │─────────────────>│              │                    │
 │                   │<────────────────────────────Event─────│              │                    │
 │                   │ [Update UI:        │                  │              │                    │
 │                   │  Mark complete,    │                  │              │                    │
 │                   │  show metrics]     │                  │              │                    │
```

---

## 3. Parallel Agent Execution Detail

```
ConversationManager
       │
       │ ExecuteParallel(messages)
       ▼
  ┌────────────────────────────────────────┐
  │          Agent Pool                    │
  │                                        │
  │  For each agent, spawn goroutine:     │
  │                                        │
  │  ┌──────────────────────────────┐     │
  │  │ goroutine 1 (Claude)         │     │
  │  │  1. Call adapter.SendMessage │────────> HTTP Request to Claude API
  │  │  2. Stream response chunks   │<────────  HTTP Streaming Response
  │  │  3. Emit "message.chunk"     │─────────> Event Bus
  │  │  4. Emit "agent.done"        │─────────> Event Bus
  │  └──────────────────────────────┘     │
  │                                        │
  │  ┌──────────────────────────────┐     │
  │  │ goroutine 2 (GPT-4)          │     │
  │  │  1. Call adapter.SendMessage │────────> HTTP Request to OpenRouter
  │  │  2. Stream response chunks   │<────────  HTTP Streaming Response
  │  │  3. Emit "message.chunk"     │─────────> Event Bus
  │  │  4. Emit "agent.done"        │─────────> Event Bus
  │  └──────────────────────────────┘     │
  │                                        │
  │  ┌──────────────────────────────┐     │
  │  │ goroutine 3 (Gemini)         │     │
  │  │  1. Call adapter.SendMessage │────────> CLI Command
  │  │  2. Stream response chunks   │<────────  stdout
  │  │  3. Emit "message.chunk"     │─────────> Event Bus
  │  │  4. Emit "agent.done"        │─────────> Event Bus
  │  └──────────────────────────────┘     │
  │                                        │
  │  Wait for all goroutines to complete  │
  │  (using sync.WaitGroup)                │
  └────────────────────────────────────────┘
                   │
                   ▼
          Return aggregated responses
```

**Key Benefits of Parallel Execution**:
- Response time = slowest agent (not sum of all agents)
- User sees fastest responses immediately
- Failed agents don't block others
- Efficient use of network I/O

---

## 4. Event Bus Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       Event Bus                             │
│                                                             │
│  Event Types:                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ - message.created  (user/agent/system message)      │   │
│  │ - message.chunk    (streaming response chunk)       │   │
│  │ - agent.typing     (agent is generating response)   │   │
│  │ - agent.done       (agent finished responding)      │   │
│  │ - agent.error      (agent failed to respond)        │   │
│  │ - conversation.started                              │   │
│  │ - conversation.saved                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Subscribers:                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │    TUI     │  │ Persistence│  │   Metrics  │           │
│  │  (display) │  │  (save)    │  │ (tracking) │           │
│  └────────────┘  └────────────┘  └────────────┘           │
└─────────────────────────────────────────────────────────────┘

Implementation:
- Thread-safe map of eventType -> []handlers
- Non-blocking publish (goroutines for each handler)
- Unsubscribe capability (return cleanup function)
```

---

## 5. Data Flow: User Message to Agent Response

```
1. User Input
   ┌─────────────────────────────────────┐
   │ User types: "Explain async/await"   │
   └─────────────────────────────────────┘
                    │
                    ▼
2. TUI → ConversationManager
   ┌─────────────────────────────────────┐
   │ manager.SendUserMessage(ctx,        │
   │   "Explain async/await")            │
   └─────────────────────────────────────┘
                    │
                    ▼
3. ConversationManager: Add to History
   ┌─────────────────────────────────────┐
   │ conversation.Messages = append(     │
   │   conversation.Messages,            │
   │   Message{                          │
   │     Role: "user",                   │
   │     Content: "Explain async/await", │
   │     Timestamp: now,                 │
   │   })                                │
   └─────────────────────────────────────┘
                    │
                    ▼
4. ConversationManager: Emit Event
   ┌─────────────────────────────────────┐
   │ eventBus.Publish(Event{             │
   │   Type: "message.created",          │
   │   Data: message,                    │
   │ })                                  │
   └─────────────────────────────────────┘
                    │
         ┌──────────┴──────────┐
         ▼                     ▼
   ┌─────────┐           ┌──────────┐
   │   TUI   │           │ Persist  │
   │ Display │           │  Save    │
   └─────────┘           └──────────┘
                    │
                    ▼
5. ConversationManager: Execute Agents
   ┌─────────────────────────────────────┐
   │ agentPool.ExecuteParallel(          │
   │   conversation.Messages)            │
   └─────────────────────────────────────┘
                    │
         ┌──────────┼──────────┐
         ▼          ▼          ▼
    ┌────────┐ ┌────────┐ ┌────────┐
    │Claude  │ │ GPT-4  │ │Gemini  │
    │Adapter │ │Adapter │ │Adapter │
    └────────┘ └────────┘ └────────┘
         │          │          │
         ▼          ▼          ▼
    (API Call) (API Call) (CLI Call)
         │          │          │
         ▼          ▼          ▼
    (Response) (Response) (Response)
         │          │          │
         └──────────┼──────────┘
                    │
                    ▼
6. AgentPool: Stream Responses
   ┌─────────────────────────────────────┐
   │ For each response chunk:            │
   │   eventBus.Publish(Event{           │
   │     Type: "message.chunk",          │
   │     Data: {                         │
   │       AgentID: "claude",            │
   │       Content: "Async/await is..."  │
   │     }                               │
   │   })                                │
   └─────────────────────────────────────┘
                    │
                    ▼
               ┌─────────┐
               │   TUI   │
               │ Display │
               │ Chunk   │
               └─────────┘
                    │
                    ▼
7. AgentPool: Complete Response
   ┌─────────────────────────────────────┐
   │ conversation.Messages = append(     │
   │   conversation.Messages,            │
   │   Message{                          │
   │     Role: "agent",                  │
   │     AgentID: "claude",              │
   │     Content: fullResponse,          │
   │     Metrics: {...},                 │
   │   })                                │
   │                                     │
   │ eventBus.Publish(Event{             │
   │   Type: "agent.done",               │
   │   Data: message,                    │
   │ })                                  │
   └─────────────────────────────────────┘
                    │
         ┌──────────┴──────────┐
         ▼                     ▼
   ┌─────────┐           ┌──────────┐
   │   TUI   │           │ Persist  │
   │ Display │           │  Save    │
   │ Metrics │           │  Message │
   └─────────┘           └──────────┘
```

---

## 6. TUI Component Breakdown

```
┌────────────────────────────────────────────────────────────────────┐
│                        AgentPipe v2 TUI                            │
├────────────────┬───────────────────────────────┬───────────────────┤
│  Agent List    │     Conversation View         │   Status Bar      │
│  (20% width)   │     (80% width)               │   (top)           │
│                │                               │                   │
│ ┌────────────┐ │ ┌───────────────────────────┐ │ Connected: 3/3   │
│ │ 🟢 Claude  │ │ │ [USER] Explain async      │ │ Turn: 5          │
│ │    Typing  │ │ │                           │ │ Cost: $0.023     │
│ │    89ms    │ │ │ [CLAUDE|145ms|234t|$0.01] │ │                  │
│ │            │ │ │ Async/await is a modern   │ │                  │
│ ├────────────┤ │ │ syntax for handling...    │ │                  │
│ │ 🟡 GPT-4   │ │ │                           │ │                  │
│ │    Typing  │ │ │ [GPT-4|203ms|456t|$0.02]  │ │                  │
│ │    134ms   │ │ │ It allows you to write    │ │                  │
│ │            │ │ │ asynchronous code that... │ │                  │
│ ├────────────┤ │ │                           │ │                  │
│ │ 🔴 Gemini  │ │ │ [GEMINI|98ms|189t|$0.005] │ │                  │
│ │    Error   │ │ │ Think of async/await as   │ │                  │
│ │    Retry   │ │ │ syntactic sugar for...    │ │                  │
│ └────────────┘ │ │                           │ │                  │
│                │ │ ▌ (cursor)                │ │                  │
│ Help: h        │ └───────────────────────────┘ │                  │
│ Filter: f      │                               │                  │
│ Search: /      │                               │                  │
│ Quit: q        │                               │                  │
├────────────────┴───────────────────────────────┴───────────────────┤
│  User Input (Multi-line Textarea)                                 │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ Now explain promises vs async/await                          │ │
│  │ ▌                                                            │ │
│  └──────────────────────────────────────────────────────────────┘ │
│  [Ctrl+Enter to send] [Esc to clear] [Ctrl+C to quit]             │
└────────────────────────────────────────────────────────────────────┘

Legend:
🟢 = Ready/Done
🟡 = Typing/Processing
🔴 = Error
```

**TUI Update Strategy**:
1. Subscribe to all event types from event bus
2. On `message.chunk`: Append text to conversation view, scroll down
3. On `agent.typing`: Update agent status to "Typing"
4. On `agent.done`: Update agent status to "Done", display metrics
5. On `agent.error`: Update agent status to "Error", show error message
6. Rate-limit renders to 60fps (16ms frame time)

---

## 7. Configuration Structure (YAML)

```yaml
# AgentPipe v2 Configuration

# Conversation settings
conversation:
  mode: parallel  # Only mode in MVP (round-robin comes in v2.1)
  timeout: 30s    # Max time for agent to respond
  save_dir: ~/.agentpipe/conversations

# Agent definitions
agents:
  # API-based agent (OpenRouter)
  - id: gpt4
    type: api
    adapter: openrouter
    name: GPT-4
    model: openai/gpt-4
    config:
      api_key_env: OPENROUTER_API_KEY
      temperature: 0.7
      max_tokens: 2000

  # API-based agent (Claude)
  - id: claude
    type: api
    adapter: claude-api
    name: Claude
    model: claude-sonnet-4.5
    config:
      api_key_env: ANTHROPIC_API_KEY
      temperature: 0.7
      max_tokens: 2000

  # CLI-based agent (legacy support)
  - id: gemini
    type: cli
    adapter: gemini-cli
    name: Gemini
    model: gemini-pro
    config:
      cli_path: /usr/local/bin/gemini
      extra_flags: ["--json"]

# TUI settings
tui:
  theme: default
  agent_list_width: 20  # percentage
  show_metrics: true
  typing_indicator: true
  max_messages_display: 100

# Logging
logging:
  level: info
  file: ~/.agentpipe/logs/agentpipe.log
  structured: true

# Persistence
persistence:
  auto_save: true
  save_interval: 30s
  backup_count: 5
```

---

## 8. Package Structure (Go Modules)

```
pkg/v2/
├── core/
│   ├── message.go          // Message, Metrics types
│   ├── agent.go            // Agent type
│   ├── conversation.go     // Conversation type
│   └── events.go           // Event type
│
├── events/
│   ├── bus.go              // EventBus interface + implementation
│   └── bus_test.go         // Tests
│
├── manager/
│   ├── manager.go          // ConversationManager implementation
│   ├── manager_test.go     // Tests
│   └── integration_test.go // Integration tests
│
├── pool/
│   ├── pool.go             // AgentPool implementation
│   ├── executor.go         // Parallel execution logic
│   └── pool_test.go        // Tests
│
├── adapters/
│   ├── adapter.go          // AgentAdapter interface
│   ├── api/
│   │   ├── openrouter.go   // OpenRouter API adapter
│   │   ├── claude.go       // Claude API adapter
│   │   └── openai.go       // OpenAI API adapter
│   └── cli/
│       ├── claude.go       // Claude CLI adapter (legacy)
│       └── gemini.go       // Gemini CLI adapter (legacy)
│
├── tui/
│   ├── tui.go              // Main TUI implementation (bubbletea)
│   ├── components/
│   │   ├── agent_list.go   // Agent list component
│   │   ├── conversation.go // Conversation view component
│   │   └── input.go        // User input component
│   └── tui_test.go         // Tests
│
├── config/
│   ├── config.go           // Config loading
│   ├── migrate.go          // v1 → v2 config migration
│   └── config_test.go      // Tests
│
└── persistence/
    ├── save.go             // Conversation save/load
    ├── export.go           // Export to JSON/Markdown
    └── persistence_test.go // Tests
```

---

## 9. Sequence Diagram: Complete Conversation Flow

```
┌──────┐  ┌─────┐  ┌────────┐  ┌─────────┐  ┌─────────┐  ┌──────┐
│ User │  │ TUI │  │ Manager│  │EventBus │  │  Pool   │  │Agents│
└──┬───┘  └──┬──┘  └───┬────┘  └────┬────┘  └────┬────┘  └──┬───┘
   │         │         │            │            │          │
   │ Type    │         │            │            │          │
   │message  │         │            │            │          │
   │────────>│         │            │            │          │
   │         │ SendUserMessage()    │            │          │
   │         │────────>│            │            │          │
   │         │         │ Publish    │            │          │
   │         │         │(message)   │            │          │
   │         │         │───────────>│            │          │
   │         │<─────────────Event───│            │          │
   │         │         │            │            │          │
   │ See     │         │            │            │          │
   │message  │         │            │            │          │
   │<────────│         │            │            │          │
   │         │         │ ExecuteParallel()       │          │
   │         │         │────────────────────────>│          │
   │         │         │            │            │ Send     │
   │         │         │            │            │─────────>│
   │         │         │            │            │          │
   │         │         │            │            │          │
   │         │         │ Publish    │            │<─Chunk───│
   │         │         │(chunk)     │            │          │
   │         │         │<───────────────────────────────────│
   │         │         │───────────>│            │          │
   │         │<─────────────Event───│            │          │
   │         │         │            │            │          │
   │ See     │         │            │            │          │
   │chunk    │         │            │            │          │
   │<────────│         │            │            │          │
   │         │         │            │            │          │
   │ [Repeat for each chunk...]                 │          │
   │         │         │            │            │          │
   │         │         │ Publish    │            │<─Done────│
   │         │         │(done)      │            │          │
   │         │         │<───────────────────────────────────│
   │         │         │───────────>│            │          │
   │         │<─────────────Event───│            │          │
   │         │         │            │            │          │
   │ See     │         │            │            │          │
   │complete │         │            │            │          │
   │+metrics │         │            │            │          │
   │<────────│         │            │            │          │
   │         │         │            │            │          │
```

---

## 10. Error Handling & Resilience

```
┌─────────────────────────────────────────────────────────┐
│              Error Handling Strategy                    │
└─────────────────────────────────────────────────────────┘

1. Agent Adapter Errors
   ┌──────────────────────────────────────┐
   │ Network timeout                      │──> Retry 3× with backoff
   │ API rate limit                       │──> Wait + retry
   │ Invalid response                     │──> Parse error + log
   │ API key invalid                      │──> Fail fast + user message
   └──────────────────────────────────────┘

2. Parallel Execution Errors
   ┌──────────────────────────────────────┐
   │ One agent fails                      │──> Continue with others
   │ All agents fail                      │──> Return error to user
   │ Timeout exceeded                     │──> Cancel slow agents
   └──────────────────────────────────────┘

3. Event Bus Errors
   ┌──────────────────────────────────────┐
   │ Handler panic                        │──> Recover + log + continue
   │ Event queue full                     │──> Drop old events (FIFO)
   └──────────────────────────────────────┘

4. TUI Errors
   ┌──────────────────────────────────────┐
   │ Render panic                         │──> Recover + show error
   │ Terminal resize                      │──> Recalculate layout
   └──────────────────────────────────────┘

5. Persistence Errors
   ┌──────────────────────────────────────┐
   │ Save failed                          │──> Log + continue
   │ Load failed                          │──> Fallback to empty state
   │ Disk full                            │──> Warn user + disable save
   └──────────────────────────────────────┘

Monitoring:
- All errors logged with structured fields
- Metrics track error rates per component
- User sees actionable error messages
```

---

## 11. Performance Optimization Points

```
1. Parallel Execution
   ┌────────────────────────────────────────┐
   │ Before: Sequential (N × slowest agent) │
   │ After:  Parallel (1 × slowest agent)   │
   │ Improvement: 3-5× faster for 3-5 agents│
   └────────────────────────────────────────┘

2. Streaming Responses
   ┌────────────────────────────────────────┐
   │ Before: Wait for full response         │
   │ After:  Stream chunks as generated     │
   │ Improvement: Perceived 10× faster      │
   └────────────────────────────────────────┘

3. Event Bus (Non-blocking)
   ┌────────────────────────────────────────┐
   │ Before: Blocking event handlers        │
   │ After:  Async goroutines               │
   │ Improvement: No UI blocking            │
   └────────────────────────────────────────┘

4. TUI Rendering
   ┌────────────────────────────────────────┐
   │ Before: Render on every event          │
   │ After:  Rate-limited (60fps max)       │
   │ Improvement: Smooth UI, lower CPU      │
   └────────────────────────────────────────┘

5. Message History
   ┌────────────────────────────────────────┐
   │ Before: Full history to every agent    │
   │ After:  Sliding window (last 20 msgs)  │
   │ Improvement: Lower token costs         │
   └────────────────────────────────────────┘
```

---

## Summary

**Key Architectural Decisions**:

1. **Event-Driven**: Clean separation between business logic and UI
2. **Parallel-First**: All agents execute concurrently by default
3. **Streaming-First**: Responses appear as they're generated
4. **Modular**: Easy to add new adapters, event handlers, TUI components
5. **Resilient**: Errors don't cascade; failed agents don't block others

**What Makes This Simple**:
- Only one conversation mode (parallel)
- No complex turn-taking logic
- Event bus handles all communication
- Adapters hide implementation complexity

**What Makes This Extensible**:
- Add new agents = implement AgentAdapter interface
- Add new UI features = subscribe to events
- Add new export formats = implement exporter
- Add new orchestration modes = extend ConversationManager

This architecture supports the 4-week MVP goal while setting up clean extension points for v2.1+.
