# GenerateInstructions Injection - Implementation Plan

**Date:** 2026-01-18
**Branch:** `feature/artifact-collection`
**Status:** Ready for implementation

## Problem Statement

The `GenerateInstructions()` function exists in `pkg/artifact/types.go` but is never called. Agents don't receive artifact creation instructions on their first turn, only on subsequent turns via `GenerateContextHeader()` (which shows existing artifacts).

**Current behavior:**

- First turn: Agent gets no artifact instructions from orchestrator
- Subsequent turns: Agent gets context header showing existing artifacts (if any)

**Desired behavior:**

- First turn: Agent gets artifact creation instructions via `GenerateInstructions()`
- Subsequent turns: Agent gets both instructions AND context header

## Implementation Details

### Step 1: Add First-Turn Detection Method

**File:** `pkg/orchestrator/orchestrator.go`

Add a new method to detect if an agent has responded before:

```go
// isFirstTurnForAgent checks if this is the agent's first response in the conversation.
// Returns true if the agent has not yet sent any messages with Role="agent".
func (o *Orchestrator) isFirstTurnForAgent(agentID string) bool {
    o.mu.RLock()
    defer o.mu.RUnlock()

    for _, msg := range o.messages {
        // Check for prior responses FROM this agent (not system messages ABOUT them)
        if msg.AgentID == agentID && msg.Role == "agent" {
            return false // Agent has responded before
        }
    }
    return true // No prior responses = first turn
}
```

**Location:** Add after line ~900 (before `getAgentResponse`)

### Step 2: Inject Instructions on First Turn

**File:** `pkg/orchestrator/orchestrator.go`
**Location:** In `getAgentResponse()`, around line 935

Modify the existing artifact context injection block:

```go
// Get current messages
messages := o.getMessages()

// Inject artifact context if artifact instructions are enabled
o.mu.RLock()
artifactCfg := o.artifactConfig
collectedArtifacts := make([]artifact.Artifact, len(o.collectedArtifacts))
copy(collectedArtifacts, o.collectedArtifacts)
o.mu.RUnlock()

// NEW: Inject artifact creation instructions on first turn
if artifactCfg.Enabled && artifactCfg.InstructAgents {
    if o.isFirstTurnForAgent(a.GetID()) {
        instructMsg := agent.Message{
            AgentID:   "system",
            AgentName: "SYSTEM",
            Content:   artifact.GenerateInstructions(artifactCfg.OutputDir),
            Timestamp: time.Now().Unix(),
            Role:      "system",
        }
        messages = append([]agent.Message{instructMsg}, messages...)

        log.WithFields(map[string]interface{}{
            "agent_id":   a.GetID(),
            "agent_name": a.GetName(),
        }).Debug("injected artifact creation instructions for first turn")
    }
}

// EXISTING: Inject context header showing existing artifacts (unchanged)
if artifactCfg.Enabled && artifactCfg.InstructAgents && len(collectedArtifacts) > 0 {
    contextHeader := artifact.GenerateContextHeader(collectedArtifacts)
    if contextHeader != "" {
        contextMsg := agent.Message{
            AgentID:   "system",
            AgentName: "SYSTEM",
            Content:   contextHeader,
            Timestamp: time.Now().Unix(),
            Role:      "system",
        }
        messages = append([]agent.Message{contextMsg}, messages...)
    }
}
```

### Step 3: Add Tests

**File:** `pkg/orchestrator/orchestrator_test.go`

Add tests for:

1. `TestIsFirstTurnForAgent` - Verify detection logic
2. `TestArtifactInstructionsInjectedOnFirstTurn` - Verify injection happens
3. `TestArtifactInstructionsNotInjectedOnSubsequentTurns` - Verify no duplicate injection

```go
func TestIsFirstTurnForAgent(t *testing.T) {
    orch := NewOrchestrator(OrchestratorConfig{}, nil)

    // Initially, all agents are on first turn
    if !orch.isFirstTurnForAgent("agent-1") {
        t.Error("Expected first turn for new agent")
    }

    // Add a system message (should NOT affect first-turn detection)
    orch.messages = append(orch.messages, agent.Message{
        AgentID: "agent-1",
        Role:    "system",
        Content: "agent-1 has joined",
    })

    if !orch.isFirstTurnForAgent("agent-1") {
        t.Error("System messages should not count as agent responses")
    }

    // Add an agent response (should mark as NOT first turn)
    orch.messages = append(orch.messages, agent.Message{
        AgentID: "agent-1",
        Role:    "agent",
        Content: "Hello, I am agent-1",
    })

    if orch.isFirstTurnForAgent("agent-1") {
        t.Error("Expected NOT first turn after agent response")
    }

    // Different agent should still be on first turn
    if !orch.isFirstTurnForAgent("agent-2") {
        t.Error("Different agent should be on first turn")
    }
}
```

### Step 4: Verify Integration

Run the collaborative planning test to verify instructions are injected:

```bash
./agentpipe run -c examples/collaborative-planning-test.yaml
```

Expected: Agents should receive artifact instructions on their first turn and create artifacts correctly.

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/orchestrator/orchestrator.go` | Add `isFirstTurnForAgent()` method, modify `getAgentResponse()` |
| `pkg/orchestrator/orchestrator_test.go` | Add tests for first-turn detection and injection |

## Testing Checklist

- [ ] `go build -o agentpipe .` passes
- [ ] `go test -race ./pkg/orchestrator/...` passes
- [ ] `go test -race ./pkg/artifact/...` passes (existing tests still pass)
- [ ] `golangci-lint run --timeout=5m` passes
- [ ] Manual test with `examples/collaborative-planning-test.yaml` shows instructions injected

## Edge Cases

1. **Agent re-joining conversation** - If an agent crashes and re-joins, they may get instructions again. This is acceptable since instructions are idempotent.

2. **InstructAgents=false** - No instructions injected at all (existing behavior preserved)

3. **Artifacts disabled** - No instructions injected (existing behavior preserved)

4. **Empty outputDir** - `GenerateInstructions()` handles this gracefully (uses current dir)

## Rollback Plan

If issues arise, revert the changes to `getAgentResponse()` - the `isFirstTurnForAgent()` method is side-effect-free.

## Success Criteria

1. Agents receive artifact creation instructions on their first turn
2. Instructions are NOT duplicated on subsequent turns
3. Existing artifact context header continues to work
4. All existing tests pass
5. No performance regression (one extra loop through messages per turn)

---

**Implementation Time Estimate:** ~30 minutes
**Risk Level:** Low (additive change, existing behavior unchanged)
