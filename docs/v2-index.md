# AgentPipe v2 Documentation Index

## Overview

This directory contains all documentation for AgentPipe v2 MVP design and implementation.

**Goal**: Ship a working multi-agent conversation platform in 4 weeks where users send a message and all agents respond in real-time.

---

## Core Documents

### 1. MVP Implementation Plan
**File**: `v2-mvp-implementation-plan.md`

**What it covers**:
- MVP feature set (must-have vs nice-to-have)
- 4-week implementation timeline
- Core architecture components
- Testing strategy
- Migration from v1
- Success criteria
- Risks and mitigations
- Post-MVP roadmap

**Read this first** to understand the project scope and timeline.

**Key sections**:
- **Section 1**: MVP Feature Set (what we're building)
- **Section 2**: Minimal Architecture (how it works)
- **Section 3**: Implementation Timeline (week-by-week plan)
- **Section 5**: Migration from v1 (what to keep, what to rebuild)

---

### 2. Architecture Diagrams
**File**: `v2-architecture-diagram.md`

**What it covers**:
- High-level system architecture
- Message flow diagrams
- Parallel agent execution detail
- Event bus architecture
- Data flow diagrams
- TUI component breakdown
- Configuration structure
- Package structure

**Read this** to understand how components interact.

**Key diagrams**:
- **Diagram 1**: High-level system architecture (TUI → Manager → Pool → Adapters)
- **Diagram 2**: Message flow (user input → parallel agent responses)
- **Diagram 3**: Parallel execution (goroutines + event bus)
- **Diagram 5**: Data flow (end-to-end message lifecycle)

---

### 3. Core Interfaces Reference
**File**: `v2-core-interfaces.md`

**What it covers**:
- Core data types (Message, Agent, Conversation, Event, Metrics)
- Essential interfaces (AgentAdapter, ConversationManager, EventBus, AgentPool)
- Adapter registry
- Complete code examples
- Testing interfaces (mocks)

**Read this** when implementing components.

**Key interfaces**:
- **AgentAdapter**: Communication with AI agents
- **ConversationManager**: Orchestrates conversation flow
- **EventBus**: Pub/sub event distribution
- **AgentPool**: Parallel agent execution

---

### 4. Migration Guide
**File**: `v2-migration-guide.md`

**What it covers**:
- What changed from v1 to v2
- Breaking changes
- Feature comparison
- Migration steps for users and developers
- Config compatibility
- Performance comparison
- Deprecation timeline

**Read this** if you're familiar with v1 or need to migrate.

**Key sections**:
- **Section 2**: Breaking Changes (config format, orchestration modes, event system)
- **Section 3**: What Stayed the Same (agent interface, message structure, utilities)
- **Section 6**: Performance Comparison (2-4× faster with parallel execution)

---

### 5. Developer Quickstart
**File**: `v2-developer-quickstart.md`

**What it covers**:
- 5-minute setup guide
- Development workflow
- Week 1 implementation tasks with code
- Common patterns (thread-safety, context, shutdown)
- Debugging tips
- Testing strategies

**Read this** to start contributing.

**Key sections**:
- **Quick Start**: Clone, setup, run example
- **Week 1 Tasks**: Core data structures, event bus, adapters (with code)
- **Common Patterns**: Thread-safe state, context timeouts, graceful shutdown
- **Debugging Tips**: Logging, profiling, race detection

---

## Document Dependencies

```
Start Here:
    v2-mvp-implementation-plan.md
         │
         ├─> Understand architecture
         │   v2-architecture-diagram.md
         │
         ├─> Learn interfaces
         │   v2-core-interfaces.md
         │
         ├─> Migrate from v1 (optional)
         │   v2-migration-guide.md
         │
         └─> Start coding
             v2-developer-quickstart.md
```

---

## Quick Reference

### For Project Managers
- **Timeline**: `v2-mvp-implementation-plan.md` → Section 3 (4-week timeline)
- **Success Metrics**: `v2-mvp-implementation-plan.md` → Section 6
- **Risks**: `v2-mvp-implementation-plan.md` → Section 7

### For Architects
- **Architecture**: `v2-architecture-diagram.md` → Diagram 1 (high-level)
- **Data Flow**: `v2-architecture-diagram.md` → Diagram 5 (message lifecycle)
- **Interfaces**: `v2-core-interfaces.md` → Section 2 (all interfaces)

### For Developers
- **Setup**: `v2-developer-quickstart.md` → Quick Start
- **Implementation**: `v2-developer-quickstart.md` → Week 1-3 tasks
- **Patterns**: `v2-developer-quickstart.md` → Common Patterns

### For Users (v1 → v2 Migration)
- **What Changed**: `v2-migration-guide.md` → Section 1 (high-level changes)
- **Migration Steps**: `v2-migration-guide.md` → Section 5 (user guide)
- **FAQ**: `v2-migration-guide.md` → Section 9

---

## Weekly Checkpoints

### Week 1: Core Infrastructure
- **Deliverable**: Event bus + 2 adapters working
- **Read**: `v2-developer-quickstart.md` → Week 1 tasks
- **Test**: Run `go test ./pkg/v2/core ./pkg/v2/events ./pkg/v2/adapters`

### Week 2: Conversation Engine
- **Deliverable**: Conversation manager handling parallel responses
- **Read**: `v2-architecture-diagram.md` → Parallel execution
- **Test**: Integration tests for parallel agent execution

### Week 3: TUI Interface
- **Deliverable**: TUI showing real-time conversation
- **Read**: `v2-architecture-diagram.md` → TUI component breakdown
- **Test**: Manual TUI smoke tests

### Week 4: Integration & Testing
- **Deliverable**: Shippable v2.0.0-mvp release
- **Read**: `v2-mvp-implementation-plan.md` → Success criteria
- **Test**: Full end-to-end tests, cross-platform testing

---

## Key Metrics (Success Criteria)

### Functional
- ✅ User sends message → 3+ agents respond in parallel
- ✅ Responses stream in real-time
- ✅ TUI shows agent status indicators
- ✅ Metrics display correctly
- ✅ Conversation save/resume works

### Performance
- ✅ First response: <2s (fast models)
- ✅ Streaming latency: <100ms between chunks
- ✅ Memory: <100MB for 100-message conversation
- ✅ TUI: <16ms frame time (60fps)

### Quality
- ✅ Test coverage: >80%
- ✅ Zero panics in normal operation
- ✅ Graceful error handling
- ✅ Clear error messages

---

## Contributing

### Before You Start
1. Read `v2-mvp-implementation-plan.md` (understand the vision)
2. Read `v2-architecture-diagram.md` (understand how it works)
3. Read `v2-core-interfaces.md` (understand the contracts)
4. Read `v2-developer-quickstart.md` (set up environment)

### Development Process
1. Pick a task from the timeline
2. Write tests first (TDD)
3. Implement feature
4. Run tests, lint, format
5. Submit PR with clear description

### Code Standards
- **Go Version**: 1.24+
- **Coverage**: >80% for new code
- **Concurrency**: Use channels, avoid shared state
- **Documentation**: Godoc for all exported types
- **Testing**: Unit + integration tests

---

## FAQ

### Q: Where do I start?
**A**: Read `v2-mvp-implementation-plan.md` first, then `v2-developer-quickstart.md` for setup.

### Q: What's the fastest way to understand the architecture?
**A**: Look at diagrams in `v2-architecture-diagram.md`, especially Diagram 1 and Diagram 5.

### Q: I'm familiar with v1. What changed?
**A**: Read `v2-migration-guide.md` → Section 1 (high-level changes).

### Q: How do I implement a new adapter?
**A**: See `v2-core-interfaces.md` → Section 2.1 (AgentAdapter interface) and `v2-developer-quickstart.md` → Week 1, Task 3.

### Q: What's the event bus and why do we need it?
**A**: See `v2-architecture-diagram.md` → Diagram 4 (event bus) and `v2-developer-quickstart.md` → Week 1, Task 2.

### Q: How do I test my changes?
**A**: See `v2-developer-quickstart.md` → Testing Strategy.

### Q: When will v2 be released?
**A**: See `v2-mvp-implementation-plan.md` → Section 3 (4-week timeline, Week 4 = release).

### Q: Can I use v2 and v1 side-by-side?
**A**: Yes. See `v2-migration-guide.md` → Section 5 (migration steps).

---

## Document Changelog

| Date | Document | Changes |
|------|----------|---------|
| 2026-01-18 | All | Initial v2 documentation created |
| | v2-mvp-implementation-plan.md | MVP feature set, 4-week timeline, architecture |
| | v2-architecture-diagram.md | System architecture, message flow, diagrams |
| | v2-core-interfaces.md | Core types, interfaces, code examples |
| | v2-migration-guide.md | v1 → v2 migration, breaking changes, FAQ |
| | v2-developer-quickstart.md | Setup, Week 1-3 tasks, patterns, debugging |
| | v2-index.md | Documentation index and quick reference |

---

## Next Steps

1. **Project Managers**: Review timeline in `v2-mvp-implementation-plan.md`
2. **Architects**: Review architecture in `v2-architecture-diagram.md`
3. **Developers**: Set up environment with `v2-developer-quickstart.md`
4. **Users**: Prepare for migration with `v2-migration-guide.md`

---

## Summary

**AgentPipe v2 MVP** is a ground-up rewrite focused on **real-time parallel multi-agent conversation**. The MVP ships in **4 weeks** with:

- ✅ Parallel agent execution (2-4× faster than v1)
- ✅ Real-time streaming responses
- ✅ Event-driven architecture
- ✅ Modern TUI with live updates
- ✅ API-first design (OpenRouter, Claude, etc.)

**Start here**: `v2-mvp-implementation-plan.md`

**Get coding**: `v2-developer-quickstart.md`

Let's ship it! 🚀
