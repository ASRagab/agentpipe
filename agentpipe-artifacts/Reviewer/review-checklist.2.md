# Review Checklist — AgentPipe 3-Agent Orchestration

## References
- Architecture: `architecture-design.md` (phases, modes, artifact naming, success criteria)
- Implementation: `implementation-sketch.go` (orchestrator loop, state, artifact parsing, success checks)

## Architecture Decisions
- [ ] Orchestration flow maps to **Phase 1–6** in `architecture-design.md` and is explicitly represented in control flow (e.g., `Run` loop + phase gates).
- [ ] Conversation modes (**sequential**, **reactive**, **free-form**) are implemented per `architecture-design.md`; fallback behavior is documented when non-sequential is not fully supported.
- [ ] Artifact naming convention and storage paths align with `architecture-design.md` (e.g., `docs/design/...`, `docs/reviews/...`) and are enforced in artifact parsing.
- [ ] Success criteria (architecture approved, tests pass, reviewer checklist clear, agent confirmations) are encoded in `isFeatureComplete` per `architecture-design.md`.
- [ ] Cross-references between design, code, and review artifacts are preserved and traceable (e.g., state includes both design and review artifacts).

## Implementation Approach
- [ ] `Orchestrator.Run` implements agent turn sequencing and termination conditions as sketched in `implementation-sketch.go`.
- [ ] `getAgentByTurn` respects mode-specific logic; sequential rotation matches Architect → Coder → Reviewer in `architecture-design.md`.
- [ ] `buildPrompt` includes role instructions + history + relevant artifacts, matching the “Cross-References” guidance in `architecture-design.md`.
- [ ] `parseArtifacts` correctly extracts fenced code blocks (