# AgentPipe Documentation

This directory contains comprehensive documentation for the AgentPipe project.

## 🚀 AgentPipe v2 MVP (January 2026)

**NEW**: Complete v2 MVP design and implementation plan focusing on real-time parallel multi-agent conversation.

### Quick Start (v2)

1. **Overview** - Read `v2-index.md` first (navigation and quick reference)
2. **MVP Plan** - Understand scope: `v2-mvp-implementation-plan.md` (4-week timeline)
3. **Architecture** - See how it works: `v2-architecture-diagram.md` (visual diagrams)
4. **Start Coding** - Developer setup: `v2-developer-quickstart.md` (Week 1-3 tasks)

### v2 Documentation Index

| Document | Purpose | Audience | Status |
|----------|---------|----------|--------|
| `v2-index.md` | Documentation navigation and quick reference | All | ✅ Complete |
| `v2-mvp-implementation-plan.md` | MVP feature set, 4-week timeline, testing | PM, Tech Leads | ✅ Complete |
| `v2-architecture-diagram.md` | System architecture, message flow, diagrams | Architects, Developers | ✅ Complete |
| `v2-core-interfaces.md` | Core types, interfaces, code examples | Developers | ✅ Complete |
| `v2-migration-guide.md` | v1 → v2 migration, breaking changes, FAQ | Users, Developers | ✅ Complete |
| `v2-developer-quickstart.md` | Setup, Week 1-3 tasks, patterns, debugging | Developers | ✅ Complete |

### Key v2 Features

**Core Value**: User sends message → All agents respond in parallel with real-time streaming

**Architecture Highlights**:
- Event-driven (pub/sub event bus)
- Parallel-first (all agents execute concurrently)
- Streaming-first (responses appear as generated)
- API-first (OpenRouter, Claude, etc.)
- Simple (1 orchestration mode vs 5 in v1)

**Performance**:
- 2-4× faster than v1 (parallel vs sequential)
- First response in ~1.5s (vs ~6.5s in v1)
- Real-time streaming (<100ms latency)

**Timeline**: 4 weeks to shippable MVP
- Week 1: Core infrastructure (event bus, adapters)
- Week 2: Conversation engine (parallel execution)
- Week 3: TUI interface (real-time updates)
- Week 4: Integration, testing, release

### v2 Research Documents

Supporting research for v2 design (read after MVP docs):

| Document | Purpose | Relevance to MVP |
|----------|---------|-----------------|
| `v2-conversation-loop-architecture.md` | Multi-agent loop patterns | Background research |
| `v2-group-chat-ux-design.md` | UX design patterns | Future enhancements |
| `v2-conversation-systems-research.md` | Industry analysis | Design influences |
| `v2-realtime-feedback-system.md` | Streaming patterns | Implemented in MVP |
| `v2-agent-to-agent-protocol.md` | Agent communication | Post-MVP (v2.1) |

---

## Architecture Research (January 2026)

### 📋 Quick Start

1. **Executive Summary** - Read `research-summary.txt` first (2 pages)
2. **Quick Reference** - For developers: `architecture-patterns-quick-reference.md` (code examples, patterns)
3. **Full Report** - Deep dive: `architecture-research-modern-patterns.md` (45 pages)

### 📚 Documentation Index

| Document | Purpose | Audience | Length |
|----------|---------|----------|--------|
| `research-summary.txt` | High-level findings, priorities | Leadership, PM | 2 pages |
| `architecture-patterns-quick-reference.md` | Code examples, patterns catalog | Developers | 15 pages |
| `architecture-research-modern-patterns.md` | Comprehensive analysis | Architects, Tech Leads | 45 pages |

### 🎯 Key Findings Summary

**Overall Grade**: A- (Excellent foundation, targeted improvements)

**Top 5 Strengths**:
1. Interface-driven design (hexagonal architecture)
2. Middleware pattern (production-ready)
3. Orchestrator coordination (thread-safe, resilient)
4. Streaming bridge (non-blocking events)
5. Observability (metrics, logging, events)

**Top 5 Improvements**:
1. Worker pool for parallel execution (HIGH)
2. Refactor orchestrator.go (HIGH)
3. Component-based TUI (HIGH)
4. Config schema validation (HIGH)
5. Circuit breaker pattern (MEDIUM)

### 📖 Using This Research

#### For Product Managers
- Read: `research-summary.txt`
- Focus on: Implementation Priorities section
- Action: Create roadmap based on HIGH/MEDIUM/LOW priorities

#### For Developers
- Read: `architecture-patterns-quick-reference.md`
- Focus on: Code examples, refactoring guide
- Action: Start with high-priority improvements

#### For Architects
- Read: `architecture-research-modern-patterns.md`
- Focus on: Design patterns, scalability, security
- Action: Review ADRs, plan architectural evolution

#### For Tech Leads
- Read all three documents
- Focus on: Implementation priorities, team planning
- Action: Create sprint plan, assign work

### 🔄 Implementation Roadmap

#### Phase 1: High Priority (3 months)
- [ ] Refactor orchestrator.go into smaller files
- [ ] Add errgroup for parallel agent execution
- [ ] Implement worker pool pattern
- [ ] Add config schema validation
- [ ] Enhance error context with sentinel errors

#### Phase 2: Medium Priority (3-6 months)
- [ ] Implement circuit breaker pattern
- [ ] Add response caching layer
- [ ] Create golden file tests for TUI
- [ ] Write architectural decision records
- [ ] Add webhook system

#### Phase 3: Low Priority (6-12 months)
- [ ] Evaluate event bus architecture
- [ ] Consider plugin system (go-plugin)
- [ ] Explore API server mode
- [ ] Add distributed tracing
- [ ] Implement fuzzing tests

### 📊 Research Methodology

**Approach**:
1. Analyzed 222 Go files in the codebase
2. Reviewed architecture patterns (DDD, Hexagonal, etc.)
3. Compared with similar projects (kubectl, gh, k9s, lazygit)
4. Identified strengths and improvement opportunities
5. Prioritized by impact and effort

**Focus Areas**:
- Scalability (worker pools, concurrency)
- Maintainability (code organization, refactoring)
- Testability (coverage, mocking, golden files)
- Extensibility (plugins, webhooks, API)
- Security (secrets, validation, auditing)
- Performance (caching, pooling, profiling)

### 🔗 External References

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Concurrency Patterns](https://talks.golang.org/2012/concurrency.slide)
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [Cobra Documentation](https://github.com/spf13/cobra)

### 📅 Research Timeline

- **2026-01-18**: Initial research completed
- **Next**: Team review and roadmap planning
- **Future**: Quarterly architecture reviews

### ✉️ Contact

For questions about this research:
- Architecture questions: Review full report
- Implementation questions: Check quick reference
- Priority questions: See research summary

---

**Research Version**: 1.0
**Last Updated**: 2026-01-18
**Status**: Complete
**Next Review**: Q2 2026
