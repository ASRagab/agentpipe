# Review Checklist: AgentPipe Multi-Agent Orchestration

References:

- Architecture: `architecture-design.md`
- Implementation sketch: `agentpipe-artifacts/Coder/implementation-sketch.go`

## Architecture decisions

- [ ] Role separation matches the three-agent model (Architect, Coder, Reviewer) with distinct responsibilities.
- [ ] Phase gates align with the orchestration flow (Requirements -> Design Validation -> Implementation -> Review -> Iteration).
- [ ] Conversation modes (sequential, reactive, free-form) are explicitly supported with clear selection logic.
- [ ] Artifact naming conventions and output directories follow the design (e.g., `docs/design/...`, `docs/reviews/...`).
- [ ] Success criteria are enforceable and mapped to explicit signals (review approval, test pass, agent confirmations).

## Implementation approach

- [ ] Orchestrator loop enforces turn order in sequential mode (Architect -> Coder -> Reviewer) as in `getAgentByTurn`.
- [ ] Prompt building includes role instructions, conversation history, and relevant artifacts per cross-reference rules.
- [ ] Artifact parsing handles fenced blocks with `language:path` and supports multiple artifacts per response.
- [ ] State updates prevent unauthorized artifact overwrites (e.g., Coder cannot overwrite design docs).
- [ ] Completion check covers review checklist status, approved architecture, and test signals.

## Risks and checks

- [ ] Missing error handling around LLM calls, artifact parsing, and persistence is addressed.
- [ ] Reactive/free-form modes have deterministic fallback or selection rules (no ambiguous agent selection).
- [ ] Config validation ensures exactly three agents and required fields (mode, max-turns, artifacts).
- [ ] Output directory creation and file writes are safe and atomic to avoid partial artifacts.
- [ ] Logging/telemetry avoids leaking sensitive prompts or credentials.
