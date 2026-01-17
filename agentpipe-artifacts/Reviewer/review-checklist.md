# Reviewer Checklist for 3-Agent Orchestration (Architect/Coder/Reviewer)

This checklist validates the orchestration design and its implementation sketch. It references:
- `architecture-design.md` (roles, phases, artifact management, modes, success criteria, config)
- `implementation-sketch.go` (Agent/Artifact types, Orchestrator loop, state tracking, mode handling)

## Architecture Decisions (from `architecture-design.md`)
- [ ] **Role boundaries** are explicit and enforced in prompts (Architect: Requirements/Design, Coder: Implementation, Reviewer: Validation).  
  *Ref: "Agent Roles" section.*
- [ ] **Phases** are represented as gated transitions (Requirements → Design → Implementation → Review → Iteration).  
  *Ref: "Orchestration Flow" phases 1–6.*
- [ ] **Artifact naming & cross-references** are enforced (design docs, code paths, reviews).  
  *Ref: "Artifact Management" / "Naming Convention" / "Cross-References."*
- [ ] **Conversation modes** are implemented or at least scoped with clear behavior (Sequential/Reactive/Free-form).  
  *Ref: "Conversation Modes" sections.*
- [ ] **Success criteria** are explicit and machine-checkable (review approval, tests passing, confirmations).  
  *Ref: "Success Criteria."*
- [ ] **Configuration schema** covers mode/agents/max-turns/artifacts.  
  *Ref: "AgentPipe Configuration" YAML block.*

## Implementation Approach (from `implementation-sketch.go`)
- [ ] **Core structs align with architecture**: Agent, Artifact, ConversationState.  
  *Ref: "Core Types" mapping to roles/artifact management.*
- [ ] **Orchestrator loop** mirrors phased flow and max-turns logic.  
  *Ref: `Run()` and "Orchestration Flow".*
- [ ] **Mode handling** is complete (sequential/reactive/free-form not just fallback).  
  *Ref: `getAgentByTurn()` and "Conversation Modes".*
- [ ] **Artifact parsing** supports fenced blocks with filenames per instructions.  
  *Ref: `executeTurn()` comments.*
- [ ] **State updates** persist conversation history and artifacts consistently.  
  *Ref: `updateState()` and "Artifact Management".*
- [ ] **Completion checks** implement the success criteria (review approval, tests, agent confirmations).  
  *Ref: `isFeatureComplete()` and "Success Criteria".*
- [ ] **Error handling** is robust for config load, LLM calls, and artifact parsing.  
  *Ref: `NewOrchestrator()` and `executeTurn()` placeholders.*

## Risks / Gaps to Address
- [ ] **Inconsistent artifact naming** between architecture and sketch (e.g., `docs/design/...` vs `agentpipe-artifacts/...`).  
  *Ref: "Naming Convention" vs `implementation-sketch.go` comment.*
- [ ] **Mode fallback hides missing logic** for reactive/free-form; define explicit heuristics.  
  *Ref: `getAgentByTurn()` and "Reactive"/"Free-form".*
- [ ] **No explicit reviewer gate** before coder begins implementation.  
  *Ref: Phase 2 checkpoint vs `Run()` loop.*
- [ ] **No validation of artifact ownership** (Reviewer should not overwrite code artifacts, etc.).  
  *Ref: "Agent Roles" and Artifact Owner field.*

## Test & Validation
- [ ] Unit tests for:
  - Agent sequencing by mode.
  - Artifact parsing from fenced code blocks.
  - Success-criteria evaluation.
- [ ] Integration test for a 3-agent run producing:
  - `docs/design/...`
  - `pkg/feature/...`
  - `docs/reviews/...`
