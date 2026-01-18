# AgentPipe Refactoring Roadmap

## TL;DR

AgentPipe has **3 critical architectural issues** that must be addressed within 2-3 releases:

1. 🔴 **16 adapters with ~6,000 lines of duplicated code** (80% duplication)
2. 🔴 **Orchestrator handling 10+ responsibilities** (1,297 lines, God Object)
3. 🟡 **CLI/API agents forced into incompatible interface** (Interface Pollution)

**Breaking Point:** 5-10 more adapters or 3-4 major features → Unmaintainable
**Refactoring Window:** Next 2-4 months before debt becomes unmanageable

---

## Refactoring Priorities

### 🔴 Priority 1: Extract Adapter Template (URGENT)
**Timeline:** 2-3 weeks | **Effort:** Medium | **Risk:** Medium

**Problem:** Adding artifact instructions required identical changes to 16 files (commits #485-#493)

**Solution:**
```go
type BaseAdapter struct {
    agent.BaseAgent
    execPath    string
    cliCommand  string
    flagBuilder FlagBuilder  // Strategy for CLI flags
}

// Shared implementations (no more duplication)
func (b *BaseAdapter) buildPrompt(messages []Message) string
func (b *BaseAdapter) filterRelevantMessages(messages []Message) []Message
func (b *BaseAdapter) SendMessage(ctx, messages) (string, error)
```

**Impact:**
- ✅ Eliminate ~6,000 lines of duplication
- ✅ Feature changes require 1 edit instead of 16
- ✅ Bug fixes propagate automatically

---

### 🔴 Priority 2: Split Orchestrator (URGENT)
**Timeline:** 4-6 weeks | **Effort:** High | **Risk:** High

**Problem:** 1,297 lines handling 10+ responsibilities → Hard to test, modify, understand

**Extract Components:**
1. **RetryManager** - Exponential backoff logic (currently nested in getAgentResponse)
2. **ArtifactProcessor** - Extract/save artifacts (currently inline in orchestrator)
3. **BridgeEventPublisher** - Observer pattern for events (replace direct coupling)
4. **SummaryGenerator** - Conversation summaries (currently in orchestrator)
5. **TurnStrategy** - Strategy pattern for modes (replace runRoundRobin/Reactive/FreeForm)

**Target Structure:**
```go
type Orchestrator struct {
    agents       []agent.Agent
    messages     []agent.Message
    turnStrategy TurnStrategy      // Strategy pattern
    middleware   *middleware.Chain
    eventBus     *EventBus         // Decouple bridge/artifacts
    retryManager *RetryManager
}
```

**Impact:**
- ✅ Orchestrator reduces to ~400 lines (core logic only)
- ✅ Each component testable in isolation
- ✅ Clear separation of concerns
- ✅ Easy to add features without touching orchestrator

---

### 🟡 Priority 3: Agent Capabilities Pattern
**Timeline:** 2-3 weeks | **Effort:** Medium | **Risk:** Medium

**Problem:** API agents forced to implement GetCLIVersion() returning "N/A (API)"

**Solution:**
```go
// Core interface - all agents
type Agent interface {
    GetID() string
    GetName() string
    SendMessage(ctx, messages) (string, error)
}

// Capability interfaces
type CLIExecutable interface {
    Agent
    GetCLIVersion() string
    GetCLIPath() string
}

type APICallable interface {
    Agent
    GetAPIEndpoint() string
}

type Streamable interface {
    StreamMessage(ctx, messages, writer) error
}
```

**Usage:**
```go
if cli, ok := agent.(CLIExecutable); ok {
    version := cli.GetCLIVersion()
}
```

**Impact:**
- ✅ No interface pollution
- ✅ Type-safe capability checks
- ✅ Clean separation: CLI vs API agents

---

### 🟢 Priority 4: Event-Driven Architecture
**Timeline:** 2-3 weeks | **Effort:** Medium | **Risk:** Low

**Problem:** Orchestrator tightly coupled to bridge and artifacts via direct method calls

**Solution:**
```go
type EventBus interface {
    Publish(event Event)
    Subscribe(eventType string, handler EventHandler)
}

// Orchestrator emits events (no direct coupling)
eventBus.Publish(MessageCreatedEvent{...})

// Bridge subscribes (decoupled)
eventBus.Subscribe("message.created", bridge.HandleMessageCreated)
```

**Impact:**
- ✅ Orchestrator doesn't know about bridge/artifacts
- ✅ Enable third-party plugins
- ✅ Easy to add event subscribers

---

### 🟢 Priority 5: Turn Strategy Pattern
**Timeline:** 1 week | **Effort:** Low | **Risk:** Low

**Problem:** 3 mode implementations (runRoundRobin, runReactive, runFreeForm) share no abstraction

**Solution:**
```go
type TurnStrategy interface {
    SelectNextAgent(agents []Agent, history []Message) Agent
    ShouldContinue(turn, maxTurns int) bool
}

type RoundRobinStrategy struct { currentIndex int }
type ReactiveStrategy struct { lastSpeaker string }
type FreeFormStrategy struct {}
```

**Impact:**
- ✅ Custom modes via new strategy implementations
- ✅ Reduce code duplication
- ✅ User-defined orchestrator modes

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-3)
- [ ] Create BaseAdapter template
- [ ] Migrate 2-3 adapters to template (pilot)
- [ ] Verify tests pass
- [ ] **Checkpoint:** If successful, migrate remaining 13 adapters

### Phase 2: Core Split (Weeks 4-9)
- [ ] Extract RetryManager
- [ ] Extract TurnStrategy (Priority 5)
- [ ] Extract BridgeEventPublisher (Priority 4 EventBus)
- [ ] Extract ArtifactProcessor
- [ ] Extract SummaryGenerator
- [ ] **Checkpoint:** Orchestrator reduced to ~400 lines

### Phase 3: Interface Clean-up (Weeks 10-12)
- [ ] Design Agent capability interfaces
- [ ] Implement CLIExecutable, APICallable, Streamable
- [ ] Migrate existing agents to new interfaces
- [ ] **Checkpoint:** No more interface pollution

### Phase 4: Validation (Weeks 13-14)
- [ ] Comprehensive integration tests
- [ ] Performance regression tests
- [ ] Documentation updates
- [ ] **Release:** v1.0 with clean architecture

---

## Success Metrics

### Before Refactoring
- Adapter duplication: **~6,000 lines**
- Orchestrator size: **1,297 lines**
- Feature cost: **~15 hours** (16 files to change)
- Responsibilities per file: **10+ in orchestrator**
- Interface violations: **API agents faking CLI methods**

### After Refactoring
- Adapter duplication: **~0 lines** (shared template)
- Orchestrator size: **~400 lines** (focused on core)
- Feature cost: **~3.5 hours** (1 template change)
- Responsibilities per file: **1-2 max** (SRP)
- Interface violations: **0** (capability pattern)

### Efficiency Gain
**4.3× faster feature development** after refactoring

---

## Risk Mitigation

### High-Risk Changes
1. **Orchestrator Split** - Core component, extensive testing required
   - Mitigation: Incremental extraction, keep old code until fully tested
   - Testing: Integration tests, comparison tests (old vs new behavior)

2. **Adapter Template Migration** - 16 adapters to migrate
   - Mitigation: Pilot with 2-3 adapters, validate before mass migration
   - Testing: Adapter-specific test suites must all pass

### Low-Risk Changes
1. **TurnStrategy** - Well-contained, clear abstraction
2. **EventBus** - Additive change, can coexist with existing code

---

## Decision Points

### Week 3 (After Phase 1)
**Question:** Continue with orchestrator split?
**Criteria:**
- ✅ All 16 adapters migrated successfully
- ✅ No test regressions
- ✅ Code review approved
- ❌ If issues found → Fix adapter template before proceeding

### Week 9 (After Phase 2)
**Question:** Proceed with interface refactoring?
**Criteria:**
- ✅ Orchestrator tests pass
- ✅ Performance benchmarks within 5% of baseline
- ✅ Integration tests pass
- ❌ If issues found → Fix extracted components before proceeding

### Week 12 (Before Release)
**Question:** Ship v1.0?
**Criteria:**
- ✅ All integration tests pass
- ✅ No regressions in existing features
- ✅ Performance meets or exceeds baseline
- ✅ Documentation complete
- ❌ If not ready → Extend validation phase

---

## Rollback Plan

### If Refactoring Fails
1. **Keep old code paths** during migration
2. **Feature flags** to toggle new vs old implementations
3. **Git branches** for each phase - easy to revert
4. **Comprehensive tests** to catch regressions early

### Rollback Triggers
- Test pass rate < 95%
- Performance regression > 10%
- Critical bug in production
- Schedule slip > 4 weeks beyond estimate

---

## Communication Plan

### Stakeholders
- **Development Team:** Weekly updates on progress
- **Users:** Release notes explaining improvements
- **Contributors:** Migration guide for custom adapters

### Milestones
1. **Week 3:** Adapter template complete
2. **Week 9:** Orchestrator refactored
3. **Week 12:** Capabilities implemented
4. **Week 14:** v1.0 release

---

## Questions & Answers

### Q: Why not rewrite from scratch?
**A:** Refactoring preserves working code, tests, and domain knowledge. Rewrite risks introducing new bugs and losing edge cases.

### Q: Can we do this incrementally?
**A:** Yes! Each priority can be tackled independently. Start with P1 (adapters), then P5 (strategies), then P2 (orchestrator).

### Q: What if we skip refactoring?
**A:** Technical debt compounds. In 6 months, cost will be 3-4× higher. In 12 months, may require full rewrite.

### Q: How do we prevent this from happening again?
**A:**
1. Architectural review for major changes
2. Code review checklist (DRY, SRP, interface segregation)
3. Automated linting for complexity/duplication
4. Regular architectural health checks

---

## Next Steps

1. **Team Review:** Present this roadmap, get buy-in
2. **Create Issues:** Break down each priority into tasks
3. **Assign Owners:** Each phase needs a lead developer
4. **Set Deadlines:** Commit to timeline, track progress weekly
5. **Start Phase 1:** Begin adapter template extraction

---

**Document Owner:** System Architecture Designer
**Last Updated:** 2026-01-18
**Status:** 📋 Proposal - Awaiting Team Review
