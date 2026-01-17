# Review Checklist: AgentPipe 3-Agent Orchestration

## Architecture Decisions (from `architecture-design.md`)
- [ ] Roles & phases are enforced: Architect (Requirements/Design), Coder (Implementation), Reviewer (Validation) align with defined flow and phase gates.
- [ ] Orchestration flow matches the six phases, including explicit Design Validation checkpoint before implementation.
- [ ] Conversation modes are implemented with documented behavior: sequential default, reactive, free-form with fallbacks.
- [ ] Success criteria are codified: architecture approval, tests passing, zero critical review items, all agents confirm completion.
- [ ] Artifact naming convention and cross-references are enforced (`docs/design/...`, `pkg/...`, `docs/reviews/...`).

## Implementation Approach (from `implementation-sketch.go`)
- [ ] Orchestrator `Run` loop follows the phase sequencing and handles max-turns with completion checks.
- [ ] Agent selection logic (`getAgentByTurn`) honors sequential mode and has safe fallback for reactive/free-form.
- [ ] Artifact parsing from fenced blocks is implemented and validated against the expected format.
- [ ] State updates persist history + artifacts and prevent silent overwrites or loss.
- [ ] Prompt construction includes role instructions + history + relevant artifacts as described.
- [ ] Error handling covers LLM failures, missing agents, and invalid config.

## Integration & Quality Gates
- [ ] Config loading matches YAML described in `architecture-design.md` (mode, agents, max-turns, artifacts).
- [ ] Reviewer gate is enforced before Coder starts if Design phase not approved.
- [ ] Tests exist for artifact parsing, agent sequencing, and completion logic.

## Security & Reliability
- [ ] Artifact writes are constrained to allowed paths and avoid traversal.
- [ ] Large artifact or history payloads are bounded to prevent prompt overflow.
- [ ] External LLM calls include retry/timeout and cancellation via context.

## Questions / Assumptions
- [ ] Is “approval” a structured signal (e.g., checklist artifact status) or free-form text?
- [ ] How is “tests passing” validated in the orchestration loop (local vs external hook)?
