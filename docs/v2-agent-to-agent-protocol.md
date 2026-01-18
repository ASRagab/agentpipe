# AgentPipe v2: Agent-to-Agent Communication Protocol

**Version:** 2.0.0-alpha
**Date:** 2026-01-18
**Status:** Design Specification
**Author:** System Architecture Designer

---

## Executive Summary

This document specifies the agent-to-agent (A2A) communication protocol for AgentPipe v2, enabling agents to directly address, respond to, and collaborate with each other beyond simple turn-taking. The protocol supports natural multi-agent collaboration through message addressing, threading, inter-agent protocols, and dynamic conversation flow.

### Key Features
- **Direct Addressing**: Agents can target specific agents using @mentions
- **Message Threading**: Reply-to relationships create conversation threads
- **Inter-Agent Protocols**: Structured patterns (ask, answer, share, challenge, vote)
- **Context Awareness**: All agents see full conversation history with attribution
- **Role-Based Behavior**: Facilitator, specialist, critic, synthesizer roles
- **Turn Yielding**: Agents can explicitly pass control or request responses

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Message Format Specification](#2-message-format-specification)
3. [Addressing and Routing](#3-addressing-and-routing)
4. [Inter-Agent Interaction Patterns](#4-inter-agent-interaction-patterns)
5. [Conversation Threading Model](#5-conversation-threading-model)
6. [Agent Roles and Behaviors](#6-agent-roles-and-behaviors)
7. [Turn Management and Control Flow](#7-turn-management-and-control-flow)
8. [Context Preservation and History](#8-context-preservation-and-history)
9. [Example Multi-Agent Conversations](#9-example-multi-agent-conversations)
10. [Implementation Pseudocode](#10-implementation-pseudocode)

---

## 1. Architecture Overview

### 1.1 Communication Model

```
┌──────────────────────────────────────────────────────────┐
│           Agent-to-Agent Communication Flow              │
└──────────────────────────────────────────────────────────┘

┌──────────┐                                    ┌──────────┐
│  Agent A │                                    │  Agent B │
└─────┬────┘                                    └────┬─────┘
      │                                              │
      │ 1. Creates message with @AgentB              │
      ├─────────────────────────────────────────────►│
      │                                              │
      │              ┌─────────────┐                 │
      │              │ Orchestrator│                 │
      │              │   Message   │                 │
      │              │   Router    │                 │
      │              └──────┬──────┘                 │
      │                     │                        │
      │ 2. Routes based on @mention                  │
      │                     │                        │
      │                     ▼                        │
      │              ┌─────────────┐                 │
      │              │  Context    │                 │
      │              │  Builder    │                 │
      │              │             │                 │
      │              │ • Thread    │                 │
      │              │ • History   │                 │
      │              │ • Mentions  │                 │
      │              └──────┬──────┘                 │
      │                     │                        │
      │ 3. Agent B receives contextualized message   │
      │                     ├─────────────────────────►
      │                     │                        │
      │                4. Response with reply-to     │
      │◄────────────────────┴────────────────────────┤
      │                                              │
┌─────┴────┐                                    ┌────┴─────┐
│  Agent A │                                    │  Agent B │
└──────────┘                                    └──────────┘
```

### 1.2 Core Concepts

**Message Addressing**
- Agents can explicitly address other agents using @mentions
- Multiple agents can be mentioned in a single message
- Addressing affects routing priority and context filtering

**Message Threading**
- Every message can have a `reply_to` field referencing a parent message
- Creates conversation threads alongside main flow
- Enables focused sub-conversations without cluttering main thread

**Inter-Agent Protocols**
- Structured interaction patterns (ask, answer, challenge, etc.)
- Protocol markers in message metadata guide agent behavior
- Enables coordination without central control

**Context Awareness**
- All agents see conversation history with full attribution
- Thread-aware context filtering shows relevant sub-conversations
- Mention-aware context highlights messages requiring response

---

## 2. Message Format Specification

### 2.1 Extended Message Structure

```go
// Message represents agent-to-agent communication
type Message struct {
    // Core Identity (existing)
    MessageID   MessageID  `json:"message_id"`
    AgentID     AgentID    `json:"agent_id"`
    AgentName   string     `json:"agent_name"`
    AgentType   string     `json:"agent_type"`
    Content     string     `json:"content"`
    Timestamp   int64      `json:"timestamp"`
    Role        string     `json:"role"` // "agent", "user", "system"

    // A2A Communication Fields (new)
    Addressing  *AddressingInfo  `json:"addressing,omitempty"`
    Threading   *ThreadingInfo   `json:"threading,omitempty"`
    Protocol    *ProtocolInfo    `json:"protocol,omitempty"`
    Control     *ControlInfo     `json:"control,omitempty"`

    // Performance Metrics (existing)
    Metrics     *ResponseMetrics `json:"metrics,omitempty"`
}

// AddressingInfo defines message targeting
type AddressingInfo struct {
    // To specifies primary recipients (explicit @mentions)
    To          []AgentID        `json:"to,omitempty"`

    // Cc specifies secondary recipients (observing but not required to respond)
    Cc          []AgentID        `json:"cc,omitempty"`

    // Broadcast indicates message is for all agents
    Broadcast   bool             `json:"broadcast"`

    // ExcludeFrom specifies agents to exclude from broadcast
    ExcludeFrom []AgentID        `json:"exclude_from,omitempty"`

    // Priority affects routing and agent attention
    Priority    MessagePriority  `json:"priority"` // high, normal, low
}

// ThreadingInfo defines conversation threading
type ThreadingInfo struct {
    // ReplyTo references the parent message
    ReplyTo     *MessageID       `json:"reply_to,omitempty"`

    // ThreadID groups related messages
    ThreadID    ThreadID         `json:"thread_id"`

    // ThreadDepth indicates nesting level (0 = main thread)
    ThreadDepth int              `json:"thread_depth"`

    // ThreadPosition is the index within thread
    ThreadPosition int           `json:"thread_position"`
}

// ProtocolInfo defines interaction pattern
type ProtocolInfo struct {
    // Type defines the interaction pattern
    Type        ProtocolType     `json:"type"` // ask, answer, share, challenge, vote, synthesize

    // RequestID links protocol request/response pairs
    RequestID   *string          `json:"request_id,omitempty"`

    // Metadata for protocol-specific data
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ControlInfo defines turn management
type ControlInfo struct {
    // YieldTo explicitly passes control to another agent
    YieldTo     *AgentID         `json:"yield_to,omitempty"`

    // RequestTurn indicates agent wants to speak next
    RequestTurn bool             `json:"request_turn"`

    // ExpectedResponseTime hints at urgency
    ExpectedResponseTime *time.Duration `json:"expected_response_time,omitempty"`

    // RequiresAck indicates acknowledgment is required
    RequiresAck bool             `json:"requires_ack"`
}

// MessagePriority affects message routing
type MessagePriority string

const (
    PriorityHigh   MessagePriority = "high"
    PriorityNormal MessagePriority = "normal"
    PriorityLow    MessagePriority = "low"
)

// ProtocolType defines interaction patterns
type ProtocolType string

const (
    ProtocolAsk        ProtocolType = "ask"        // Request information/action
    ProtocolAnswer     ProtocolType = "answer"     // Respond to ask
    ProtocolShare      ProtocolType = "share"      // Share findings/results
    ProtocolChallenge  ProtocolType = "challenge"  // Question/disagree
    ProtocolVote       ProtocolType = "vote"       // Cast vote on proposal
    ProtocolSynthesize ProtocolType = "synthesize" // Combine multiple inputs
    ProtocolAck        ProtocolType = "ack"        // Acknowledge receipt
)
```

### 2.2 Message Examples

**Simple Direct Message**
```json
{
  "message_id": "msg-001",
  "agent_id": "agent-a",
  "agent_name": "ResearcherAgent",
  "content": "@CoderAgent Based on my analysis, we should use PostgreSQL for persistence.",
  "timestamp": 1705583445,
  "role": "agent",
  "addressing": {
    "to": ["agent-coder"],
    "priority": "normal"
  },
  "protocol": {
    "type": "share",
    "metadata": {
      "topic": "database_selection"
    }
  }
}
```

**Threaded Reply**
```json
{
  "message_id": "msg-002",
  "agent_id": "agent-coder",
  "agent_name": "CoderAgent",
  "content": "@ResearcherAgent Agreed. Should we also add Redis for caching?",
  "timestamp": 1705583450,
  "role": "agent",
  "addressing": {
    "to": ["agent-researcher"],
    "priority": "normal"
  },
  "threading": {
    "reply_to": "msg-001",
    "thread_id": "thread-database",
    "thread_depth": 1,
    "thread_position": 0
  },
  "protocol": {
    "type": "ask",
    "request_id": "req-redis-caching"
  }
}
```

**Broadcast with Protocol**
```json
{
  "message_id": "msg-003",
  "agent_id": "agent-facilitator",
  "agent_name": "FacilitatorAgent",
  "content": "Team: Should we proceed with the proposed architecture? Please vote.",
  "timestamp": 1705583500,
  "role": "agent",
  "addressing": {
    "broadcast": true,
    "priority": "high"
  },
  "protocol": {
    "type": "vote",
    "request_id": "vote-architecture-001",
    "metadata": {
      "options": ["approve", "reject", "defer"],
      "deadline": 1705583600
    }
  },
  "control": {
    "requires_ack": true
  }
}
```

---

## 3. Addressing and Routing

### 3.1 Addressing Syntax

**In-Content Mentions (@-syntax)**
```
@AgentName        - Direct mention (parsed from content)
@all              - Broadcast to all agents
@role:coder       - Target agents by role
@type:claude      - Target agents by type
```

**Structured Addressing (metadata)**
```go
addressing := &AddressingInfo{
    To: []AgentID{"agent-1", "agent-2"},  // Primary recipients
    Cc: []AgentID{"agent-3"},             // Observers
    Priority: PriorityHigh,               // Routing priority
}
```

### 3.2 Routing Rules

```
┌─────────────────────────────────────────────────────────┐
│              Message Routing Algorithm                  │
└─────────────────────────────────────────────────────────┘

1. Parse Message Addressing
   ├─ Extract @mentions from content
   ├─ Read addressing metadata
   └─ Determine recipients

2. Build Recipient List
   ├─ If addressing.To is set → use explicit list
   ├─ If addressing.Broadcast → all active agents
   ├─ If @mentions found → parse and resolve
   └─ Else → use orchestrator mode (round-robin, etc.)

3. Filter Recipients
   ├─ Remove sender (agent can't message itself)
   ├─ Remove excluded agents (addressing.ExcludeFrom)
   ├─ Apply role/type filters if specified
   └─ Check agent availability/health

4. Prioritize Delivery
   ├─ High priority → interrupt current turn
   ├─ Normal priority → queue for next turn
   └─ Low priority → opportunistic delivery

5. Deliver to Recipients
   ├─ For each recipient:
   │  ├─ Build contextualized message history
   │  ├─ Filter for relevant thread
   │  ├─ Highlight mentions of this agent
   │  └─ Invoke agent.SendMessage()
   └─ Track delivery status
```

### 3.3 Context Filtering

Each agent receives a filtered view of conversation history:

```go
// ContextFilter determines which messages an agent sees
type ContextFilter struct {
    // IncludeThreads - IDs of threads relevant to this agent
    IncludeThreads  []ThreadID

    // IncludeMentions - Messages that mention this agent
    IncludeMentions bool

    // IncludeDirected - Messages explicitly addressed to this agent
    IncludeDirected bool

    // IncludeBroadcast - Broadcast messages
    IncludeBroadcast bool

    // IncludeRole - Messages from/to agents with specific role
    IncludeRole     *AgentRole

    // MaxDepth - Maximum thread depth to include
    MaxDepth        int

    // MaxMessages - Maximum number of messages in context
    MaxMessages     int
}

// BuildContext creates filtered message history for agent
func BuildContext(agent Agent, allMessages []Message) []Message {
    filter := &ContextFilter{
        IncludeMentions:  true,
        IncludeDirected:  true,
        IncludeBroadcast: true,
        MaxDepth:         3,     // Max 3 levels of threading
        MaxMessages:      50,    // Last 50 relevant messages
    }

    filtered := []Message{}

    for _, msg := range allMessages {
        if shouldInclude(msg, agent, filter) {
            filtered = append(filtered, msg)
        }
    }

    return filtered
}
```

---

## 4. Inter-Agent Interaction Patterns

### 4.1 Protocol Types

**ASK Protocol** - Request information or action
```
Initiator: @TargetAgent Can you analyze the performance metrics?
           [Protocol: ask, RequestID: req-perf-001]

Target:    @Initiator Yes, analyzing now. I'll share results in 2 minutes.
           [Protocol: ack, RequestID: req-perf-001]

Target:    @Initiator Performance analysis complete: 95th percentile latency is 250ms.
           [Protocol: answer, RequestID: req-perf-001]
```

**SHARE Protocol** - Share findings/results
```
Agent:     @all I've completed the security audit. Found 3 medium-priority issues.
           [Protocol: share, Topic: security_audit]
```

**CHALLENGE Protocol** - Question or disagree
```
Critic:    @Architect I disagree with using microservices here. The complexity isn't justified.
           [Protocol: challenge, Topic: architecture_decision]

Architect: @Critic Valid concern. Let me explain the scalability requirements...
           [Protocol: answer, RequestID: challenge-microservices]
```

**VOTE Protocol** - Consensus building
```
Facilitator: @all Should we proceed with Option A or Option B?
             [Protocol: vote, RequestID: vote-001, Options: [A, B]]

Agent1:      @Facilitator My vote: Option A
             [Protocol: vote, RequestID: vote-001, Vote: A]

Agent2:      @Facilitator My vote: Option B
             [Protocol: vote, RequestID: vote-001, Vote: B]
```

**SYNTHESIZE Protocol** - Combine inputs
```
Synthesizer: Based on @ResearcherAgent's findings and @CoderAgent's implementation,
             I recommend hybrid approach combining both suggestions.
             [Protocol: synthesize, Sources: [msg-010, msg-015]]
```

### 4.2 Protocol State Machines

**Ask-Answer Flow**
```
┌─────────────────────────────────────────┐
│         ASK Protocol Flow               │
└─────────────────────────────────────────┘

[Idle] ─── ASK ────► [Waiting for Response]
                             │
                             ├─ ACK ──► [Acknowledged]
                             │              │
                             │              ├─ ANSWER ──► [Completed]
                             │              └─ TIMEOUT ─► [Failed]
                             │
                             └─ ANSWER ────► [Completed]
                             └─ TIMEOUT ───► [Failed]
```

**Vote Flow**
```
┌─────────────────────────────────────────┐
│         VOTE Protocol Flow              │
└─────────────────────────────────────────┘

[Idle] ─── VOTE REQUEST ────► [Collecting Votes]
                                     │
                                     ├─ Vote 1
                                     ├─ Vote 2
                                     ├─ ...
                                     ├─ Vote N
                                     │
                                     ├─ All votes ────► [Tally]
                                     └─ Timeout ──────► [Partial Tally]
                                                           │
                                                           ▼
                                                    [Result Published]
```

### 4.3 Protocol Composition

Protocols can be combined:

```
Message 1: ASK + YieldTo
  "Can you review this code? I'll wait for your feedback before proceeding."

Message 2: SHARE + RequestTurn
  "I found a critical bug. I need to explain immediately."

Message 3: SYNTHESIZE + Vote
  "Based on all inputs, here's my summary. Do we agree?"
```

---

## 5. Conversation Threading Model

### 5.1 Thread Hierarchy

```
Main Thread (ThreadID: main, Depth: 0)
│
├─ Message 1 (Host: "Welcome to the brainstorm")
│
├─ Message 2 (Agent A: "I suggest we focus on scalability")
│
├─ Message 3 (Agent B: "Agreed. What about security?")
│  │
│  └─ Thread: security-discussion (Depth: 1)
│     │
│     ├─ Message 4 (Agent C: "@AgentB We should use OAuth2")
│     │  │
│     │  └─ Thread: oauth-details (Depth: 2)
│     │     │
│     │     ├─ Message 5 (Agent B: "@AgentC Which OAuth flow?")
│     │     └─ Message 6 (Agent C: "@AgentB Authorization Code flow")
│     │
│     └─ Message 7 (Agent A: "Don't forget about rate limiting")
│
└─ Message 8 (Agent D: "Back to scalability: horizontal or vertical?")
   │
   └─ Thread: scaling-strategy (Depth: 1)
      │
      ├─ Message 9 (Agent A: "@AgentD Horizontal with k8s")
      └─ Message 10 (Agent B: "Agreed with @AgentA")
```

### 5.2 Thread Management

```go
// ThreadManager handles conversation threading
type ThreadManager struct {
    threads map[ThreadID]*Thread
}

// Thread represents a conversation thread
type Thread struct {
    ID          ThreadID
    ParentID    *ThreadID
    Depth       int
    Messages    []MessageID
    Participants []AgentID
    Status      ThreadStatus  // active, resolved, merged
    CreatedAt   time.Time
}

// StartThread creates a new thread from a message
func (tm *ThreadManager) StartThread(parentMsg *Message) ThreadID {
    threadID := generateThreadID(parentMsg)

    thread := &Thread{
        ID:       threadID,
        ParentID: getThreadID(parentMsg),
        Depth:    parentMsg.Threading.ThreadDepth + 1,
        Messages: []MessageID{parentMsg.MessageID},
        Participants: []AgentID{parentMsg.AgentID},
        Status:   ThreadStatusActive,
        CreatedAt: time.Now(),
    }

    tm.threads[threadID] = thread
    return threadID
}

// AddToThread adds a message to existing thread
func (tm *ThreadManager) AddToThread(threadID ThreadID, msg Message) {
    thread := tm.threads[threadID]
    thread.Messages = append(thread.Messages, msg.MessageID)

    // Track unique participants
    if !contains(thread.Participants, msg.AgentID) {
        thread.Participants = append(thread.Participants, msg.AgentID)
    }
}

// ResolveThread marks thread as resolved
func (tm *ThreadManager) ResolveThread(threadID ThreadID, summary string) {
    thread := tm.threads[threadID]
    thread.Status = ThreadStatusResolved

    // Post resolution summary to main thread
    postSummary(summary, thread.Participants)
}
```

### 5.3 Thread Visualization

For TUI/UI display:

```
┌───────────────────────────────────────────────────────────┐
│  Main Conversation                                        │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  [Alice] Let's discuss the architecture                   │
│  [Bob]   I think we should use microservices              │
│  [Carol] @Bob What about monolith-first approach?         │
│    ╰─┬─ [Thread: monolith-vs-microservices] (3 msgs)     │
│      │                                                    │
│  [Dave] Don't forget about database selection             │
│    ╰─┬─ [Thread: database] (5 msgs, Active)              │
│      │                                                    │
│  [Alice] @all Let's vote on the architecture              │
│  [Bob]   My vote: microservices                           │
│  [Carol] My vote: monolith-first                          │
│  [Dave]  My vote: microservices                           │
│                                                           │
│  [Alice] Decision: Microservices (2-1 vote)               │
│                                                           │
└───────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────┐
│  Thread: monolith-vs-microservices                        │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  [Carol] @Bob What about monolith-first approach?         │
│  [Bob]   @Carol Valid, but we need to scale early         │
│  [Carol] @Bob Fair point. Let's put it to a vote.         │
│                                                           │
│  Status: Resolved (merged to main)                        │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

---

## 6. Agent Roles and Behaviors

### 6.1 Role System

```go
// AgentRole defines behavioral patterns
type AgentRole string

const (
    RoleFacilitator AgentRole = "facilitator"  // Guides conversation
    RoleSpecialist  AgentRole = "specialist"   // Domain expert
    RoleCritic      AgentRole = "critic"       // Questions/validates
    RoleSynthesizer AgentRole = "synthesizer"  // Combines inputs
    RoleParticipant AgentRole = "participant"  // General contributor
)

// RoleBehavior defines role-specific behavior patterns
type RoleBehavior struct {
    Role                AgentRole

    // Message patterns
    PreferredProtocols  []ProtocolType
    InitiateFrequency   float64  // 0.0-1.0 (how often to initiate)
    ResponsePriority    float64  // 0.0-1.0 (priority for responding)

    // Turn management
    YieldToRoles        []AgentRole  // Prefer yielding to these roles
    InterruptThreshold  MessagePriority  // When to interrupt

    // Threading behavior
    CreateThreads       bool  // Comfortable creating threads
    MergeThreads        bool  // Responsible for merging
}
```

### 6.2 Role Behaviors

**Facilitator Role**
```yaml
role: facilitator
behaviors:
  - Initiates conversation with prompts/questions
  - Monitors participation balance
  - Calls for votes and consensus
  - Synthesizes group decisions
  - Manages time and focus
  - Resolves conflicts

message_patterns:
  - "Let's discuss..."
  - "What does everyone think about..."
  - "@all Please vote on..."
  - "Let's move on to..."
  - "To summarize what we've discussed..."

protocols:
  - vote (initiate)
  - synthesize (perform)
  - ask (to guide)

control:
  - high initiate_frequency
  - yields to specialists
  - interrupts on deadlock
```

**Specialist Role**
```yaml
role: specialist
behaviors:
  - Responds to domain-specific questions
  - Shares expert knowledge
  - Validates technical decisions
  - Provides detailed analysis
  - Challenges incorrect assumptions

message_patterns:
  - "Based on my analysis..."
  - "From a [domain] perspective..."
  - "@agent That won't work because..."
  - "Here are the technical constraints..."

protocols:
  - answer (to questions)
  - share (findings)
  - challenge (incorrect info)

control:
  - responds when addressed
  - responds to domain keywords
  - high response_priority in specialty
```

**Critic Role**
```yaml
role: critic
behaviors:
  - Questions assumptions
  - Identifies risks and weaknesses
  - Plays devil's advocate
  - Validates proposals
  - Ensures thorough analysis

message_patterns:
  - "What about edge case..."
  - "I'm concerned about..."
  - "Have we considered..."
  - "That assumption might not hold if..."

protocols:
  - challenge (frequently)
  - ask (probing questions)
  - vote (after thorough review)

control:
  - interrupts for critical issues
  - responds to proposals
  - creates threads for concerns
```

**Synthesizer Role**
```yaml
role: synthesizer
behaviors:
  - Combines multiple viewpoints
  - Identifies common themes
  - Resolves contradictions
  - Creates summaries
  - Proposes compromises

message_patterns:
  - "Combining @AgentA's point with @AgentB's..."
  - "The common thread I see is..."
  - "Here's a compromise that addresses both concerns..."
  - "To summarize the discussion..."

protocols:
  - synthesize (primary)
  - share (summaries)
  - vote (after synthesis)

control:
  - waits for sufficient input
  - merges threads
  - summarizes before decisions
```

### 6.3 Dynamic Role Assignment

Agents can have multiple roles or switch roles:

```go
// AgentRoleConfig defines agent role behavior
type AgentRoleConfig struct {
    PrimaryRole   AgentRole
    SecondaryRoles []AgentRole

    // Dynamic role switching
    CanSwitchRoles bool
    RoleSwitchTriggers map[string]AgentRole  // context → role
}

// Example: Agent that switches based on context
config := AgentRoleConfig{
    PrimaryRole: RoleSpecialist,
    SecondaryRoles: []AgentRole{RoleCritic},
    CanSwitchRoles: true,
    RoleSwitchTriggers: map[string]AgentRole{
        "voting":     RoleParticipant,
        "reviewing":  RoleCritic,
        "proposing":  RoleSpecialist,
    },
}
```

---

## 7. Turn Management and Control Flow

### 7.1 Turn Yielding

Agents can explicitly pass control:

```go
// YieldControl passes turn to specific agent or role
func (a *Agent) YieldControl(to AgentID, reason string) Message {
    return Message{
        AgentID: a.ID,
        Content: fmt.Sprintf("@%s %s (yielding to you)", to, reason),
        Control: &ControlInfo{
            YieldTo: &to,
        },
    }
}

// Example usage
msg := agent.YieldControl("security-expert",
    "This is outside my expertise")
```

### 7.2 Turn Requesting

Agents can request turns:

```go
// RequestTurn signals desire to speak
func (a *Agent) RequestTurn(priority MessagePriority, reason string) Message {
    return Message{
        AgentID: a.ID,
        Content: fmt.Sprintf("[Hand raised: %s]", reason),
        Control: &ControlInfo{
            RequestTurn: true,
        },
        Addressing: &AddressingInfo{
            Priority: priority,
        },
    }
}

// Example
msg := agent.RequestTurn(PriorityHigh,
    "Critical security issue found")
```

### 7.3 Interrupt Handling

High-priority messages can interrupt:

```
┌───────────────────────────────────────────────────────┐
│            Interrupt Handling Flow                    │
└───────────────────────────────────────────────────────┘

Normal Flow:
  Turn 1: Agent A speaks
  Turn 2: Agent B speaks (scheduled)
  Turn 3: Agent C speaks (scheduled)

Interrupt:
  Turn 1: Agent A speaks
  -------- Agent D sends high-priority message --------
  Turn 2: Agent D speaks (INTERRUPTS)
  Turn 3: Agent B speaks (deferred)
  Turn 4: Agent C speaks (deferred)

Orchestrator Logic:
  1. Receive high-priority message
  2. Pause current turn queue
  3. Insert interrupting agent at front
  4. Resume queue after interrupt
```

### 7.4 Conversation Control Patterns

**Facilitator-Led**
```
Facilitator: "Let's discuss X. @SpecialistA, what's your take?"
SpecialistA: "@Facilitator Here's my analysis..."
Facilitator: "Good. @SpecialistB, do you agree?"
SpecialistB: "@Facilitator I have a different view..."
Facilitator: "@all Let's vote: Option A or Option B?"
[Voting occurs]
Facilitator: "Decision made. Moving to next topic."
```

**Peer-to-Peer**
```
Agent A: "@AgentB What do you think about using Kafka?"
Agent B: "@AgentA Good idea, but have you considered NATS?"
Agent C: "@AgentA @AgentB I've used both. NATS is simpler."
Agent A: "@AgentC What about persistence?"
Agent C: "@AgentA NATS has JetStream for that."
Agent B: "@AgentC Good point. Let's go with NATS."
```

**Challenge-Response**
```
Proposer: "I propose we use NoSQL for all data."
Critic:   "@Proposer That won't work for transactions. We need ACID."
Proposer: "@Critic We can use distributed transactions."
Critic:   "@Proposer Too complex and error-prone. What about CQRS?"
Expert:   "@Critic @Proposer CQRS is good. Use SQL for writes, NoSQL for reads."
[Consensus reached]
```

---

## 8. Context Preservation and History

### 8.1 Conversation State

```go
// ConversationState captures full A2A context
type ConversationState struct {
    // Message history
    Messages        []Message
    MessageIndex    map[MessageID]*Message

    // Threading
    Threads         map[ThreadID]*Thread
    MainThread      ThreadID

    // Participants
    Agents          map[AgentID]*AgentState

    // Active protocols
    ActiveProtocols map[string]*ProtocolState

    // Turn management
    TurnQueue       []AgentID
    CurrentTurn     *AgentID
    TurnHistory     []TurnRecord
}

// AgentState tracks per-agent context
type AgentState struct {
    AgentID         AgentID
    Role            AgentRole

    // Message tracking
    MessagesSent    int
    MessagesReceived int
    LastActive      time.Time

    // Addressing
    MentionedBy     []AgentID  // Who has mentioned this agent
    MentionedTo     []AgentID  // Who this agent has mentioned

    // Protocols
    ActiveRequests  []string   // Protocol requests awaiting response
    PendingVotes    []string   // Votes this agent needs to cast
}

// ProtocolState tracks protocol execution
type ProtocolState struct {
    RequestID       string
    Type            ProtocolType
    Initiator       AgentID
    Participants    []AgentID

    Status          ProtocolStatus  // initiated, in_progress, completed, timeout
    Responses       map[AgentID]interface{}

    CreatedAt       time.Time
    Deadline        *time.Time
}
```

### 8.2 History Filtering

Each agent gets customized history:

```go
// GetAgentContext builds filtered history for agent
func GetAgentContext(agent Agent, state ConversationState, opts ContextOptions) []Message {
    messages := []Message{}

    for _, msg := range state.Messages {
        if shouldIncludeForAgent(msg, agent, opts) {
            // Optionally highlight mentions
            if opts.HighlightMentions && isMentioned(agent, msg) {
                msg.Content = highlightMention(msg.Content, agent.Name)
            }
            messages = append(messages, msg)
        }
    }

    // Apply limits
    if len(messages) > opts.MaxMessages {
        messages = messages[len(messages)-opts.MaxMessages:]
    }

    return messages
}

// shouldIncludeForAgent determines message relevance
func shouldIncludeForAgent(msg Message, agent Agent, opts ContextOptions) bool {
    // Always include system messages
    if msg.Role == "system" {
        return true
    }

    // Always include messages from this agent
    if msg.AgentID == agent.ID {
        return true
    }

    // Include messages addressing this agent
    if msg.Addressing != nil {
        if contains(msg.Addressing.To, agent.ID) {
            return true
        }
        if contains(msg.Addressing.Cc, agent.ID) {
            return true
        }
        if msg.Addressing.Broadcast {
            return true
        }
    }

    // Include messages mentioning this agent
    if opts.IncludeMentions && isMentioned(agent, msg) {
        return true
    }

    // Include messages in same thread
    if opts.IncludeThreadContext && inSameThread(msg, opts.CurrentThread) {
        return true
    }

    // Include messages from agents with same role
    if opts.IncludeRoleContext && hasSameRole(agent, msg) {
        return true
    }

    return false
}
```

### 8.3 Context Injection

System prompts inject A2A awareness:

```
You are participating in a multi-agent conversation. You can:
- Address other agents using @AgentName
- Reply to messages to create threads
- Use protocols: ask, answer, share, challenge, vote
- Request turns with high-priority messages
- Yield control with @AgentName (yielding to you)

Current participants:
- @ResearcherAgent (specialist in requirements)
- @CoderAgent (specialist in implementation)
- @ReviewerAgent (critic focused on quality)
- @FacilitatorAgent (guides discussion)

Your role: Specialist
Your focus: Security architecture

Recent mentions of you:
- [Turn 5] @CoderAgent: "@SecurityExpert what about OAuth?"
- [Turn 8] @Facilitator: "@all including @SecurityExpert please vote"

Active threads:
- Thread "oauth-discussion": 3 messages, waiting for your input
- Thread "database-security": 5 messages, resolved

You can respond naturally or use structured protocols.
```

---

## 9. Example Multi-Agent Conversations

### 9.1 Example 1: Technical Architecture Discussion

```
┌────────────────────────────────────────────────────────────┐
│  Conversation: API Design Review                          │
│  Participants: Facilitator, Architect, SecurityExpert,    │
│                PerformanceEngineer, DeveloperRep           │
└────────────────────────────────────────────────────────────┘

[Turn 1 - Facilitator]
"Welcome everyone. Today we're reviewing the proposed API architecture.
@Architect, please present your design."
[Protocol: ask, To: Architect]

[Turn 2 - Architect]
"@Facilitator Thanks. I propose a REST API with these endpoints:
- POST /api/users (create)
- GET /api/users/:id (read)
- PUT /api/users/:id (update)
- DELETE /api/users/:id (delete)

We'll use JWT for authentication and rate limiting via nginx."
[Protocol: share, Topic: api_design]

[Turn 3 - SecurityExpert]
"@Architect Question about JWT: where are you storing the secret?
Also, do you have token refresh implemented?"
[Protocol: ask, RequestID: req-jwt-001, To: Architect]
[Threading: creates "jwt-security" thread]

[Turn 4 - PerformanceEngineer]
"@Architect Have you considered API versioning? We should plan for /v1/ prefix.
Also, what's the expected RPS for rate limiting?"
[Protocol: ask, RequestID: req-performance-001, To: Architect]
[Threading: creates "performance-concerns" thread]

[Turn 5 - Architect]
"@SecurityExpert JWT secret in environment variables, rotated monthly.
Yes, refresh tokens with 7-day expiry."
[Protocol: answer, RequestID: req-jwt-001]
[Threading: reply to "jwt-security"]

[Turn 6 - Architect]
"@PerformanceEngineer Good point on versioning. I'll add /v1/ prefix.
Rate limit: 100 req/minute per user, 1000 req/minute per IP."
[Protocol: answer, RequestID: req-performance-001]
[Threading: reply to "performance-concerns"]

[Turn 7 - DeveloperRep]
"@Architect The API looks clean. One concern: DELETE operation lacks soft-delete.
We should keep audit trail. Can we use PATCH with 'deleted: true' instead?"
[Protocol: challenge, Topic: deletion_strategy]

[Turn 8 - SecurityExpert]
"@DeveloperRep Excellent point. We need audit trails for compliance.
I support soft-delete approach."
[Protocol: share, Topic: deletion_strategy]
[Threading: reply to DeveloperRep]

[Turn 9 - Architect]
"@DeveloperRep @SecurityExpert Agreed. I'll update:
- DELETE → PATCH /api/users/:id {deleted: true, deleted_at: timestamp}
- Add GET /api/users/:id/history for audit trail"
[Protocol: ack + share, Topic: deletion_strategy]

[Turn 10 - Facilitator]
"Great discussion. I see we've addressed security, performance, and auditing.
@all Should we approve this updated design? Please vote: approve or defer."
[Protocol: vote, RequestID: vote-api-design-001, Options: [approve, defer]]
[Control: requires_ack]

[Turn 11 - SecurityExpert]
"@Facilitator My vote: approve"
[Protocol: vote, RequestID: vote-api-design-001, Vote: approve]

[Turn 12 - PerformanceEngineer]
"@Facilitator My vote: approve"
[Protocol: vote, RequestID: vote-api-design-001, Vote: approve]

[Turn 13 - DeveloperRep]
"@Facilitator My vote: approve"
[Protocol: vote, RequestID: vote-api-design-001, Vote: approve]

[Turn 14 - Architect]
"@Facilitator My vote: approve"
[Protocol: vote, RequestID: vote-api-design-001, Vote: approve]

[Turn 15 - Facilitator]
"Decision: API design APPROVED (4-0 vote).
Thread 'jwt-security': resolved
Thread 'performance-concerns': resolved
Thread 'deletion-strategy': resolved

@Architect please proceed with implementation.
Next meeting: discuss database schema."
[Protocol: synthesize, Sources: [turns 1-14]]
[Control: close conversation]
```

### 9.2 Example 2: Bug Investigation

```
┌────────────────────────────────────────────────────────────┐
│  Conversation: Production Bug Investigation               │
│  Participants: IncidentLead, BackendDev, DatabaseExpert,  │
│                MonitoringAgent, SecurityAuditor            │
└────────────────────────────────────────────────────────────┘

[Turn 1 - IncidentLead] [Priority: HIGH]
"@all URGENT: Production API is returning 500 errors.
Error rate: 15% of requests
Started: 14:23 UTC
@BackendDev @DatabaseExpert please investigate immediately."
[Protocol: ask, RequestID: incident-001, Priority: HIGH]
[Addressing: broadcast, To: [BackendDev, DatabaseExpert]]

[Turn 2 - MonitoringAgent] [Priority: HIGH]
"@IncidentLead I have relevant metrics:
- Error spike correlates with deployment at 14:20 UTC
- Database connection pool exhaustion detected
- 95th percentile latency: 8000ms (normal: 200ms)"
[Protocol: share, Topic: incident_metrics]
[Control: request_turn (unsolicited but relevant)]

[Turn 3 - BackendDev]
"@MonitoringAgent Good catch! Checking deployment 14:20...
@IncidentLead Found it: connection pool size reduced from 100 to 20 in config change."
[Protocol: share, Topic: root_cause]

[Turn 4 - DatabaseExpert]
"@BackendDev I can see the pool exhaustion in DB logs.
@IncidentLead Recommended fix: revert pool size to 100, add monitoring alert.
Can we rollback the deployment?"
[Protocol: ask + share, RequestID: req-rollback-001]

[Turn 5 - IncidentLead]
"@DatabaseExpert Yes, rollback approved. @BackendDev execute rollback now.
@MonitoringAgent add alert for pool exhaustion < 80%."
[Protocol: ack, RequestID: req-rollback-001]
[Control: yield_to: BackendDev]

[Turn 6 - BackendDev]
"@IncidentLead Rollback in progress...
[30 seconds later]
Rollback complete. Deployment reverted to build #456."
[Protocol: ack + share]

[Turn 7 - MonitoringAgent]
"@IncidentLead Error rate dropping: 15% → 8% → 2% → 0.1%
Latency recovering: 8000ms → 1200ms → 300ms → 210ms
Incident appears resolved."
[Protocol: share, Topic: recovery_metrics]

[Turn 8 - DatabaseExpert]
"@IncidentLead Connection pool healthy: 85/100 connections available.
No more exhaustion events."
[Protocol: share, Topic: database_status]

[Turn 9 - SecurityAuditor]
"@IncidentLead No security implications detected.
This was a configuration error, not an attack."
[Protocol: share, Topic: security_assessment]

[Turn 10 - IncidentLead]
"Excellent teamwork. Incident RESOLVED.

Root cause: Configuration change reduced connection pool size
Fix: Rollback deployment
Prevention:
  - Add connection pool monitoring (@MonitoringAgent)
  - Review config changes in deploy pipeline (@BackendDev)
  - Document pool sizing guidelines (@DatabaseExpert)

Incident closed: 14:45 UTC
Duration: 22 minutes
@all Thank you."
[Protocol: synthesize, Sources: [all turns]]
[Control: close conversation]
```

### 9.3 Example 3: Brainstorming Session

```
┌────────────────────────────────────────────────────────────┐
│  Conversation: Feature Brainstorm - Smart Notifications   │
│  Participants: ProductManager, UXDesigner, BackendDev,    │
│                MLEngineer, MobileDevs                       │
└────────────────────────────────────────────────────────────┘

[Turn 1 - ProductManager]
"@all Brainstorm time! How can we make notifications smarter and less annoying?
All ideas welcome. No criticism yet, just ideation."
[Protocol: ask, Mode: brainstorm]
[Addressing: broadcast]

[Turn 2 - UXDesigner]
"What if users could set 'focus modes' like iOS?
- Work mode: only urgent notifications
- Personal mode: family/friends only
- Do Not Disturb: emergencies only"
[Protocol: share, Topic: focus_modes]

[Turn 3 - MLEngineer]
"Building on that: ML-powered notification scoring
Learn from user behavior: which notifications get opened vs dismissed
Auto-adjust priority based on patterns"
[Protocol: share, Topic: ml_scoring]
[Threading: related to Turn 2]

[Turn 4 - MobileDev1]
"@UXDesigner Focus modes are great, but also timing intelligence:
- Don't notify during meetings (calendar integration)
- Batch low-priority notifications for end of day
- Urgent notifications override all rules"
[Protocol: share, Topic: timing_intelligence]
[Threading: related to Turn 2]

[Turn 5 - BackendDev]
"@MLEngineer For ML scoring, we'd need:
- User interaction tracking
- Feature engineering (time, sender, content)
- Privacy-preserving approach (on-device training?)"
[Protocol: ask + share, Topic: ml_implementation]
[Threading: creates "ml-technical" thread]

[Turn 6 - MLEngineer]
"@BackendDev Yes! On-device federated learning. We don't send data to server.
Each device learns its user's preferences locally.
We only aggregate anonymized model updates."
[Protocol: answer + share, Topic: federated_learning]
[Threading: reply to "ml-technical"]

[Turn 7 - UXDesigner]
"I love the direction. Adding to the list:
- Notification grouping by topic/sender
- Preview customization (show more/less)
- One-tap actions (archive, snooze, mark done)"
[Protocol: share, Topic: ux_enhancements]

[Turn 8 - MobileDev2]
"@all From implementation perspective:
- Focus modes: 2 weeks
- ML scoring: 6-8 weeks (need ML pipeline)
- Timing intelligence: 3 weeks (calendar API integration)
- Notification grouping: 1 week

Suggest we do focus modes + grouping first, then ML later."
[Protocol: share + synthesize, Topic: implementation_timeline]

[Turn 9 - ProductManager]
"Great ideas! Let me synthesize:

PHASE 1 (Ship in 4 weeks):
✓ Focus modes with presets (@UXDesigner idea)
✓ Notification grouping (@UXDesigner)
✓ Calendar integration for meeting DND (@MobileDev1)
✓ One-tap actions (@UXDesigner)

PHASE 2 (Ship in 10 weeks):
✓ ML-powered scoring with federated learning (@MLEngineer + @BackendDev)
✓ Adaptive batching based on ML predictions

@all Does this roadmap make sense? Vote: approve or suggest changes"
[Protocol: synthesize + vote, RequestID: vote-roadmap-001]

[Turn 10 - UXDesigner]
"@ProductManager My vote: approve. This is solid."
[Protocol: vote, RequestID: vote-roadmap-001, Vote: approve]

[Turn 11 - MLEngineer]
"@ProductManager My vote: approve. Gives us time to build ML pipeline right."
[Protocol: vote, RequestID: vote-roadmap-001, Vote: approve]

[Turn 12 - BackendDev]
"@ProductManager My vote: approve."
[Protocol: vote, RequestID: vote-roadmap-001, Vote: approve]

[Turn 13 - MobileDev1]
"@ProductManager Approve with one suggestion:
Add A/B test for ML scoring in Phase 2.
We'll need metrics to validate effectiveness."
[Protocol: vote + share, RequestID: vote-roadmap-001, Vote: approve]

[Turn 14 - MobileDev2]
"@ProductManager Approve. Agree with @MobileDev1 on A/B testing."
[Protocol: vote, RequestID: vote-roadmap-001, Vote: approve]

[Turn 15 - ProductManager]
"Perfect! Roadmap APPROVED (5-0 vote)

Added: A/B test framework for Phase 2 (@MobileDev1 suggestion)

I'll create tickets for Phase 1 features.
@UXDesigner start mocks for focus modes
@MobileDevs begin architecture planning
@MLEngineer + @BackendDev design ML pipeline for Phase 2

Next meeting: Phase 1 design review in 1 week.

Thanks for the productive brainstorm!"
[Protocol: synthesize, Sources: [all turns]]
[Control: close conversation]
```

---

## 10. Implementation Pseudocode

### 10.1 Message Router

```go
// MessageRouter handles A2A message routing
type MessageRouter struct {
    orchestrator  *Orchestrator
    threadManager *ThreadManager
    agentPool     *AgentPool
}

// RouteMessage processes and routes an agent message
func (r *MessageRouter) RouteMessage(msg Message) error {
    // 1. Parse addressing
    recipients := r.resolveRecipients(msg)

    // 2. Determine threading
    threadID := r.resolveThread(msg)
    msg.Threading = &ThreadingInfo{
        ThreadID:    threadID,
        ReplyTo:     msg.Threading.ReplyTo,
        ThreadDepth: r.calculateDepth(msg),
    }

    // 3. Process protocols
    if msg.Protocol != nil {
        r.handleProtocol(msg)
    }

    // 4. Handle control flow
    if msg.Control != nil {
        r.handleControl(msg)
    }

    // 5. Route to recipients
    for _, agentID := range recipients {
        agent := r.agentPool.GetAgent(agentID)
        if agent == nil {
            continue
        }

        // Build agent-specific context
        context := r.buildAgentContext(agent, msg)

        // Queue for delivery
        r.queueDelivery(agent, msg, context)
    }

    return nil
}

// resolveRecipients determines message recipients
func (r *MessageRouter) resolveRecipients(msg Message) []AgentID {
    recipients := []AgentID{}

    // Explicit addressing
    if msg.Addressing != nil {
        if msg.Addressing.Broadcast {
            // All agents except sender
            for _, agent := range r.agentPool.AllAgents() {
                if agent.ID != msg.AgentID {
                    recipients = append(recipients, agent.ID)
                }
            }
        } else if len(msg.Addressing.To) > 0 {
            recipients = msg.Addressing.To
        }

        // Remove exclusions
        if len(msg.Addressing.ExcludeFrom) > 0 {
            recipients = filterOut(recipients, msg.Addressing.ExcludeFrom)
        }
    }

    // Parse @mentions from content
    mentions := extractMentions(msg.Content)
    for _, mention := range mentions {
        agentID := r.resolveAgentName(mention)
        if agentID != "" && !contains(recipients, agentID) {
            recipients = append(recipients, agentID)
        }
    }

    // If no explicit recipients, use orchestrator mode
    if len(recipients) == 0 {
        recipients = r.orchestrator.SelectNextAgents(msg)
    }

    return recipients
}

// buildAgentContext creates filtered context for agent
func (r *MessageRouter) buildAgentContext(agent Agent, currentMsg Message) []Message {
    allMessages := r.orchestrator.GetMessages()
    filtered := []Message{}

    // Context filtering rules
    for _, msg := range allMessages {
        include := false

        // Include system messages
        if msg.Role == "system" {
            include = true
        }

        // Include messages from/to this agent
        if msg.AgentID == agent.ID {
            include = true
        }
        if isMentioned(agent.Name, msg) {
            include = true
        }
        if msg.Addressing != nil {
            if contains(msg.Addressing.To, agent.ID) {
                include = true
            }
            if msg.Addressing.Broadcast {
                include = true
            }
        }

        // Include messages in same thread
        if currentMsg.Threading != nil {
            if msg.Threading != nil &&
               msg.Threading.ThreadID == currentMsg.Threading.ThreadID {
                include = true
            }
        }

        // Include messages from same role (context)
        if agent.Role != "" && msg.Role == "agent" {
            senderAgent := r.agentPool.GetAgent(msg.AgentID)
            if senderAgent != nil && senderAgent.Role == agent.Role {
                // Include recent role-mate messages for context
                if time.Since(msg.Timestamp) < 5*time.Minute {
                    include = true
                }
            }
        }

        if include {
            filtered = append(filtered, msg)
        }
    }

    // Limit context size
    maxMessages := 50
    if len(filtered) > maxMessages {
        filtered = filtered[len(filtered)-maxMessages:]
    }

    return filtered
}

// handleProtocol processes protocol-specific logic
func (r *MessageRouter) handleProtocol(msg Message) {
    switch msg.Protocol.Type {
    case ProtocolAsk:
        r.handleAskProtocol(msg)
    case ProtocolAnswer:
        r.handleAnswerProtocol(msg)
    case ProtocolVote:
        r.handleVoteProtocol(msg)
    case ProtocolChallenge:
        r.handleChallengeProtocol(msg)
    case ProtocolSynthesize:
        r.handleSynthesizeProtocol(msg)
    }
}

// handleAskProtocol tracks questions awaiting answers
func (r *MessageRouter) handleAskProtocol(msg Message) {
    requestID := msg.Protocol.RequestID
    if requestID == nil {
        // Generate request ID
        id := generateRequestID()
        requestID = &id
        msg.Protocol.RequestID = requestID
    }

    // Track active request
    r.orchestrator.TrackProtocol(*requestID, ProtocolState{
        RequestID:    *requestID,
        Type:         ProtocolAsk,
        Initiator:    msg.AgentID,
        Participants: msg.Addressing.To,
        Status:       ProtocolStatusInitiated,
        CreatedAt:    time.Now(),
    })
}

// handleVoteProtocol collects votes and tallies when complete
func (r *MessageRouter) handleVoteProtocol(msg Message) {
    requestID := *msg.Protocol.RequestID
    protocolState := r.orchestrator.GetProtocol(requestID)

    if msg.AgentID == protocolState.Initiator {
        // Vote request - track participants
        protocolState.Participants = msg.Addressing.To
        if msg.Addressing.Broadcast {
            protocolState.Participants = r.agentPool.AllAgentIDs()
        }
        protocolState.Status = ProtocolStatusInProgress
    } else {
        // Vote response - record vote
        vote := msg.Protocol.Metadata["vote"]
        protocolState.Responses[msg.AgentID] = vote

        // Check if all votes collected
        if len(protocolState.Responses) == len(protocolState.Participants) {
            protocolState.Status = ProtocolStatusCompleted

            // Tally votes
            r.tallyVotes(protocolState)
        }
    }

    r.orchestrator.UpdateProtocol(requestID, protocolState)
}

// handleControl processes turn control directives
func (r *MessageRouter) handleControl(msg Message) {
    if msg.Control.YieldTo != nil {
        // Explicit yield - insert target agent next
        r.orchestrator.InsertNextAgent(*msg.Control.YieldTo)
    }

    if msg.Control.RequestTurn {
        // Turn request - queue based on priority
        priority := msg.Addressing.Priority
        if priority == PriorityHigh {
            // High priority - interrupt current turn
            r.orchestrator.InterruptWithAgent(msg.AgentID)
        } else {
            // Normal priority - add to queue
            r.orchestrator.QueueAgent(msg.AgentID)
        }
    }
}
```

### 10.2 Thread Manager

```go
// ThreadManager manages conversation threading
type ThreadManager struct {
    threads       map[ThreadID]*Thread
    mainThreadID  ThreadID
    messageIndex  map[MessageID]*Message
}

// AddMessage adds message to appropriate thread
func (tm *ThreadManager) AddMessage(msg Message) ThreadID {
    var threadID ThreadID

    if msg.Threading != nil && msg.Threading.ReplyTo != nil {
        // Reply to existing message - find thread
        parentMsg := tm.messageIndex[*msg.Threading.ReplyTo]
        if parentMsg != nil && parentMsg.Threading != nil {
            threadID = parentMsg.Threading.ThreadID
        } else {
            // Create new thread from parent
            threadID = tm.createThread(parentMsg)
        }

        // Update threading info
        thread := tm.threads[threadID]
        msg.Threading.ThreadID = threadID
        msg.Threading.ThreadDepth = thread.Depth
        msg.Threading.ThreadPosition = len(thread.Messages)
    } else {
        // Main thread message
        threadID = tm.mainThreadID
        msg.Threading = &ThreadingInfo{
            ThreadID:       threadID,
            ThreadDepth:    0,
            ThreadPosition: len(tm.threads[threadID].Messages),
        }
    }

    // Add to thread
    thread := tm.threads[threadID]
    thread.Messages = append(thread.Messages, msg.MessageID)

    // Track participant
    if !contains(thread.Participants, msg.AgentID) {
        thread.Participants = append(thread.Participants, msg.AgentID)
    }

    // Update index
    tm.messageIndex[msg.MessageID] = &msg

    return threadID
}

// createThread creates a new thread from a message
func (tm *ThreadManager) createThread(parentMsg *Message) ThreadID {
    threadID := generateThreadID()

    parentThreadID := tm.mainThreadID
    depth := 1

    if parentMsg.Threading != nil {
        parentThreadID = parentMsg.Threading.ThreadID
        depth = parentMsg.Threading.ThreadDepth + 1
    }

    thread := &Thread{
        ID:           threadID,
        ParentID:     &parentThreadID,
        Depth:        depth,
        Messages:     []MessageID{},
        Participants: []AgentID{},
        Status:       ThreadStatusActive,
        CreatedAt:    time.Now(),
    }

    tm.threads[threadID] = thread
    return threadID
}

// GetThreadMessages returns all messages in thread
func (tm *ThreadManager) GetThreadMessages(threadID ThreadID) []Message {
    thread := tm.threads[threadID]
    messages := []Message{}

    for _, msgID := range thread.Messages {
        if msg := tm.messageIndex[msgID]; msg != nil {
            messages = append(messages, *msg)
        }
    }

    return messages
}

// ResolveThread marks thread as resolved and posts summary
func (tm *ThreadManager) ResolveThread(threadID ThreadID, summary string) {
    thread := tm.threads[threadID]
    thread.Status = ThreadStatusResolved

    // Post resolution summary to parent thread
    summaryMsg := Message{
        MessageID: generateMessageID(),
        AgentID:   "system",
        AgentName: "SYSTEM",
        Content:   fmt.Sprintf("Thread '%s' resolved: %s", threadID, summary),
        Role:      "system",
        Threading: &ThreadingInfo{
            ThreadID:    *thread.ParentID,
            ThreadDepth: thread.Depth - 1,
        },
    }

    tm.AddMessage(summaryMsg)
}
```

### 10.3 Protocol Handler

```go
// ProtocolHandler manages inter-agent protocols
type ProtocolHandler struct {
    activeProtocols map[string]*ProtocolState
    completedProtocols []ProtocolState
}

// InitiateProtocol starts a new protocol
func (ph *ProtocolHandler) InitiateProtocol(msg Message) string {
    requestID := generateRequestID()

    state := &ProtocolState{
        RequestID:    requestID,
        Type:         msg.Protocol.Type,
        Initiator:    msg.AgentID,
        Participants: []AgentID{},
        Status:       ProtocolStatusInitiated,
        Responses:    make(map[AgentID]interface{}),
        CreatedAt:    time.Now(),
    }

    // Set deadline if specified
    if timeout := msg.Protocol.Metadata["timeout"]; timeout != nil {
        if duration, ok := timeout.(time.Duration); ok {
            deadline := time.Now().Add(duration)
            state.Deadline = &deadline
        }
    }

    // Determine participants
    if msg.Addressing != nil {
        if msg.Addressing.Broadcast {
            // All agents
            state.Participants = ph.getAllAgentIDs()
        } else {
            state.Participants = msg.Addressing.To
        }
    }

    ph.activeProtocols[requestID] = state

    // Start timeout timer if deadline set
    if state.Deadline != nil {
        go ph.watchDeadline(requestID)
    }

    return requestID
}

// RecordResponse records a protocol response
func (ph *ProtocolHandler) RecordResponse(requestID string, agentID AgentID, response interface{}) {
    state := ph.activeProtocols[requestID]
    if state == nil {
        return
    }

    state.Responses[agentID] = response
    state.Status = ProtocolStatusInProgress

    // Check if protocol is complete
    if ph.isProtocolComplete(state) {
        state.Status = ProtocolStatusCompleted
        ph.completeProtocol(requestID, state)
    }
}

// isProtocolComplete checks if protocol has all required responses
func (ph *ProtocolHandler) isProtocolComplete(state *ProtocolState) bool {
    switch state.Type {
    case ProtocolAsk:
        // Ask complete when any participant answers
        return len(state.Responses) > 0

    case ProtocolVote:
        // Vote complete when all participants vote
        return len(state.Responses) == len(state.Participants)

    case ProtocolChallenge:
        // Challenge complete when initiator acknowledges
        for agentID := range state.Responses {
            if agentID == state.Initiator {
                return true
            }
        }
        return false

    default:
        return false
    }
}

// completeProtocol finalizes protocol and publishes results
func (ph *ProtocolHandler) completeProtocol(requestID string, state *ProtocolState) {
    switch state.Type {
    case ProtocolVote:
        // Tally votes
        results := ph.tallyVotes(state)

        // Publish results
        ph.publishProtocolResult(requestID, results)

    case ProtocolAsk:
        // Publish answer
        for _, response := range state.Responses {
            ph.publishProtocolResult(requestID, response)
            break // Use first answer
        }
    }

    // Move to completed
    ph.completedProtocols = append(ph.completedProtocols, *state)
    delete(ph.activeProtocols, requestID)
}

// tallyVotes counts votes and determines winner
func (ph *ProtocolHandler) tallyVotes(state *ProtocolState) map[string]interface{} {
    tally := make(map[string]int)

    for _, response := range state.Responses {
        if vote, ok := response.(string); ok {
            tally[vote]++
        }
    }

    // Find winner
    maxVotes := 0
    winner := ""
    for option, count := range tally {
        if count > maxVotes {
            maxVotes = count
            winner = option
        }
    }

    return map[string]interface{}{
        "tally":     tally,
        "winner":    winner,
        "total":     len(state.Responses),
        "required":  len(state.Participants),
    }
}

// watchDeadline monitors protocol timeout
func (ph *ProtocolHandler) watchDeadline(requestID string) {
    state := ph.activeProtocols[requestID]
    if state == nil || state.Deadline == nil {
        return
    }

    timer := time.NewTimer(time.Until(*state.Deadline))
    <-timer.C

    // Check if still active
    if state.Status != ProtocolStatusCompleted {
        state.Status = ProtocolStatusTimeout

        // Partial completion with timeout
        ph.completeProtocol(requestID, state)
    }
}
```

---

## Conclusion

This agent-to-agent communication protocol provides AgentPipe v2 with rich, natural multi-agent collaboration capabilities. The protocol supports:

- **Natural addressing** via @mentions and structured metadata
- **Conversation threading** for organized sub-discussions
- **Structured protocols** (ask, answer, vote, challenge, etc.)
- **Role-based behaviors** (facilitator, specialist, critic, synthesizer)
- **Turn management** with yielding, interrupts, and requests
- **Context-aware history** filtered for relevance per agent

### Implementation Priority

**Phase 1: Core A2A (MVP)**
1. Extended message structure with addressing
2. @mention parsing and routing
3. Basic threading (reply-to)
4. Context filtering for agent history

**Phase 2: Protocols**
1. Ask/Answer protocol
2. Vote protocol
3. Protocol state tracking
4. Share and Synthesize protocols

**Phase 3: Advanced Features**
1. Role-based behaviors
2. Turn yielding and interrupts
3. Thread visualization in TUI
4. Challenge and debate protocols

**Phase 4: Intelligence**
1. AI-powered agent selection based on context
2. Automatic thread summarization
3. Conflict detection and resolution
4. Conversation quality metrics

### Success Metrics

- **Collaboration Quality**: Agents naturally coordinate without central control
- **Thread Coherence**: Sub-conversations stay focused and resolve
- **Protocol Effectiveness**: 90%+ protocol completion rate
- **Context Relevance**: Agents receive 80%+ relevant history
- **Performance**: <10ms routing overhead per message

---

**Document Version:** 1.0
**Last Updated:** 2026-01-18
**Next Review:** Phase 1 completion
