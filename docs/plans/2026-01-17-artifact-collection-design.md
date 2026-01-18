# Artifact Collection Design

**Date:** 2026-01-17
**Status:** Approved
**Author:** Brainstorming session

## Overview

Enable agents to create files during conversation using a special fenced code block syntax. Artifacts are extracted in real-time and saved to a project-relative workspace.

## Problem Statement

Currently AgentPipe captures conversation text but produces no tangible outputs. Agents discuss, debate, and propose code/docs, but these remain embedded in the conversation transcript. Users must manually extract and save valuable content.

## Solution

Introduce an artifact extraction system that:
1. Recognizes a filename convention in fenced code blocks
2. Extracts and saves artifacts in real-time
3. Organizes artifacts by agent in a workspace folder
4. Provides visibility into artifact creation via TUI feedback

## Design Decisions

### Syntax Convention

Agents signal artifacts using fenced code blocks with a filename suffix:

```
```yaml:config/api-spec.yaml
openapi: 3.0.0
info:
  title: My API
paths:
  /users:
    get:
      summary: List users
```​
```

**Format:** ` ```language:path/to/filename.ext`

- `language` - syntax highlighting hint (yaml, sql, go, markdown, etc.)
- `path/to/filename.ext` - relative path within agent's workspace

### Workspace Structure

```
./agentpipe-artifacts/           # Default location (project-relative)
├── alice/                       # Agent subdirectory
│   ├── api-spec.yaml           # First artifact
│   └── api-spec.yaml.2         # Revised version (same filename)
├── bob/
│   ├── api-spec.yaml           # Bob's version (no conflict with Alice)
│   └── schema.sql
└── charlie/
    └── docs/
        └── plan.md             # Nested paths supported
```

**Location:** `./agentpipe-artifacts/` by default, configurable via `--output-dir` flag.

### Timing: Real-Time Extraction

Artifacts are saved immediately as each agent responds:
- User sees progress during conversation
- Artifacts survive conversation interrupts (Ctrl+C)
- TUI displays `[Artifact saved: alice/api-spec.yaml]` inline

### Conflict Resolution

1. **Cross-agent:** Agent subdirectories prevent conflicts (Alice and Bob can both have `api-spec.yaml`)
2. **Same-agent:** Version suffix added for repeated filenames (`.2`, `.3`, etc.)

### Agent Prompt Instructions

Add to the structured prompt (in `buildPrompt()`):

```
ARTIFACT CREATION:
To create a saveable artifact, use fenced code blocks with a filename:
  ```language:path/to/filename.ext
  content here
  ```
Artifacts will be saved to the workspace automatically.
```

This informs agents they can create persistent outputs.

## Implementation Plan

### Phase 1: Core Extraction (pkg/artifact/)

1. **Create `pkg/artifact/` package**
   - `parser.go` - Extract artifacts from message content
   - `writer.go` - Save artifacts to filesystem
   - `types.go` - Artifact struct definition

2. **Artifact struct:**
   ```go
   type Artifact struct {
       AgentID   string
       AgentName string
       Language  string
       Filename  string    // Original path from code block
       Content   string
       Timestamp time.Time
       Version   int       // For conflict resolution
   }
   ```

3. **Parser logic:**
   - Regex to match ` ```language:filename` pattern
   - Extract content between fences
   - Return slice of artifacts + cleaned message content

### Phase 2: Orchestrator Integration

1. **Add artifact extraction to message processing**
   - After agent responds, parse for artifacts
   - Save each artifact via writer
   - Emit artifact event (for TUI display)

2. **Configuration additions:**
   ```yaml
   artifacts:
     enabled: true
     output_dir: ./agentpipe-artifacts  # default
   ```

3. **CLI flags:**
   - `--output-dir` - Override artifact directory
   - `--no-artifacts` - Disable artifact extraction

### Phase 3: TUI Display

1. **Artifact notification in chat:**
   - Show `[Artifact saved: agent/path/file.ext]` as system message
   - Color-coded (green for success, red for write errors)

2. **Optional: Artifacts panel**
   - Future enhancement: dedicated panel showing all artifacts
   - Click to preview content

### Phase 4: Agent Prompt Updates

1. **Modify `buildPrompt()` in each adapter**
   - Add artifact instruction section
   - Consistent across all adapters (claude, gemini, cursor, etc.)

2. **Make instructions configurable**
   - Some users may not want agents creating artifacts unprompted
   - Config option: `artifacts.instruct_agents: true/false`

## File Changes Required

| File | Change |
|------|--------|
| `pkg/artifact/parser.go` | NEW - Parse artifacts from content |
| `pkg/artifact/writer.go` | NEW - Write artifacts to filesystem |
| `pkg/artifact/types.go` | NEW - Type definitions |
| `pkg/orchestrator/orchestrator.go` | Integrate artifact extraction |
| `pkg/adapters/*.go` | Add artifact instructions to prompts |
| `pkg/config/config.go` | Add artifacts config section |
| `pkg/tui/enhanced.go` | Display artifact notifications |
| `cmd/run.go` | Add --output-dir, --no-artifacts flags |

## Testing Strategy

1. **Unit tests for parser**
   - Various code block formats
   - Edge cases (no filename, nested blocks, malformed)
   - Multiple artifacts in one message

2. **Integration tests**
   - End-to-end artifact creation
   - Conflict resolution (versioning)
   - Directory creation

3. **Manual testing**
   - Run conversation with artifact-aware prompts
   - Verify TUI feedback
   - Check workspace structure

## Future Enhancements

1. **Model C (Synthesis):** Auto-generate summary docs from conversation
2. **Model D (Pipeline):** Pass artifacts between agents as inputs
3. **Artifact preview:** View artifacts in TUI before saving
4. **Artifact diff:** Show changes when agent revises a file
5. **Git integration:** Auto-commit artifacts with meaningful messages

## Open Questions (Resolved)

| Question | Decision |
|----------|----------|
| How to signal artifacts? | Filename in fence: ` ```lang:file` |
| Where to save? | Project-relative `./agentpipe-artifacts/` |
| When to save? | Real-time (immediate) |
| Handle conflicts? | Agent subdirectories + version suffix |
| Inform agents? | Yes, add to prompt instructions |
