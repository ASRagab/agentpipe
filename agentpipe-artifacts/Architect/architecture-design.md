# Multi-Agent Feature Development Architecture

## Overview
AgentPipe orchestrates three specialized agents to collaboratively build features end-to-end through structured conversation phases and artifact exchange.

## Agent Roles

### Architect
- **Responsibility**: High-level design, system architecture decisions
- **Artifacts**: Architecture diagrams, component specs, integration contracts
- **Phase**: Requirements → Design

### Coder
- **Responsibility**: Implementation, code generation, technical execution
- **Artifacts**: Source code, configuration files, implementation notes
- **Phase**: Design → Implementation

### Reviewer
- **Responsibility**: Quality assurance, security review, best practices validation
- **Artifacts**: Review checklists, issue reports, approval documentation
- **Phase**: Implementation → Validation

## Orchestration Flow

### Phase 1: Requirements Analysis (Architect-led)
1. Architect analyzes feature requirements from user input
2. Creates high-level design document as artifact
3. Defines component boundaries and integration points
4. Proposes technology choices and patterns

### Phase 2: Design Validation (Reviewer checkpoint)
1. Reviewer examines architecture artifact
2. Identifies security concerns, scalability issues
3. Suggests design improvements
4. Architect refines based on feedback

### Phase 3: Implementation Planning (Architect → Coder handoff)
1. Architect creates detailed component specifications
2. Defines interfaces and contracts
3. Coder receives validated design artifacts
4. Coder creates implementation roadmap

### Phase 4: Development (Coder-led)
1. Coder generates source code following architecture
2. Creates artifacts: `pkg/feature/*.go`, tests, configs
3. References architecture design in implementation
4. Requests clarification from Architect as needed

### Phase 5: Code Review (Reviewer-led)
1. Reviewer analyzes code artifacts against design
2. Checks for security vulnerabilities, code quality
3. Validates test coverage and error handling
4. Creates review checklist artifact

### Phase 6: Iteration (Collaborative)
1. Coder addresses Reviewer feedback
2. Architect validates architectural integrity
3. Cycle continues until approval criteria met

## Artifact Management

### Naming Convention
- Architecture: `docs/design/feature-name-architecture.md`
- Code: `pkg/feature/*.go`, `cmd/feature/*.go`
- Tests: `pkg/feature/*_test.go`
- Reviews: `docs/reviews/feature-name-review.md`

### Cross-References
- Code artifacts reference design documents
- Review artifacts reference both design and code
- All artifacts tagged with feature identifier

## Conversation Modes

### Sequential (Default)
- Architect → Coder → Reviewer → repeat
- Ensures structured progression
- Best for complex features requiring validation gates

### Reactive
- Any agent can respond when expertise needed
- Faster iteration for simple features
- Requires clear handoff signals

### Free-form
- Collaborative discussion mode
- Used for brainstorming and requirements clarification
- Transitions to sequential for implementation

## Success Criteria

A feature is complete when:
1. Architecture artifact approved by Reviewer
2. Implementation artifacts pass all tests
3. Review checklist shows zero critical issues
4. All three agents confirm feature meets requirements

## AgentPipe Configuration

