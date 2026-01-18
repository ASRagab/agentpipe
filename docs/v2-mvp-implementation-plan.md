# AgentPipe v2 MVP Implementation Plan

## Executive Summary

**Goal**: Ship a working multi-agent conversation platform in 4 weeks where users send a message and all agents respond in real-time, with agents seeing each other's responses.

**Core Value Proposition**: Real-time multi-agent collaboration with streaming responses and minimal latency.

**Target Release**: 4 weeks from kickoff

---

## 1. MVP Feature Set

### Must-Have Features (Week 1-4)

#### Core Conversation Loop ✅
- **User Input**: User sends a single message via TUI
- **Broadcast**: Message is sent to all configured agents
- **Parallel Execution**: All agents process simultaneously
- **Real-time Streaming**: Agent responses stream back as they arrive
- **Shared Context**: Each agent sees all prior messages (user + other agents)
- **Turn-based Flow**: After all agents respond, user can send next message

#### Agent Support ✅
- **API-Based Agents**: OpenRouter (400+ models), Claude API, OpenAI API
- **CLI-Based Agents**: Claude CLI, Gemini CLI (legacy compatibility)
- **Agent Interface**: Unified interface for both CLI and API agents
- **Health Checks**: Verify agent availability before conversation starts

#### TUI Interface ✅
- **Three-Panel Layout**: Agent list | Conversation | User input
- **Real-time Updates**: Streaming responses appear as agents type
- **Agent Status**: Show which agents are typing/responding/done
- **Color Coding**: Different colors per agent for readability
- **Metrics Display**: Tokens, cost, duration per agent response

#### Conversation Management ✅
- **Session Persistence**: Save/resume conversations
- **Message History**: Full conversation context maintained
- **Export**: Save conversations to JSON/Markdown
- **Basic Config**: YAML config for agents and settings

### Nice-to-Have (v2.1 - Post-MVP)

#### Advanced Conversation Features 🔜
- **@Mentions**: User can address specific agents (`@claude what do you think?`)
- **Agent-to-Agent**: Agents can reply to each other's messages
- **Threading**: Side conversations between subset of agents
- **Conditional Participation**: Agents decide if they should respond
- **Turn-Taking Modes**: Round-robin, reactive, selective

#### Advanced TUI 🔜
- **Agent Details Modal**: View agent config, model, cost per message
- **Search/Filter**: Search conversation history, filter by agent
- **Split Views**: Multiple conversation views
- **Vim Keybindings**: Power user shortcuts

#### Advanced Export 🔜
- **Artifact Extraction**: Auto-save code blocks to files
- **Summary Generation**: AI-generated conversation summaries
- **Analytics**: Token usage trends, cost analysis

---

## 2. Minimal Architecture

### 2.1 High-Level Components

```
┌─────────────────────────────────────────────────────┐
│                    TUI Layer                        │
│  (bubbletea - displays conversation + user input)   │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│              Conversation Manager                   │
│  - Manages conversation state                       │
│  - Coordinates agent pool                           │
│  - Emits events to TUI                              │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│                 Agent Pool                          │
│  - Parallel execution of agent requests             │
│  - Streaming response aggregation                   │
│  - Error handling and retries                       │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│                Event Bus                            │
│  - Message events (user, agent, system)             │
│  - Status events (typing, done, error)              │
│  - Metrics events (tokens, cost, duration)          │
└─────────────────────────────────────────────────────┘
```

### 2.2 Core Data Structures

```go
// Message represents any message in the conversation
type Message struct {
    ID        string    // Unique message ID
    Timestamp time.Time // When message was created
    Role      string    // "user", "agent", "system"
    AgentID   string    // ID of sender (empty for user)
    Content   string    // Message content
    Status    string    // "pending", "streaming", "complete", "error"
    Metrics   *Metrics  // Optional performance metrics
}

// Metrics captures agent response performance
type Metrics struct {
    Duration     time.Duration
    InputTokens  int
    OutputTokens int
    TotalTokens  int
    Model        string
    Cost         float64
}

// Agent represents a configured AI agent
type Agent struct {
    ID       string            // Unique agent ID
    Type     string            // "api" or "cli"
    Name     string            // Display name
    Model    string            // Model identifier
    Config   map[string]any    // Agent-specific config
    Adapter  AgentAdapter      // Implementation adapter
}

// Conversation represents the current conversation state
type Conversation struct {
    ID       string
    Messages []Message
    Agents   []Agent
    Started  time.Time
    Status   string // "active", "paused", "completed"
}

// Event represents an event on the event bus
type Event struct {
    Type      string    // "message.created", "agent.typing", etc.
    Timestamp time.Time
    Data      any       // Event-specific payload
}
```

### 2.3 Essential Interfaces

```go
// AgentAdapter handles communication with a specific agent
type AgentAdapter interface {
    // Initialize configures the adapter
    Initialize(config map[string]any) error

    // SendMessage sends messages and returns response
    SendMessage(ctx context.Context, messages []Message) (string, error)

    // StreamMessage sends messages and streams response to writer
    StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error

    // IsAvailable checks if agent is ready
    IsAvailable() bool

    // GetModel returns the model identifier
    GetModel() string
}

// ConversationManager orchestrates the conversation
type ConversationManager interface {
    // Start begins a new conversation
    Start(ctx context.Context) error

    // SendUserMessage sends a message from the user
    SendUserMessage(ctx context.Context, content string) error

    // GetMessages returns all conversation messages
    GetMessages() []Message

    // Subscribe registers an event listener
    Subscribe(eventType string, handler func(Event))
}

// EventBus handles event distribution
type EventBus interface {
    // Publish sends an event to all subscribers
    Publish(event Event)

    // Subscribe registers a handler for an event type
    Subscribe(eventType string, handler func(Event)) func()
}

// AgentPool manages parallel agent execution
type AgentPool interface {
    // ExecuteParallel sends message to all agents concurrently
    ExecuteParallel(ctx context.Context, messages []Message) []Response

    // GetStatus returns current execution status
    GetStatus() map[string]string // agentID -> status
}
```

---

## 3. MVP Implementation Timeline

### Week 1: Core Infrastructure

**Days 1-2: Data Structures & Interfaces**
- [ ] Define core data structures (Message, Agent, Conversation)
- [ ] Define essential interfaces (AgentAdapter, ConversationManager, EventBus)
- [ ] Create minimal test suite for data structures
- [ ] Set up project structure under `pkg/v2/`

**Days 3-4: Event Bus Implementation**
- [ ] Implement simple pub/sub event bus
- [ ] Add event types: message.created, agent.typing, agent.done, error
- [ ] Write unit tests for event bus
- [ ] Add thread-safe event handling

**Days 5-7: Agent Adapters**
- [ ] Implement OpenRouter API adapter (priority: widest model support)
- [ ] Implement Claude API adapter
- [ ] Add basic retry logic and error handling
- [ ] Write adapter integration tests with real APIs
- [ ] Add health check implementation

**Deliverable**: Working event bus + 2 API adapters with tests

---

### Week 2: Conversation Engine

**Days 8-10: Conversation Manager**
- [ ] Implement ConversationManager with message history
- [ ] Add SendUserMessage method with validation
- [ ] Integrate event bus for message events
- [ ] Add conversation state persistence (JSON)
- [ ] Write unit tests for conversation flow

**Days 11-12: Agent Pool**
- [ ] Implement parallel agent execution using goroutines
- [ ] Add streaming response aggregation
- [ ] Implement timeout and error handling
- [ ] Add agent status tracking (pending/typing/done/error)
- [ ] Write integration tests for parallel execution

**Days 13-14: Response Streaming**
- [ ] Implement real-time response streaming from agents
- [ ] Add buffering and event emission for partial responses
- [ ] Handle agent failures gracefully
- [ ] Test streaming with multiple concurrent agents

**Deliverable**: Working conversation engine that can handle user messages and parallel agent responses

---

### Week 3: TUI Interface

**Days 15-17: Basic TUI Layout**
- [ ] Set up bubbletea TUI framework
- [ ] Implement three-panel layout (agents | conversation | input)
- [ ] Add agent list with status indicators
- [ ] Add conversation view with message rendering
- [ ] Add user input textarea

**Days 18-19: Real-time Updates**
- [ ] Connect TUI to event bus
- [ ] Implement streaming message updates in UI
- [ ] Add typing indicators for active agents
- [ ] Show agent status in real-time
- [ ] Add smooth scrolling for new messages

**Days 20-21: Metrics & Polish**
- [ ] Display token count, cost, duration per message
- [ ] Add color coding per agent
- [ ] Implement basic keyboard shortcuts (quit, clear, etc.)
- [ ] Add help overlay
- [ ] Polish UI styling and formatting

**Deliverable**: Functional TUI that displays real-time multi-agent conversation

---

### Week 4: Integration & Testing

**Days 22-24: End-to-End Integration**
- [ ] Wire up TUI + ConversationManager + AgentPool
- [ ] Implement session save/resume
- [ ] Add basic YAML config loading
- [ ] Test with 2-3 agents simultaneously
- [ ] Fix critical bugs and edge cases

**Days 25-26: Testing & Documentation**
- [ ] Write integration tests for full conversation flow
- [ ] Test with different agent combinations
- [ ] Add example YAML configurations
- [ ] Write basic user guide (README)
- [ ] Document API for extending with new agents

**Days 27-28: MVP Release Preparation**
- [ ] Performance testing (latency, memory usage)
- [ ] Fix any remaining critical bugs
- [ ] Create release notes
- [ ] Tag v2.0.0-mvp release
- [ ] Deploy demo video/screenshots

**Deliverable**: Shippable v2.0.0-mvp release

---

## 4. Testing Strategy

### 4.1 Unit Testing (Day-to-Day)
- **Coverage Target**: >80% for core packages
- **Focus Areas**: Data structures, event bus, conversation manager
- **Tools**: Go's built-in testing + testify for assertions
- **Mocking**: Mock AgentAdapter for testing without real API calls

### 4.2 Integration Testing (Weekly)
- **Agent Adapters**: Test with real API endpoints (rate-limited)
- **Conversation Flow**: Test user message → agent responses → next message
- **Event Bus**: Test event propagation under load
- **Persistence**: Test save/resume with various conversation states

### 4.3 Manual Testing (End of Each Week)
- **TUI Smoke Tests**: Launch TUI, send messages, verify responses
- **Multi-Agent Tests**: Test with 2, 3, and 5 agents simultaneously
- **Error Scenarios**: Test network failures, API errors, timeouts
- **Performance**: Check latency and memory usage with streaming

### 4.4 Release Testing (Week 4)
- **End-to-End Tests**: Full conversation flows with various configs
- **Cross-Platform**: Test on macOS, Linux, Windows
- **Documentation**: Verify all examples work as documented
- **Regression**: Ensure no breaking changes from v1 (config compatibility)

---

## 5. Migration from v1 to v2

### 5.1 What to Keep from v1

#### Proven Components ✅
- **Agent Adapters**: Reuse OpenRouter, Claude CLI adapters (proven implementations)
- **Message Structure**: Keep `agent.Message` type (well-designed)
- **Config Format**: Keep YAML config structure (user-friendly)
- **Metrics**: Keep token counting and cost estimation utilities
- **Logging**: Keep structured logging (proven useful for debugging)

#### Battle-Tested Code ✅
- **pkg/client/**: OpenAI-compatible HTTP client (well-tested, has retry logic)
- **pkg/utils/tokens.go**: Token estimation (accurate, no need to change)
- **pkg/logger/**: Chat logging (works well for persistence)
- **pkg/config/**: YAML config loading (solid implementation)

### 5.2 What to Rebuild/Rethink

#### Orchestration Layer 🔨
- **Current**: `pkg/orchestrator/orchestrator.go` (1300 lines, complex turn-taking)
- **Problem**: Too many orchestration modes, sequential execution, complex state
- **v2 Approach**: Simplified ConversationManager, parallel-first, event-driven
- **Migration**: Extract message handling logic, discard complex turn-taking

#### TUI Implementation 🔨
- **Current**: `pkg/tui/tui.go` (monolithic, complex state management)
- **Problem**: Tightly coupled to orchestrator, hard to extend
- **v2 Approach**: Event-driven TUI that reacts to event bus
- **Migration**: Keep bubbletea framework, rebuild with event subscriptions

#### Agent Coordination 🔨
- **Current**: Sequential or round-robin execution in orchestrator
- **Problem**: Slow, no real-time streaming, complex retry logic
- **v2 Approach**: AgentPool with parallel goroutines, streaming-first
- **Migration**: Extract agent execution code, rebuild with channels

### 5.3 Config Compatibility

To maintain v1 → v2 compatibility:

```yaml
# v1 config (works in v2 with migration)
agents:
  - id: claude
    type: claude
    name: Claude
    model: claude-sonnet-4.5
    prompt: "You are a helpful assistant"

# v2 config (same structure, new features optional)
agents:
  - id: claude
    type: api  # New: explicit api vs cli
    name: Claude
    model: claude-sonnet-4.5
    prompt: "You are a helpful assistant"
    adapter: claude-api  # New: specific adapter
    config:  # New: adapter-specific settings
      api_key_env: ANTHROPIC_API_KEY
```

**Migration Strategy**:
1. Auto-detect v1 config (absence of `adapter` field)
2. Map `type: claude` → `adapter: claude-api` (if API key present) or `claude-cli`
3. Warn user about deprecated config format
4. Auto-upgrade config on first run (with backup)

### 5.4 Data Migration

**Conversation History**:
- v1 stores as JSON in `~/.agentpipe/chats/`
- v2 uses same directory, upgraded schema
- Migration: Add `version` field, convert on load

**State Files**:
- v1 has resumable conversations
- v2 keeps same resume feature
- Migration: Convert turn-based state to event-based state

---

## 6. Success Criteria

### 6.1 Functional Requirements

- [ ] User can send a message and see 3+ agents respond in parallel
- [ ] Responses stream in real-time as agents generate them
- [ ] Each agent sees full conversation history (user + other agents)
- [ ] TUI shows agent status indicators (typing, done, error)
- [ ] Metrics display correctly (tokens, cost, duration)
- [ ] Conversation can be saved and resumed
- [ ] Config can specify agents, models, and basic settings

### 6.2 Performance Requirements

- [ ] First response latency: <2s for fast models (haiku, gpt-4o-mini)
- [ ] Streaming latency: <100ms between chunks
- [ ] Memory usage: <100MB for 100-message conversation
- [ ] TUI responsiveness: <16ms frame time (60fps)
- [ ] Parallel execution: N agents complete in ~time of slowest agent (not N×slowest)

### 6.3 Quality Requirements

- [ ] Test coverage: >80% for core packages
- [ ] Zero panics in normal operation
- [ ] Graceful degradation when agents fail
- [ ] Clear error messages for user-facing issues
- [ ] Documentation covers all MVP features

### 6.4 User Experience

- [ ] First-time setup: <5 minutes (install + config)
- [ ] Learning curve: <10 minutes to first conversation
- [ ] TUI is intuitive (no manual required for basic use)
- [ ] Error messages are actionable
- [ ] Performance feels snappy (no noticeable lag)

---

## 7. Risks & Mitigations

### 7.1 Technical Risks

**Risk**: Parallel execution causes race conditions
- **Likelihood**: Medium
- **Impact**: High (data corruption, crashes)
- **Mitigation**: Use channels for agent communication, extensive concurrency testing

**Risk**: Streaming responses overwhelm TUI rendering
- **Likelihood**: Medium
- **Impact**: Medium (UI lag, poor UX)
- **Mitigation**: Buffer events, rate-limit UI updates, benchmark with fast models

**Risk**: Agent API rate limits block parallel execution
- **Likelihood**: High
- **Impact**: Medium (slower responses)
- **Mitigation**: Per-agent rate limiting, queue management, user warnings

### 7.2 Schedule Risks

**Risk**: Agent adapter complexity underestimated
- **Likelihood**: Medium
- **Impact**: High (delays Week 1)
- **Mitigation**: Start with OpenRouter (simplest), Claude API as backup

**Risk**: TUI integration harder than expected
- **Likelihood**: Low
- **Impact**: High (delays Week 3)
- **Mitigation**: Prototype TUI early, use proven bubbletea patterns

**Risk**: Testing takes longer than planned
- **Likelihood**: Medium
- **Impact**: Medium (delays release)
- **Mitigation**: Test continuously, automate integration tests, cut nice-to-haves

### 7.3 User Experience Risks

**Risk**: Parallel responses are confusing to users
- **Likelihood**: Low
- **Impact**: Medium (poor UX)
- **Mitigation**: Clear visual indicators, agent colors, typing animations

**Risk**: Config migration breaks existing setups
- **Likelihood**: Medium
- **Impact**: High (user frustration)
- **Mitigation**: Thorough migration testing, backup configs, rollback option

---

## 8. Post-MVP Roadmap (v2.1+)

### v2.1 - Advanced Conversation (Week 5-6)
- Agent-to-agent direct messaging (`@mentions`)
- Conditional participation (agents decide to respond)
- Conversation threading (side conversations)
- Advanced turn-taking modes

### v2.2 - Enhanced TUI (Week 7-8)
- Agent detail modals
- Search and filter
- Vim keybindings
- Split conversation views
- Custom themes

### v2.3 - Artifact System (Week 9-10)
- Auto-extract code blocks to files
- Artifact reference in conversations
- Artifact versioning
- Multi-file artifacts

### v2.4 - Analytics & Insights (Week 11-12)
- Token usage trends
- Cost analytics per agent/conversation
- Response quality metrics
- Conversation summaries

---

## 9. Development Guidelines

### 9.1 Code Organization

```
pkg/v2/
├── core/           # Core data structures (Message, Agent, Conversation)
├── events/         # Event bus implementation
├── manager/        # ConversationManager
├── pool/           # AgentPool for parallel execution
├── adapters/       # Agent adapters (api/, cli/)
├── tui/            # TUI implementation
├── config/         # Config loading and migration
└── persistence/    # Conversation save/load
```

### 9.2 Coding Standards

- **Go Version**: 1.24+ (match current)
- **Error Handling**: Always return errors, never panic in library code
- **Concurrency**: Use channels and select, avoid shared state
- **Testing**: Write tests alongside code, not after
- **Documentation**: Godoc for all exported types/functions
- **Naming**: Clear, descriptive names (no abbreviations unless standard)

### 9.3 Git Workflow

- **Branch**: `feature/v2-mvp`
- **Commits**: Small, focused commits with clear messages
- **PRs**: Self-review before requesting review, link to plan sections
- **Tags**: Weekly tags (`v2.0.0-alpha.1`, `v2.0.0-alpha.2`, etc.)
- **Release**: `v2.0.0-mvp` after Week 4

### 9.4 Daily Workflow

1. **Morning**: Review yesterday's progress, pick today's tasks
2. **Code**: Implement feature with tests
3. **Test**: Run unit tests + integration tests
4. **Commit**: Push working code (don't push broken code)
5. **Evening**: Update progress in this doc, note blockers

---

## 10. Success Metrics

### 10.1 Development Velocity

- **Week 1**: Event bus + 2 adapters working
- **Week 2**: Conversation engine handling parallel responses
- **Week 3**: TUI showing real-time conversation
- **Week 4**: MVP released and documented

### 10.2 Code Quality

- **Test Coverage**: >80% for pkg/v2/
- **Linting**: Zero golangci-lint errors
- **Build**: Green CI on every commit
- **Documentation**: All public APIs documented

### 10.3 User Validation

- **Internal Testing**: 3+ team members use MVP
- **GitHub Issues**: <5 critical bugs in first week
- **User Feedback**: Positive feedback on core conversation loop
- **Adoption**: 10+ external users try MVP in first month

---

## Conclusion

This MVP focuses on **shipping a working product in 4 weeks** rather than building a perfect system. The core value is **real-time multi-agent collaboration**, and everything else is secondary.

**Key Principles**:
1. **Parallel-first**: All agents respond simultaneously
2. **Streaming-first**: Responses appear as they're generated
3. **Event-driven**: Clean separation between engine and UI
4. **Simple but extensible**: Easy to add features in v2.1+

**Non-Goals for MVP**:
- Complex orchestration modes (round-robin, reactive, etc.)
- Agent-to-agent messaging (comes in v2.1)
- Advanced TUI features (search, split views, etc.)
- Artifact extraction (comes in v2.3)
- Analytics and insights (comes in v2.4)

**Ship the MVP, iterate from there.**
