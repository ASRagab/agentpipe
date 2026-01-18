# AgentPipe v2: Modular Architecture Proposal

**Version:** 2.0.0-alpha
**Date:** 2026-01-18
**Status:** Proposal
**Authors:** System Architecture Team

---

## Executive Summary

This document proposes a comprehensive architectural redesign for AgentPipe v2, focusing on modularity, extensibility, and maintainability. The architecture emphasizes clean separation of concerns, plugin-based extensibility, and event-driven coordination.

### Key Improvements Over v1
- **Domain-Driven Design**: Clear separation into bounded contexts
- **Plugin Architecture**: Hot-swappable components with well-defined contracts
- **Event-Driven Core**: Decoupled components communicating via events
- **Observability-First**: Built-in metrics, tracing, and logging
- **API-First Design**: Core functionality accessible via APIs
- **Enhanced Testing**: >90% coverage goal with comprehensive test strategies

---

## Table of Contents

1. [High-Level Architecture](#1-high-level-architecture)
2. [Domain Model](#2-domain-model)
3. [Component Architecture](#3-component-architecture)
4. [Plugin System](#4-plugin-system)
5. [Event-Driven Orchestration](#5-event-driven-orchestration)
6. [Configuration & State Management](#6-configuration--state-management)
7. [Observability Architecture](#7-observability-architecture)
8. [Extensibility Points](#8-extensibility-points)
9. [Data Flow Diagrams](#9-data-flow-diagrams)
10. [Migration Strategy](#10-migration-strategy)
11. [Implementation Roadmap](#11-implementation-roadmap)

---

## 1. High-Level Architecture

### 1.1 Architectural Principles

**SOLID Principles**
- **Single Responsibility**: Each component has one reason to change
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Interfaces over implementations
- **Interface Segregation**: Minimal, focused interfaces
- **Dependency Inversion**: Depend on abstractions

**Additional Principles**
- **Hexagonal Architecture**: Ports & adapters for external integrations
- **Domain-Driven Design**: Bounded contexts with ubiquitous language
- **Event Sourcing**: Conversation history as immutable event stream
- **CQRS Pattern**: Separate read/write models where beneficial
- **API-First**: Every feature accessible programmatically

### 1.2 Layered Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   CLI    │  │   TUI    │  │   HTTP   │  │  gRPC    │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                   APPLICATION LAYER                         │
│  ┌────────────────┐  ┌────────────────┐  ┌──────────────┐  │
│  │  Orchestration │  │  Conversation  │  │   Artifact   │  │
│  │    Services    │  │   Management   │  │  Processing  │  │
│  └────────────────┘  └────────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                     DOMAIN LAYER                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Agent   │  │ Message  │  │  Event   │  │  Policy  │   │
│  │  Domain  │  │  Domain  │  │  Domain  │  │  Domain  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                 INFRASTRUCTURE LAYER                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Agent   │  │  Event   │  │  State   │  │ Metrics  │   │
│  │ Adapters │  │   Bus    │  │  Store   │  │  Export  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Core Components

```
┌────────────────────────────────────────────────────────────┐
│                     AgentPipe Core                         │
│                                                            │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Plugin Registry & Loader               │  │
│  └─────────────────────────────────────────────────────┘  │
│                            ↓                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐   │
│  │  Agent   │  │  Event   │  │  State   │  │ Config  │   │
│  │  Pool    │  │   Bus    │  │ Manager  │  │ Service │   │
│  └──────────┘  └──────────┘  └──────────┘  └─────────┘   │
│                                                            │
│  ┌─────────────────────────────────────────────────────┐  │
│  │           Orchestration Engine                      │  │
│  │  • Turn Management  • Middleware Pipeline           │  │
│  │  • Retry Logic      • Rate Limiting                 │  │
│  └─────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
         ↓                ↓                 ↓
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Agent      │  │   Storage    │  │   External   │
│  Plugins     │  │   Plugins    │  │   Services   │
│              │  │              │  │              │
│ • CLI-based  │  │ • File       │  │ • Bridge API │
│ • API-based  │  │ • Database   │  │ • Metrics    │
│ • Custom     │  │ • S3/Cloud   │  │ • Webhooks   │
└──────────────┘  └──────────────┘  └──────────────┘
```

---

## 2. Domain Model

### 2.1 Bounded Contexts

**Core Contexts**

```
┌─────────────────────────────────────────────────────────┐
│                 AGENT CONTEXT                           │
│                                                         │
│  Entities:                                              │
│  • Agent        - Represents an AI agent instance       │
│  • AgentType    - Metadata about agent capabilities     │
│  • Capability   - What an agent can do                  │
│                                                         │
│  Value Objects:                                         │
│  • AgentID      - Unique identifier                     │
│  • ModelConfig  - Model-specific configuration          │
│  • RateLimit    - Rate limiting policy                  │
│                                                         │
│  Aggregates:                                            │
│  • AgentPool    - Collection of available agents        │
│                                                         │
│  Domain Services:                                       │
│  • AgentFactory     - Creates agent instances           │
│  • HealthChecker    - Monitors agent health             │
│  • VersionDetector  - Detects CLI versions              │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│              CONVERSATION CONTEXT                       │
│                                                         │
│  Entities:                                              │
│  • Conversation  - A multi-agent interaction session    │
│  • Turn          - Single round of agent responses      │
│  • Message       - Individual agent communication       │
│                                                         │
│  Value Objects:                                         │
│  • ConversationID - Unique identifier                   │
│  • MessageID      - Unique message identifier           │
│  • Timestamp      - When something occurred             │
│  • Metrics        - Performance/cost measurements       │
│                                                         │
│  Aggregates:                                            │
│  • ConversationHistory - Complete message timeline      │
│                                                         │
│  Domain Services:                                       │
│  • SummaryGenerator  - AI-powered summarization         │
│  • CostCalculator    - Estimates conversation costs     │
│  • TokenCounter      - Counts tokens in messages        │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│              ORCHESTRATION CONTEXT                      │
│                                                         │
│  Entities:                                              │
│  • OrchestrationStrategy - How agents take turns        │
│  • Policy                - Rules for agent selection    │
│                                                         │
│  Value Objects:                                         │
│  • Mode          - round-robin, reactive, free-form     │
│  • TurnPolicy    - When/how to select next agent        │
│  • RetryPolicy   - How to handle failures               │
│                                                         │
│  Domain Services:                                       │
│  • AgentSelector     - Chooses next agent               │
│  • TurnCoordinator   - Manages turn-taking              │
│  • RetryHandler      - Handles failures                 │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                 EVENT CONTEXT                           │
│                                                         │
│  Entities:                                              │
│  • Event         - Something that happened              │
│  • Subscription  - Event listener registration          │
│                                                         │
│  Value Objects:                                         │
│  • EventType     - Type of event                        │
│  • EventPayload  - Event data                           │
│  • Priority      - Event processing priority            │
│                                                         │
│  Domain Services:                                       │
│  • EventPublisher  - Publishes events                   │
│  • EventRouter     - Routes events to subscribers       │
└─────────────────────────────────────────────────────────┘
```

### 2.2 Ubiquitous Language

**Terms and Definitions**

| Term | Definition | Context |
|------|------------|---------|
| **Agent** | An AI entity capable of generating responses | Agent Context |
| **Adapter** | Infrastructure component that connects to agent CLIs/APIs | Infrastructure |
| **Conversation** | A series of turns between multiple agents | Conversation Context |
| **Turn** | One complete round where selected agent(s) respond | Orchestration |
| **Message** | A single piece of communication from an agent | Conversation |
| **Artifact** | Structured output extracted from agent responses | Conversation |
| **Event** | A notification that something occurred in the system | Event Context |
| **Policy** | A rule or strategy that governs behavior | Orchestration |
| **Middleware** | Processing layer that transforms messages | Infrastructure |
| **Bridge** | External service integration for streaming events | Infrastructure |

---

## 3. Component Architecture

### 3.1 Core Package Structure

```
agentpipe/
├── cmd/                          # Entry points
│   ├── agentpipe/               # Main CLI binary
│   ├── agent-server/            # Agent API server (new)
│   └── plugin-dev/              # Plugin development tools (new)
│
├── pkg/                          # Public packages
│   ├── core/                    # Core domain (new)
│   │   ├── agent/              # Agent domain
│   │   ├── conversation/       # Conversation domain
│   │   ├── orchestration/      # Orchestration domain
│   │   └── event/              # Event domain
│   │
│   ├── application/             # Application services (new)
│   │   ├── services/           # Application services
│   │   ├── dto/                # Data transfer objects
│   │   └── usecases/           # Use case implementations
│   │
│   ├── infrastructure/          # Infrastructure implementations
│   │   ├── adapters/           # Agent adapters (CLI/API)
│   │   ├── eventbus/           # Event bus implementations
│   │   ├── storage/            # State persistence
│   │   └── telemetry/          # Metrics, logging, tracing
│   │
│   ├── interfaces/              # Presentation layer (new)
│   │   ├── cli/                # CLI interface
│   │   ├── tui/                # Terminal UI
│   │   ├── http/               # HTTP API (new)
│   │   └── grpc/               # gRPC API (future)
│   │
│   └── plugin/                  # Plugin system (new)
│       ├── api/                # Plugin API contracts
│       ├── loader/             # Plugin discovery/loading
│       └── registry/           # Plugin registration
│
├── internal/                    # Private packages
│   ├── bridge/                 # Bridge implementation
│   ├── providers/              # Provider pricing
│   ├── registry/               # Agent registry
│   └── version/                # Version management
│
├── plugins/                     # Official plugins (new)
│   ├── agents/                 # Agent plugins
│   │   ├── claude/
│   │   ├── gemini/
│   │   └── openrouter/
│   ├── storage/                # Storage plugins
│   │   ├── filesystem/
│   │   └── postgresql/
│   └── exporters/              # Export format plugins
│       ├── markdown/
│       └── html/
│
├── api/                         # API definitions (new)
│   ├── proto/                  # Protocol buffers (gRPC)
│   └── openapi/                # OpenAPI specs (HTTP)
│
└── test/                        # Test utilities
    ├── integration/
    ├── e2e/
    └── fixtures/
```

### 3.2 Dependency Flow

```
┌──────────────────────────────────────────────────────┐
│              Dependency Inversion                    │
│                                                      │
│  Interfaces (pkg/core)     ←─────────┐              │
│       ↑                               │              │
│       │ implements                    │ depends on   │
│       │                               │              │
│  Infrastructure (pkg/infrastructure)  │              │
│       ↑                               │              │
│       │ uses                          │              │
│       │                               │              │
│  Application (pkg/application)  ──────┘              │
│       ↑                                              │
│       │ uses                                         │
│       │                                              │
│  Presentation (pkg/interfaces)                       │
└──────────────────────────────────────────────────────┘

Rule: Higher layers depend on lower layers
      Core has zero external dependencies
      Infrastructure implements core interfaces
      Application orchestrates infrastructure
      Presentation adapts application for users
```

### 3.3 Core Interfaces

**pkg/core/agent/agent.go**
```go
// Agent represents a conversational AI entity
type Agent interface {
    // Identity
    ID() AgentID
    Type() AgentType
    Name() string

    // Communication
    SendMessage(ctx context.Context, req MessageRequest) (MessageResponse, error)
    StreamMessage(ctx context.Context, req MessageRequest) (<-chan MessageChunk, error)

    // Lifecycle
    Initialize(config Config) error
    HealthCheck(ctx context.Context) HealthStatus
    Shutdown(ctx context.Context) error

    // Metadata
    Capabilities() []Capability
    Metadata() Metadata
}

// Repository for agent persistence
type Repository interface {
    Save(ctx context.Context, agent Agent) error
    FindByID(ctx context.Context, id AgentID) (Agent, error)
    FindByType(ctx context.Context, typ AgentType) ([]Agent, error)
    Delete(ctx context.Context, id AgentID) error
}

// Factory for creating agents
type Factory interface {
    Create(typ AgentType, config Config) (Agent, error)
    SupportedTypes() []AgentType
}
```

**pkg/core/conversation/conversation.go**
```go
// Conversation represents a multi-agent interaction
type Conversation interface {
    ID() ConversationID
    AddMessage(msg Message) error
    Messages() []Message
    Participants() []AgentID
    State() State
    Metrics() Metrics
}

// Repository for conversation persistence
type Repository interface {
    Save(ctx context.Context, conv Conversation) error
    FindByID(ctx context.Context, id ConversationID) (Conversation, error)
    Query(ctx context.Context, query Query) ([]Conversation, error)
}

// Event publisher for conversation events
type EventPublisher interface {
    PublishStarted(ctx context.Context, conv Conversation) error
    PublishMessage(ctx context.Context, msg Message) error
    PublishCompleted(ctx context.Context, conv Conversation) error
    PublishError(ctx context.Context, err Error) error
}
```

**pkg/core/orchestration/orchestrator.go**
```go
// Orchestrator coordinates agent interactions
type Orchestrator interface {
    // Lifecycle
    Start(ctx context.Context, config Config) error
    Stop(ctx context.Context) error

    // Orchestration
    ExecuteTurn(ctx context.Context, turn Turn) (Result, error)
    SelectNextAgent(ctx context.Context, state State) (AgentID, error)

    // Configuration
    SetStrategy(strategy Strategy)
    SetPolicy(policy Policy)
}

// Strategy defines how agents are selected
type Strategy interface {
    Name() string
    SelectNext(ctx context.Context, state State) (AgentID, error)
}

// Policy defines orchestration rules
type Policy interface {
    ShouldContinue(ctx context.Context, state State) bool
    ShouldRetry(ctx context.Context, err error, attempt int) bool
    Timeout(ctx context.Context) time.Duration
}
```

---

## 4. Plugin System

### 4.1 Plugin Architecture

**Goals**
- Hot-swappable components without recompilation
- Versioned plugin APIs with backward compatibility
- Secure sandboxing of untrusted plugins
- Performance isolation and resource limits
- Easy plugin development and testing

**Plugin Types**

```
┌──────────────────────────────────────────────────────┐
│                  Plugin Categories                   │
├──────────────────────────────────────────────────────┤
│                                                      │
│  1. Agent Plugins                                    │
│     • Implement agent.Agent interface                │
│     • Examples: claude, gemini, custom-ai            │
│     • Can be CLI-based, API-based, or embedded       │
│                                                      │
│  2. Storage Plugins                                  │
│     • Implement conversation.Repository              │
│     • Examples: filesystem, postgres, s3             │
│     • Handle persistence and retrieval               │
│                                                      │
│  3. Exporter Plugins                                 │
│     • Implement exporter.Exporter interface          │
│     • Examples: markdown, html, pdf                  │
│     • Transform conversations to output formats      │
│                                                      │
│  4. Middleware Plugins                               │
│     • Implement middleware.Middleware interface      │
│     • Examples: content-filter, encryption           │
│     • Process messages in pipeline                   │
│                                                      │
│  5. Strategy Plugins                                 │
│     • Implement orchestration.Strategy               │
│     • Examples: priority-based, ai-powered           │
│     • Control agent selection logic                  │
│                                                      │
│  6. Telemetry Plugins                                │
│     • Implement telemetry.Exporter interface         │
│     • Examples: prometheus, datadog, jaeger          │
│     • Export metrics, logs, traces                   │
└──────────────────────────────────────────────────────┘
```

### 4.2 Plugin API Contract

**pkg/plugin/api/v1/plugin.go**
```go
// Plugin is the base interface all plugins must implement
type Plugin interface {
    // Metadata
    Name() string
    Version() semver.Version
    Description() string
    Author() string

    // Lifecycle
    Initialize(config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    HealthCheck(ctx context.Context) HealthStatus

    // Dependencies
    Dependencies() []Dependency
    APIVersion() semver.Version
}

// AgentPlugin extends Plugin for agent implementations
type AgentPlugin interface {
    Plugin
    agent.Agent  // Embed core agent interface
}

// StoragePlugin extends Plugin for storage implementations
type StoragePlugin interface {
    Plugin
    conversation.Repository  // Embed repository interface
}

// Manifest describes a plugin's capabilities and requirements
type Manifest struct {
    Name        string           `json:"name"`
    Version     semver.Version   `json:"version"`
    APIVersion  semver.Version   `json:"api_version"`
    Type        PluginType       `json:"type"`
    Author      Author           `json:"author"`
    License     string           `json:"license"`
    Homepage    string           `json:"homepage"`

    // Capabilities
    Capabilities []string        `json:"capabilities"`

    // Dependencies
    Dependencies []Dependency    `json:"dependencies"`

    // Security
    Permissions  []Permission    `json:"permissions"`
    Checksum     string          `json:"checksum"`
    Signature    string          `json:"signature"`
}
```

### 4.3 Plugin Discovery & Loading

**Plugin Discovery Flow**

```
┌──────────────────────────────────────────────────────┐
│              Plugin Discovery Process                │
├──────────────────────────────────────────────────────┤
│                                                      │
│  1. Scan plugin directories                          │
│     ~/.agentpipe/plugins/                            │
│     /usr/local/share/agentpipe/plugins/              │
│     ./plugins/                                       │
│                                                      │
│  2. Read plugin.yaml manifest                        │
│     Validate schema and required fields              │
│     Check API version compatibility                  │
│                                                      │
│  3. Verify security                                  │
│     Check checksum/signature                         │
│     Validate permissions                             │
│     Assess security policy compliance                │
│                                                      │
│  4. Load plugin binary                               │
│     For Go plugins: plugin.Open()                    │
│     For WASM: wasmer runtime                         │
│     For external: RPC/gRPC                           │
│                                                      │
│  5. Initialize plugin                                │
│     Call Initialize(config)                          │
│     Resolve dependencies                             │
│     Register with plugin registry                    │
│                                                      │
│  6. Register with appropriate service                │
│     Agent plugins → AgentFactory                     │
│     Storage plugins → StorageService                 │
│     etc.                                             │
└──────────────────────────────────────────────────────┘
```

**Plugin Loader Implementation**

```go
// Loader handles plugin discovery and initialization
type Loader interface {
    // Discover finds plugins in configured directories
    Discover(ctx context.Context) ([]Plugin, error)

    // Load loads and initializes a specific plugin
    Load(ctx context.Context, path string) (Plugin, error)

    // Unload gracefully shuts down and unloads a plugin
    Unload(ctx context.Context, name string) error

    // Reload reloads a plugin (hot reload)
    Reload(ctx context.Context, name string) error
}

// Registry manages loaded plugins
type Registry interface {
    // Register adds a plugin to the registry
    Register(plugin Plugin) error

    // Unregister removes a plugin from the registry
    Unregister(name string) error

    // Get retrieves a plugin by name
    Get(name string) (Plugin, error)

    // List returns all registered plugins
    List() []Plugin

    // Find searches plugins by criteria
    Find(predicate func(Plugin) bool) []Plugin
}
```

### 4.4 Plugin Sandboxing

**Security Model**

```
┌──────────────────────────────────────────────────────┐
│              Plugin Security Layers                  │
├──────────────────────────────────────────────────────┤
│                                                      │
│  Layer 1: Permission System                          │
│    • Plugins declare required permissions            │
│    • User approves on first run                      │
│    • Examples: network, filesystem, env              │
│                                                      │
│  Layer 2: Resource Limits                            │
│    • CPU time limits                                 │
│    • Memory limits                                   │
│    • Network rate limits                             │
│    • File I/O limits                                 │
│                                                      │
│  Layer 3: Process Isolation                          │
│    • Plugins run in separate processes               │
│    • Communication via RPC/gRPC                      │
│    • Crash isolation                                 │
│                                                      │
│  Layer 4: Code Verification                          │
│    • Checksum validation                             │
│    • Digital signatures (optional)                   │
│    • Trusted plugin registry                         │
└──────────────────────────────────────────────────────┘
```

---

## 5. Event-Driven Orchestration

### 5.1 Event Architecture

**Event Flow**

```
┌────────────────────────────────────────────────────────┐
│                   Event-Driven Flow                    │
│                                                        │
│  Producer                 Event Bus           Consumer │
│  ┌────────┐             ┌──────────┐        ┌────────┐│
│  │ Agent  │──publish──→ │  Event   │──→     │ Bridge ││
│  │ Pool   │             │  Router  │   │    │ Emitter││
│  └────────┘             └──────────┘   │    └────────┘│
│                              │          │              │
│  ┌────────┐                  │          │    ┌────────┐│
│  │Orchestr│──publish──→      │          ├──→ │Metrics ││
│  │ -ator  │                  │          │    │Collector│
│  └────────┘                  │          │    └────────┘│
│                              │          │              │
│  ┌────────┐                  │          │    ┌────────┐│
│  │Convers-│──publish──→      │          └──→ │ Logger ││
│  │ation   │                  │               └────────┘│
│  └────────┘                  │                          │
│                              ↓                          │
│                      ┌──────────────┐                   │
│                      │ Event Store  │                   │
│                      │ (optional)   │                   │
│                      └──────────────┘                   │
└────────────────────────────────────────────────────────┘
```

### 5.2 Event Types

**Core Events**

```go
// ConversationEvents
type ConversationStarted struct {
    ConversationID  ConversationID
    Participants    []AgentID
    InitialPrompt   string
    Timestamp       time.Time
    Metadata        map[string]interface{}
}

type MessageCreated struct {
    ConversationID  ConversationID
    MessageID       MessageID
    AgentID         AgentID
    Content         string
    Turn            int
    Metrics         ResponseMetrics
    Timestamp       time.Time
}

type ConversationCompleted struct {
    ConversationID  ConversationID
    Status          CompletionStatus
    TotalMessages   int
    TotalTurns      int
    Duration        time.Duration
    TotalCost       float64
    Summary         *Summary
    Timestamp       time.Time
}

// AgentEvents
type AgentRegistered struct {
    AgentID    AgentID
    AgentType  AgentType
    Timestamp  time.Time
}

type AgentHealthChanged struct {
    AgentID        AgentID
    PreviousStatus HealthStatus
    CurrentStatus  HealthStatus
    Timestamp      time.Time
}

// OrchestrationEvents
type TurnStarted struct {
    ConversationID ConversationID
    TurnNumber     int
    SelectedAgent  AgentID
    Timestamp      time.Time
}

type TurnCompleted struct {
    ConversationID ConversationID
    TurnNumber     int
    AgentID        AgentID
    Duration       time.Duration
    Success        bool
    Error          error
    Timestamp      time.Time
}

// ErrorEvents
type AgentError struct {
    AgentID    AgentID
    Error      error
    Context    map[string]interface{}
    Timestamp  time.Time
}

type OrchestrationError struct {
    ConversationID ConversationID
    Error          error
    Context        map[string]interface{}
    Timestamp      time.Time
}
```

### 5.3 Event Bus Implementation

**Interface**

```go
// EventBus handles event publishing and subscription
type EventBus interface {
    // Publish sends an event to all subscribers
    Publish(ctx context.Context, event Event) error

    // Subscribe registers a handler for events matching the filter
    Subscribe(filter EventFilter, handler EventHandler) (Subscription, error)

    // Unsubscribe removes a subscription
    Unsubscribe(subscription Subscription) error

    // Close gracefully shuts down the event bus
    Close(ctx context.Context) error
}

// EventFilter determines which events a subscriber receives
type EventFilter func(event Event) bool

// EventHandler processes received events
type EventHandler func(ctx context.Context, event Event) error

// Subscription represents an active event subscription
type Subscription interface {
    ID() string
    Unsubscribe() error
}
```

**Implementation Options**

```
┌──────────────────────────────────────────────────────┐
│           Event Bus Implementations                  │
├──────────────────────────────────────────────────────┤
│                                                      │
│  1. In-Memory (Default)                              │
│     • Channels-based                                 │
│     • Fan-out to all subscribers                     │
│     • Fast, no persistence                           │
│     • Good for: single instance                      │
│                                                      │
│  2. NATS                                             │
│     • Distributed pub/sub                            │
│     • Persistence optional                           │
│     • Good for: multi-instance, cloud                │
│                                                      │
│  3. Redis Streams                                    │
│     • Persistent event log                           │
│     • Consumer groups                                │
│     • Good for: event replay, auditing               │
│                                                      │
│  4. Kafka                                            │
│     • High throughput                                │
│     • Event sourcing support                         │
│     • Good for: enterprise, analytics                │
└──────────────────────────────────────────────────────┘
```

---

## 6. Configuration & State Management

### 6.1 Configuration Architecture

**Layered Configuration**

```
┌──────────────────────────────────────────────────────┐
│           Configuration Priority Order               │
│                (highest to lowest)                   │
├──────────────────────────────────────────────────────┤
│                                                      │
│  1. Command-line flags                               │
│     --mode round-robin --max-turns 10                │
│                                                      │
│  2. Environment variables                            │
│     AGENTPIPE_MODE=round-robin                       │
│     AGENTPIPE_MAX_TURNS=10                           │
│                                                      │
│  3. Local config file                                │
│     ./agentpipe.yaml                                 │
│     ./.agentpipe.yaml                                │
│                                                      │
│  4. User config file                                 │
│     ~/.agentpipe/config.yaml                         │
│                                                      │
│  5. System config file                               │
│     /etc/agentpipe/config.yaml                       │
│                                                      │
│  6. Embedded defaults                                │
│     Built into binary                                │
└──────────────────────────────────────────────────────┘
```

**Configuration Schema**

```yaml
# agentpipe.yaml - Full configuration schema
version: "2.0"

# Core settings
core:
  mode: round-robin               # round-robin, reactive, free-form
  max_turns: 10
  turn_timeout: 30s
  response_delay: 1s

# Agent pool configuration
agents:
  - id: agent-1
    type: claude
    name: "Assistant"
    model: claude-sonnet-4-5
    enabled: true

    # Agent-specific config
    config:
      temperature: 0.7
      max_tokens: 2000

    # Resource limits
    limits:
      rate_limit: 10              # requests per second
      rate_limit_burst: 5
      timeout: 30s
      max_retries: 3

# Plugin configuration
plugins:
  enabled: true
  directories:
    - ~/.agentpipe/plugins
    - /usr/local/share/agentpipe/plugins

  # Plugin-specific settings
  config:
    storage:
      type: filesystem
      path: ~/.agentpipe/conversations

    exporter:
      default_format: markdown

    telemetry:
      prometheus:
        enabled: true
        port: 9090

# Event bus configuration
events:
  bus_type: memory                # memory, nats, redis, kafka

  # Bus-specific config
  config:
    buffer_size: 1000

# Storage configuration
storage:
  type: filesystem                # filesystem, postgres, s3

  # Type-specific config
  config:
    path: ~/.agentpipe/state
    auto_save: true
    compression: true

# Telemetry configuration
telemetry:
  # Logging
  logging:
    level: info                   # debug, info, warn, error
    format: json                  # json, text
    output: stderr                # stderr, stdout, file

  # Metrics
  metrics:
    enabled: true
    prometheus:
      enabled: true
      addr: :9090
      path: /metrics

  # Tracing
  tracing:
    enabled: false
    jaeger:
      endpoint: http://localhost:14268/api/traces

# External integrations
integrations:
  # Streaming bridge
  bridge:
    enabled: false
    url: https://agentpipe.ai
    api_key_env: AGENTPIPE_BRIDGE_API_KEY
    timeout: 10s
    retries: 3

  # Webhooks
  webhooks:
    - event: conversation.completed
      url: https://example.com/webhook
      secret_env: WEBHOOK_SECRET

# Security settings
security:
  plugins:
    require_signature: false
    trusted_keys: []

  rate_limiting:
    global_limit: 1000           # requests per hour
    per_agent_limit: 100
```

### 6.2 State Management

**State Model**

```go
// State represents the complete system state
type State struct {
    // Conversation state
    Conversations  map[ConversationID]*ConversationState

    // Agent state
    Agents         map[AgentID]*AgentState

    // Orchestration state
    Orchestrator   *OrchestratorState

    // Metadata
    Version        string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// ConversationState captures conversation snapshot
type ConversationState struct {
    ID             ConversationID
    Status         ConversationStatus
    Messages       []Message
    Participants   []AgentID
    CurrentTurn    int
    Metrics        AggregatedMetrics
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// AgentState captures agent runtime state
type AgentState struct {
    ID             AgentID
    HealthStatus   HealthStatus
    ActiveSince    time.Time
    LastActive     time.Time
    TotalRequests  int
    TotalErrors    int
    Metrics        AgentMetrics
}
```

**State Persistence**

```
┌──────────────────────────────────────────────────────┐
│              State Persistence Strategy              │
├──────────────────────────────────────────────────────┤
│                                                      │
│  1. Checkpoint Strategy                              │
│     • Periodic snapshots (e.g., every 5 turns)       │
│     • Triggered snapshots (e.g., on error)           │
│     • Allows conversation resumption                 │
│                                                      │
│  2. Event Sourcing (Optional)                        │
│     • Store all events in append-only log            │
│     • Rebuild state by replaying events              │
│     • Enables time-travel debugging                  │
│     • Full audit trail                               │
│                                                      │
│  3. Hybrid Approach                                  │
│     • Snapshots for current state                    │
│     • Events for audit trail                         │
│     • Best of both worlds                            │
└──────────────────────────────────────────────────────┘
```

---

## 7. Observability Architecture

### 7.1 Telemetry Pillars

```
┌──────────────────────────────────────────────────────┐
│              Three Pillars of Observability          │
├──────────────────────────────────────────────────────┤
│                                                      │
│  1. METRICS (What)                                   │
│     • Prometheus metrics                             │
│     • Request rates, durations, errors               │
│     • Token usage, costs                             │
│     • Agent health, availability                     │
│     • Conversation statistics                        │
│                                                      │
│  2. LOGS (Why)                                       │
│     • Structured logging (JSON)                      │
│     • Correlation IDs                                │
│     • Context-rich log entries                       │
│     • Different log levels                           │
│     • Searchable, filterable                         │
│                                                      │
│  3. TRACES (How)                                     │
│     • Distributed tracing (OpenTelemetry)            │
│     • Span per operation                             │
│     • Parent-child relationships                     │
│     • Latency breakdown                              │
│     • Service dependencies                           │
└──────────────────────────────────────────────────────┘
```

### 7.2 Metrics

**Metric Categories**

```go
// AgentMetrics - per-agent metrics
type AgentMetrics struct {
    // Counters
    RequestsTotal      prometheus.Counter
    ErrorsTotal        prometheus.Counter
    TokensInputTotal   prometheus.Counter
    TokensOutputTotal  prometheus.Counter

    // Gauges
    HealthStatus       prometheus.Gauge
    ActiveConversations prometheus.Gauge

    // Histograms
    RequestDuration    prometheus.Histogram
    TokensPerRequest   prometheus.Histogram
    CostPerRequest     prometheus.Histogram
}

// ConversationMetrics - per-conversation metrics
type ConversationMetrics struct {
    // Counters
    MessagesTotal      prometheus.Counter
    TurnsTotal         prometheus.Counter
    ErrorsTotal        prometheus.Counter

    // Gauges
    CurrentTurn        prometheus.Gauge
    ActiveParticipants prometheus.Gauge

    // Histograms
    TurnDuration       prometheus.Histogram
    MessagesPerTurn    prometheus.Histogram
}

// SystemMetrics - system-wide metrics
type SystemMetrics struct {
    // Counters
    ConversationsTotal     prometheus.Counter
    PluginsLoadedTotal     prometheus.Counter

    // Gauges
    ActiveAgents           prometheus.Gauge
    ActiveConversations    prometheus.Gauge
    MemoryUsageBytes       prometheus.Gauge
    GoRoutines             prometheus.Gauge

    // Histograms
    ConversationDuration   prometheus.Histogram
}
```

### 7.3 Structured Logging

**Log Format**

```json
{
  "timestamp": "2026-01-18T10:30:45.123Z",
  "level": "info",
  "component": "orchestrator",
  "conversation_id": "conv-abc123",
  "turn": 3,
  "agent_id": "claude-1",
  "agent_type": "claude",
  "message": "agent response received",
  "duration_ms": 2341,
  "tokens": {
    "input": 150,
    "output": 200,
    "total": 350
  },
  "cost_usd": 0.0042,
  "trace_id": "trace-xyz789",
  "span_id": "span-456"
}
```

### 7.4 Distributed Tracing

**Trace Hierarchy**

```
Conversation                                    [conversation.execute]
│
├─ Turn 1                                       [turn.execute]
│  ├─ Select Agent                              [orchestrator.select_agent]
│  ├─ Rate Limit Check                          [ratelimit.wait]
│  ├─ Send Message                              [agent.send_message]
│  │  ├─ Build Prompt                           [agent.build_prompt]
│  │  ├─ Execute CLI                            [adapter.execute_cli]
│  │  └─ Parse Response                         [adapter.parse_response]
│  ├─ Process Middleware                        [middleware.process]
│  │  ├─ Logging Middleware                     [middleware.logging]
│  │  ├─ Metrics Middleware                     [middleware.metrics]
│  │  └─ Content Filter                         [middleware.content_filter]
│  ├─ Publish Event                             [eventbus.publish]
│  └─ Save State                                [state.save]
│
├─ Turn 2                                       [turn.execute]
│  └─ ...
│
└─ Generate Summary                             [conversation.summarize]
   └─ Call Summarization Agent                  [agent.send_message]
```

---

## 8. Extensibility Points

### 8.1 Extension Interfaces

**Strategy Pattern Extensions**

```go
// Custom orchestration strategy
type CustomStrategy struct{}

func (s *CustomStrategy) SelectNext(ctx context.Context, state State) (AgentID, error) {
    // Custom selection logic
    // Example: AI-powered agent selection
    // Example: Priority-based selection
    // Example: Load-balanced selection
}

// Custom retry policy
type CustomRetryPolicy struct{}

func (p *CustomRetryPolicy) ShouldRetry(ctx context.Context, err error, attempt int) bool {
    // Custom retry logic
    // Example: Exponential backoff with jitter
    // Example: Different strategies per error type
}
```

**Middleware Extensions**

```go
// Custom middleware
type CustomMiddleware struct{}

func (m *CustomMiddleware) Process(ctx context.Context, msg Message, next Handler) (Message, error) {
    // Pre-processing
    // Example: Content transformation
    // Example: Sentiment analysis
    // Example: Language detection

    result, err := next(ctx, msg)

    // Post-processing
    // Example: Response validation
    // Example: Cost tracking

    return result, err
}
```

**Event Handler Extensions**

```go
// Custom event handler
type CustomEventHandler struct{}

func (h *CustomEventHandler) Handle(ctx context.Context, event Event) error {
    switch e := event.(type) {
    case *MessageCreated:
        // Custom logic for message events
        // Example: Real-time analytics
        // Example: Content moderation
        // Example: Notification triggers

    case *ConversationCompleted:
        // Custom logic for completion events
        // Example: Generate reports
        // Example: Archive conversations
    }

    return nil
}
```

### 8.2 Plugin Development Kit

**Scaffold Tool**

```bash
# Create new agent plugin
agentpipe plugin new --type agent --name my-agent

# Creates:
plugins/my-agent/
├── plugin.yaml           # Plugin manifest
├── main.go              # Plugin implementation
├── config.go            # Configuration
├── README.md            # Documentation
└── examples/
    └── config.yaml      # Example configuration
```

**Plugin Template**

```go
package main

import (
    "context"

    "github.com/ASRagab/agentpipe/pkg/plugin/api/v1"
    "github.com/ASRagab/agentpipe/pkg/core/agent"
)

// MyAgent implements the AgentPlugin interface
type MyAgent struct {
    api.BasePlugin
    agent.BaseAgent

    // Custom fields
    config MyAgentConfig
}

// Plugin lifecycle methods
func (a *MyAgent) Initialize(config api.Config) error {
    // Initialize plugin
    return nil
}

func (a *MyAgent) Start(ctx context.Context) error {
    // Start plugin
    return nil
}

func (a *MyAgent) Stop(ctx context.Context) error {
    // Stop plugin
    return nil
}

// Agent interface methods
func (a *MyAgent) SendMessage(ctx context.Context, req agent.MessageRequest) (agent.MessageResponse, error) {
    // Implement message sending
    return agent.MessageResponse{}, nil
}

// Plugin registration
func init() {
    api.RegisterPlugin(&MyAgent{})
}
```

---

## 9. Data Flow Diagrams

### 9.1 Conversation Flow

```
┌──────────────────────────────────────────────────────────────┐
│                  Conversation Execution Flow                 │
└──────────────────────────────────────────────────────────────┘

   User/CLI
      │
      ├─1─→ agentpipe run -c config.yaml
      │
      ↓
┌──────────────┐
│  CLI Handler │
└──────────────┘
      │
      ├─2─→ Load Config & Validate
      │
      ↓
┌────────────────────┐
│ Application Layer  │
│ (ConversationSvc)  │
└────────────────────┘
      │
      ├─3─→ Initialize Orchestrator
      ├─4─→ Load Agents (via Factory)
      ├─5─→ Setup Event Bus
      ├─6─→ Setup Telemetry
      │
      ↓
┌──────────────────┐
│  Orchestrator    │
└──────────────────┘
      │
      ├─7─→ Publish ConversationStarted Event
      │
      ↓
  ┌─────────┐
  │ Turn 1  │
  └─────────┘
      │
      ├─8──→ Select Agent (Strategy)
      ├─9──→ Check Rate Limit
      ├─10─→ Send Message to Agent
      │
      ↓
┌──────────────┐
│ Agent Adapter│
└──────────────┘
      │
      ├─11─→ Build Prompt (filter messages)
      ├─12─→ Execute CLI/API
      ├─13─→ Parse Response
      │
      ↓
┌──────────────────┐
│ Middleware Chain │
└──────────────────┘
      │
      ├─14─→ Logging Middleware
      ├─15─→ Metrics Middleware
      ├─16─→ Content Filter
      │
      ↓
┌────────────────┐
│  Event Bus     │
└────────────────┘
      │
      ├─17─→ Publish MessageCreated Event
      │       │
      │       ├──→ Bridge Emitter (stream to web)
      │       ├──→ Metrics Collector
      │       └──→ Logger
      │
      ↓
┌────────────────┐
│ State Manager  │
└────────────────┘
      │
      ├─18─→ Update Conversation State
      ├─19─→ Save Checkpoint (if enabled)
      │
      ↓
  ┌─────────┐
  │ Turn 2  │  (repeat 8-19)
  └─────────┘
      │
      ⋮
      │
      ↓
┌──────────────────┐
│  Conversation    │
│  Complete        │
└──────────────────┘
      │
      ├─20─→ Generate Summary (if enabled)
      ├─21─→ Publish ConversationCompleted Event
      ├─22─→ Save Final State
      ├─23─→ Cleanup Resources
      │
      ↓
   Display Results
```

### 9.2 Plugin Loading Flow

```
┌──────────────────────────────────────────────────────────────┐
│                   Plugin Loading Flow                        │
└──────────────────────────────────────────────────────────────┘

   Application Start
         │
         ├─1─→ Initialize Plugin System
         │
         ↓
┌───────────────────┐
│  Plugin Loader    │
└───────────────────┘
         │
         ├─2─→ Scan Plugin Directories
         │      ~/.agentpipe/plugins/
         │      /usr/local/share/agentpipe/plugins/
         │
         ↓
   ┌────────────────┐
   │ For each dir:  │
   └────────────────┘
         │
         ├─3─→ Read plugin.yaml
         │
         ↓
┌─────────────────────┐
│ Manifest Validator  │
└─────────────────────┘
         │
         ├─4─→ Check API version compatibility
         ├─5─→ Validate required fields
         ├─6─→ Check dependencies
         │
         ↓
┌──────────────────┐
│ Security Check   │
└──────────────────┘
         │
         ├─7─→ Verify checksum
         ├─8─→ Verify signature (if enabled)
         ├─9─→ Check permissions
         │
         ↓
┌─────────────────┐
│ Plugin Binary   │
│ Loader          │
└─────────────────┘
         │
         ├─10─→ Load plugin binary
         │       │
         │       ├─→ Go plugin: plugin.Open()
         │       ├─→ WASM: wasmer.NewInstance()
         │       └─→ External: start RPC server
         │
         ↓
┌────────────────┐
│ Plugin Init    │
└────────────────┘
         │
         ├─11─→ Call Initialize(config)
         ├─12─→ Resolve dependencies
         ├─13─→ Register with registry
         │
         ↓
┌─────────────────────┐
│ Service Registration│
└─────────────────────┘
         │
         ├─14─→ Register with appropriate service
         │       │
         │       ├─→ AgentPlugin → AgentFactory
         │       ├─→ StoragePlugin → StorageService
         │       └─→ etc.
         │
         ↓
   Plugin Ready
```

### 9.3 Event Propagation Flow

```
┌──────────────────────────────────────────────────────────────┐
│                  Event Propagation Flow                      │
└──────────────────────────────────────────────────────────────┘

   Event Producer
   (Orchestrator)
         │
         ├─1─→ Create Event
         │      MessageCreated {
         │        conversation_id,
         │        agent_id,
         │        content,
         │        metrics
         │      }
         │
         ↓
┌────────────────┐
│  Event Bus     │
└────────────────┘
         │
         ├─2─→ Route to subscribers
         │      (fan-out to all matching)
         │
         ├─────────────┬─────────────┬─────────────┐
         ↓             ↓             ↓             ↓
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│   Bridge    │ │   Metrics   │ │   Logger    │ │   Custom    │
│   Emitter   │ │  Collector  │ │             │ │   Handler   │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
      │               │               │               │
      ├─3a─→          ├─3b─→          ├─3c─→          ├─3d─→
      │               │               │               │
      ↓               ↓               ↓               ↓
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ HTTP POST   │ │ Prometheus  │ │ Structured  │ │   Custom    │
│ to Bridge   │ │ Metrics     │ │    Log      │ │   Logic     │
│   API       │ │   Update    │ │  (JSON)     │ │             │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
      │
      ├─4─→ Retry on failure (exponential backoff)
      │
      ↓
   External
   Service
```

---

## 10. Migration Strategy

### 10.1 Backward Compatibility

**Compatibility Matrix**

```
┌──────────────────────────────────────────────────────────────┐
│              v1 → v2 Compatibility Strategy                  │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Layer 1: CLI Compatibility                                  │
│    • All v1 commands work in v2                              │
│    • Flags mapped to new config system                       │
│    • Warnings for deprecated flags                           │
│    • Deprecation timeline: 6 months                          │
│                                                              │
│  Layer 2: Config Compatibility                               │
│    • Auto-migration of v1 YAML configs                       │
│    • agentpipe migrate config v1.yaml → v2.yaml              │
│    • Support both formats during transition                  │
│                                                              │
│  Layer 3: State Compatibility                                │
│    • v1 state files can be loaded in v2                      │
│    • Automatic schema upgrade on first load                  │
│    • Backup created before migration                         │
│                                                              │
│  Layer 4: Plugin Compatibility                               │
│    • v1 adapters wrapped as v2 plugins                       │
│    • Compatibility shim for old interfaces                   │
│    • Performance penalty noted in docs                       │
└──────────────────────────────────────────────────────────────┘
```

### 10.2 Migration Tool

```bash
# Check compatibility
agentpipe migrate check

# Migrate configuration
agentpipe migrate config --from v1.yaml --to v2.yaml

# Migrate state files
agentpipe migrate state --dir ~/.agentpipe/states

# Migrate custom adapters to plugins
agentpipe migrate adapter --input pkg/adapters/my-adapter.go \
                          --output plugins/my-adapter/

# Full migration (interactive)
agentpipe migrate --interactive
```

### 10.3 Phased Rollout

**Phase 1: Alpha (Months 1-2)**
- Core architecture implementation
- Plugin system foundation
- Event bus implementation
- Migration tools
- Limited beta testing

**Phase 2: Beta (Months 3-4)**
- Full feature parity with v1
- Comprehensive testing
- Documentation
- Migration guides
- Community feedback

**Phase 3: RC (Month 5)**
- Performance optimization
- Bug fixes
- Final API stabilization
- Release candidates

**Phase 4: GA (Month 6)**
- General availability
- v1 enters maintenance mode
- v2 becomes default
- 12-month v1 support commitment

---

## 11. Implementation Roadmap

### 11.1 Milestones

**M1: Foundation (Weeks 1-4)**
- [ ] Define core interfaces
- [ ] Implement domain models
- [ ] Setup project structure
- [ ] Plugin API v1 design
- [ ] Event bus implementation
- [ ] Configuration system

**M2: Core Components (Weeks 5-8)**
- [ ] Orchestrator refactor
- [ ] Agent pool implementation
- [ ] State management
- [ ] Middleware pipeline
- [ ] Plugin loader
- [ ] Plugin registry

**M3: Infrastructure (Weeks 9-12)**
- [ ] Agent adapter plugins
- [ ] Storage plugins
- [ ] Telemetry plugins
- [ ] Event bus plugins
- [ ] Migration tools
- [ ] Testing framework

**M4: Integration (Weeks 13-16)**
- [ ] CLI integration
- [ ] TUI integration
- [ ] HTTP API server
- [ ] Plugin SDK
- [ ] Documentation
- [ ] Examples

**M5: Testing & Polish (Weeks 17-20)**
- [ ] Unit tests (>90% coverage)
- [ ] Integration tests
- [ ] E2E tests
- [ ] Performance benchmarks
- [ ] Security audit
- [ ] User acceptance testing

**M6: Release (Weeks 21-24)**
- [ ] Release candidates
- [ ] Bug fixes
- [ ] Documentation finalization
- [ ] Migration guides
- [ ] Release automation
- [ ] v2.0.0 GA

### 11.2 Success Metrics

**Technical Metrics**
- Test coverage: >90%
- Plugin load time: <100ms
- Event latency: <10ms
- Memory overhead: <20% vs v1
- API response time: <50ms (p99)

**Quality Metrics**
- Zero critical bugs in RC
- <5 high-priority bugs in RC
- 100% API documentation coverage
- 100% migration tool coverage

**Adoption Metrics**
- 50% of users migrate within 3 months
- 90% of users migrate within 6 months
- 5+ community plugins within 6 months
- <10% rollback rate

---

## Appendices

### A. Component Diagram

```
                    ┌─────────────────────────────────────┐
                    │         AgentPipe v2                │
                    │                                     │
┌───────────────────┼─────────────────────────────────────┼───────────────────┐
│   Presentation    │                                     │                   │
│                   │  ┌─────┐  ┌─────┐  ┌──────┐       │                   │
│                   │  │ CLI │  │ TUI │  │ HTTP │       │                   │
│                   │  └─────┘  └─────┘  └──────┘       │                   │
└───────────────────┼─────────────────────────────────────┼───────────────────┘
                    │             ↓                       │
┌───────────────────┼─────────────────────────────────────┼───────────────────┐
│   Application     │  ┌──────────────────────────┐      │                   │
│                   │  │  Conversation Service    │      │                   │
│                   │  ├──────────────────────────┤      │                   │
│                   │  │  Orchestration Service   │      │                   │
│                   │  ├──────────────────────────┤      │                   │
│                   │  │  Agent Service           │      │                   │
│                   │  └──────────────────────────┘      │                   │
└───────────────────┼─────────────────────────────────────┼───────────────────┘
                    │             ↓                       │
┌───────────────────┼─────────────────────────────────────┼───────────────────┐
│   Domain          │  ┌───────┐ ┌──────────┐ ┌───────┐ │                   │
│                   │  │Agent  │ │Convers-  │ │Event  │ │                   │
│                   │  │Domain │ │ation     │ │Domain │ │                   │
│                   │  │       │ │Domain    │ │       │ │                   │
│                   │  └───────┘ └──────────┘ └───────┘ │                   │
└───────────────────┼─────────────────────────────────────┼───────────────────┘
                    │             ↓                       │
┌───────────────────┼─────────────────────────────────────┼───────────────────┐
│  Infrastructure   │  ┌──────────┐ ┌───────────┐        │                   │
│                   │  │Plugin    │ │Event Bus  │        │                   │
│                   │  │System    │ │           │        │                   │
│                   │  └──────────┘ └───────────┘        │                   │
│                   │       ↓              ↓              │                   │
│                   │  ┌──────────┐ ┌───────────┐        │                   │
│                   │  │Adapters  │ │Storage    │        │                   │
│                   │  │- CLI     │ │- File     │        │                   │
│                   │  │- API     │ │- DB       │        │                   │
│                   │  └──────────┘ └───────────┘        │                   │
└───────────────────┼─────────────────────────────────────┼───────────────────┘
                    │                                     │
                    └─────────────────────────────────────┘
```

### B. Technology Stack

**Core**
- Go 1.24+
- Domain-Driven Design patterns
- Hexagonal architecture

**Plugin System**
- Go plugin package (in-process)
- gRPC (out-of-process, optional)
- WebAssembly (future)

**Event Bus**
- In-memory channels (default)
- NATS (distributed, optional)
- Redis Streams (persistent, optional)

**Storage**
- Filesystem (default)
- PostgreSQL (optional)
- S3-compatible (optional)

**Telemetry**
- Prometheus (metrics)
- OpenTelemetry (traces)
- zerolog (structured logging)

**Testing**
- testify (assertions)
- gomock (mocking)
- httptest (HTTP testing)
- testcontainers (integration tests)

### C. Glossary

| Term | Definition |
|------|------------|
| **Adapter** | Infrastructure component that connects to external systems (CLIs, APIs) |
| **Aggregate** | Cluster of domain objects treated as a single unit |
| **Bounded Context** | Explicit boundary within which a domain model is defined |
| **CQRS** | Command Query Responsibility Segregation - separate read/write models |
| **DDD** | Domain-Driven Design - software design approach focused on the domain |
| **Event Sourcing** | Persisting state as a sequence of events |
| **Hexagonal Architecture** | Ports and adapters architecture pattern |
| **Plugin** | Modular component that extends core functionality |
| **Repository** | Abstraction for data access |
| **Value Object** | Immutable object with no identity, defined by its attributes |

---

## Conclusion

This architecture proposal for AgentPipe v2 provides a solid foundation for a modular, extensible, and maintainable system. The emphasis on domain-driven design, plugin architecture, and event-driven coordination ensures the system can evolve with changing requirements while maintaining stability and performance.

### Next Steps

1. **Review and Feedback**: Gather feedback from stakeholders and team members
2. **Proof of Concept**: Implement core components to validate architecture
3. **Detailed Design**: Create detailed design documents for each component
4. **Implementation**: Begin phased implementation following the roadmap
5. **Testing**: Continuous testing and validation throughout development
6. **Documentation**: Comprehensive documentation for developers and users

### Success Criteria

The v2 architecture will be considered successful if it achieves:
- **Modularity**: Clear separation of concerns with minimal coupling
- **Extensibility**: Easy to add new features via plugins
- **Testability**: >90% test coverage with comprehensive test strategies
- **Performance**: Equal or better performance than v1
- **Maintainability**: Easier to understand, modify, and extend than v1
- **Adoption**: Smooth migration path for existing users

---

**Document Version:** 1.0
**Last Updated:** 2026-01-18
**Maintained By:** System Architecture Team
