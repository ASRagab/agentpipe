# Multi-Agent Collaboration Test Design

**Date:** 2026-01-17
**Type:** Quick Validation Test
**Duration:** ~5 minutes (3-5 turns)
**Status:** Ready to Execute

## Overview

Test how three agents (Claude-Architect, Gemini-Coder, Codex-Reviewer) collaborate to synthesize an implementation plan for enabling agentic collaborative coding in AgentPipe.

**Primary Goal:** Validate that multiple agents can create artifacts, build on each other's work, and maintain coherent collaboration.

**Not In Scope:** Actual implementation of collaborative coding features (this is a planning test only).

## Test Objectives

### Primary Objectives

1. **Artifact Creation** - Verify multiple agents can create artifacts using the ```language:filename syntax
2. **Sequential Collaboration** - Confirm agents read and build on previous responses
3. **Artifact Organization** - Validate artifact storage in agent-specific directories
4. **Role Adherence** - Ensure agents stick to assigned roles (architect/coder/reviewer)

### Secondary Objectives

1. **Turn-Taking Quality** - Observe how effectively agents listen and contribute
2. **Planning Coherence** - Assess whether the combined output forms a logical plan
3. **System Stability** - Confirm no crashes, timeouts, or parsing errors

## Test Scenario: Collaborative Feature Builder

**Prompt:** "Design how AgentPipe could orchestrate 3 agents (architect, coder, reviewer) to collaboratively build a new feature end-to-end."

### Agent Roles

1. **Architect (Claude Sonnet 4.5)**
   - Focus: High-level architecture design
   - Artifact: `architecture-design.md`
   - Responsibilities: System design, component interactions, data flow

2. **Coder (Gemini 2.5 Pro)**
   - Focus: Implementation sketch with pseudocode
   - Artifact: `implementation-sketch.go`
   - Responsibilities: Code structure, function signatures, key algorithms

3. **Reviewer (GPT-5.2 Codex)**
   - Focus: Review checklist and quality criteria
   - Artifact: `review-checklist.md`
   - Responsibilities: Validation criteria, risks, testing strategy

### Expected Collaboration Flow

```
Turn 1: Architect → Creates architecture-design.md
  ↓
Turn 2: Coder → Reads architecture, creates implementation-sketch.go
  ↓
Turn 3: Reviewer → Reads both, creates review-checklist.md
  ↓
Turn 4-5: Iteration/refinement based on feedback
```

## Configuration

**File:** `examples/collaborative-planning-test.yaml`

**Key Settings:**

- Mode: `round-robin` (ensures equal turns)
- Max Turns: `5` (~1-2 contributions per agent)
- Timeout: `180s` per turn
- Artifacts: Enabled with agent instructions
- Output: `./agentpipe-artifacts`

**Models:**

- Claude: `claude-sonnet-4-5@20250929`
- Gemini: `gemini-2.5-pro`
- Codex: `gpt-5.2-codex`

## Execution Plan

### Pre-Test Setup

```bash
# 1. Check current artifact state
ls -la agentpipe-artifacts/ 2>/dev/null || echo "Clean start"

# 2. Verify agent availability
agentpipe doctor

# 3. Confirm config exists
cat examples/collaborative-planning-test.yaml
```

### Running the Test

```bash
agentpipe run -t -c examples/collaborative-planning-test.yaml
```

### During Execution - Observation Points

**Turn 1 (Claude-Architect):**

- [ ] Creates `architecture-design.md` artifact?
- [ ] Includes system components and interactions?
- [ ] Design is well-structured and concise?

**Turn 2 (Gemini-Coder):**

- [ ] References the architecture explicitly?
- [ ] Creates `implementation-sketch.go` artifact?
- [ ] Code structure aligns with architecture?

**Turn 3 (Codex-Reviewer):**

- [ ] References both previous artifacts?
- [ ] Creates `review-checklist.md` artifact?
- [ ] Checklist covers architecture AND implementation?

**Turns 4-5 (Iteration):**

- [ ] Agents refine based on feedback?
- [ ] New artifacts or updates to existing ones?
- [ ] Convergence toward coherent plan?

**TUI Indicators:**

- [ ] Artifact parsing messages visible?
- [ ] No error messages or timeouts?
- [ ] All agents complete their turns?

### Post-Test Analysis

```bash
# 1. Check artifact directory structure
tree agentpipe-artifacts/

# Expected structure:
# agentpipe-artifacts/
# ├── Architect/
# │   └── architecture-design.md
# ├── Coder/
# │   └── implementation-sketch.go
# └── Reviewer/
#     └── review-checklist.md

# 2. Review artifacts
cat agentpipe-artifacts/Architect/architecture-design.md
cat agentpipe-artifacts/Coder/implementation-sketch.go
cat agentpipe-artifacts/Reviewer/review-checklist.md

# 3. Check conversation log
cat ~/.agentpipe/chats/[latest-session].log
```

## Success Criteria

### Must Have (Test Passes)

- ✅ All 3 agents create their designated artifacts
- ✅ Artifacts saved to correct directories without errors
- ✅ No crashes, timeouts, or parsing failures
- ✅ Conversation completes all planned turns

### Should Have (Quality Indicators)

- ✅ Later agents reference earlier artifacts
- ✅ Architecture → Implementation flow is logical
- ✅ Review checklist covers both architecture and code
- ✅ Combined output forms coherent plan

### Could Have (Bonus)

- ✅ Agents iterate and refine based on feedback
- ✅ Disagreement handled constructively
- ✅ Artifact versioning triggered (if agents create duplicates)

## Failure Indicators

**Critical Failures:**

- Missing artifacts (agents didn't create them)
- Parse errors or file write failures
- Agent timeouts or crashes
- Artifacts in wrong directories

**Quality Failures:**

- Agents ignore each other (no cross-references)
- Incoherent or contradictory outputs
- Agents don't follow their assigned roles
- Generic responses without specificity

## Documentation Requirements

**Post-Test Report Should Include:**

1. Artifact directory tree (screenshot or text output)
2. Quality rating for each artifact (1-5 scale)
3. Collaboration patterns observed
4. Any failures or unexpected behaviors
5. Recommendations for next tests

**Quality Rating Scale:**

- 5 = Excellent (specific, coherent, builds on others)
- 4 = Good (solid work, minor issues)
- 3 = Adequate (functional but generic)
- 2 = Poor (missing key elements)
- 1 = Failed (didn't create artifact or unusable)

## Future Test Scenarios

**Note:** Save these for later iterations:

**Scenario A - Architectural Challenge:**
"Design how AgentPipe could enable agents to work on the same codebase simultaneously with conflict resolution"

- Tests: Complex problem-solving, deeper collaboration
- Duration: 10-15 turns

**Scenario B - Concrete Feature:**
"Design how AgentPipe could support collaborative debugging sessions where agents help each other fix bugs"

- Tests: Disagreement handling, consensus building
- Duration: 8-10 turns

## Notes

- This is a **planning test**, not an implementation test
- Focus on collaboration quality and artifact system validation
- Keep scope manageable (5 turns max) for this first run
- Document observations for improving future tests
- If successful, expand to scenarios A and B
