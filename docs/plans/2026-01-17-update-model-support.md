# Update Model Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Update AgentPipe's model database to reflect the latest models from Anthropic, Google (Gemini/VertexAI), and OpenAI, and fix outdated model references in examples.

**Architecture:** AgentPipe uses an embedded `providers.json` file fetched from Catwalk's GitHub repository. The update script fetches individual provider JSON files and consolidates them. Example YAML configs and documentation reference specific model IDs that need to stay current.

**Tech Stack:** Go 1.24+, embedded JSON via `//go:embed`, HTTP fetcher for Catwalk API

---

## Summary of Changes Required

### Provider Data Changes (via Catwalk fetch)

| Provider | Current Count | New Count | New Models |
|----------|--------------|-----------|------------|
| Anthropic | 9 | 10 | `claude-opus-4-5-20251101` ($5/$25) |
| Gemini | 2 | 4 | `gemini-3-pro-preview`, `gemini-3-flash-preview` |
| OpenAI | 11 | 18 | `gpt-5.2`, `gpt-5.1-codex*` variants |
| VertexAI | 2 | 7 | Gemini 3 previews + Claude models |

### Example File Fixes

| File | Current Model | Correct Model |
|------|--------------|---------------|
| `examples/simple-conversation.yaml:9` | `claude-4-sonnet` | `claude-sonnet-4-5-20250929` |
| `examples/brainstorm.yaml:17` | `claude-3-opus` | `claude-opus-4-5-20251101` |
| `docs/troubleshooting.md:341` | `claude-3-haiku` | `claude-3-5-haiku-20241022` |

---

## Task 1: Fetch Latest Provider Data from Catwalk

**Files:**

- Modify: `internal/providers/providers.json` (auto-generated)

**Step 1: Verify update script exists and is runnable**

Run: `ls -la scripts/update-providers.go`
Expected: File exists with `// +build ignore` header

**Step 2: Run the update script**

Run: `cd /Users/ahmad.ragab/Dev/tools/agentpipe && go run scripts/update-providers.go`
Expected: Output showing "Fetched 16 providers" and "Successfully wrote internal/providers/providers.json"

**Step 3: Verify the new provider counts**

Run: `cat internal/providers/providers.json | jq '.providers[] | select(.id == "anthropic" or .id == "gemini" or .id == "openai" or .id == "vertexai") | {id, model_count: (.models | length)}'`
Expected:

```json
{"id": "anthropic", "model_count": 10}
{"id": "gemini", "model_count": 4}
{"id": "openai", "model_count": 18}
{"id": "vertexai", "model_count": 7}
```

**Step 4: Verify new Anthropic model exists**

Run: `cat internal/providers/providers.json | jq '.providers[] | select(.id == "anthropic") | .models[] | select(.id | contains("opus-4-5"))'`
Expected: Model `claude-opus-4-5-20251101` with pricing $5/$25

**Step 5: Verify new Gemini models exist**

Run: `cat internal/providers/providers.json | jq '.providers[] | select(.id == "gemini") | .models[].id'`
Expected: Should include `gemini-3-pro-preview` and `gemini-3-flash-preview`

**Step 6: Commit the updated providers.json**

```bash
git add internal/providers/providers.json
git commit -m "chore: update providers.json with latest models from Catwalk

- Anthropic: Add claude-opus-4-5-20251101 (10 models total)
- Gemini: Add gemini-3-pro-preview, gemini-3-flash-preview (4 models total)
- OpenAI: Add gpt-5.2, gpt-5.1-codex variants (18 models total)
- VertexAI: Add Gemini 3 previews + Claude models (7 models total)"
```

---

## Task 2: Fix simple-conversation.yaml Model Reference

**Files:**

- Modify: `examples/simple-conversation.yaml:9`

**Step 1: Read current file to confirm line content**

Run: `sed -n '9p' examples/simple-conversation.yaml`
Expected: `model: claude-4-sonnet`

**Step 2: Fix the invalid model reference**

Replace line 9 from:

```yaml
    model: claude-4-sonnet
```

to:

```yaml
    model: claude-sonnet-4-5-20250929
```

**Step 3: Verify the change**

Run: `sed -n '9p' examples/simple-conversation.yaml`
Expected: `model: claude-sonnet-4-5-20250929`

**Step 4: Commit**

```bash
git add examples/simple-conversation.yaml
git commit -m "fix(examples): update simple-conversation.yaml to valid model ID

- Changed claude-4-sonnet to claude-sonnet-4-5-20250929"
```

---

## Task 3: Fix brainstorm.yaml Model Reference

**Files:**

- Modify: `examples/brainstorm.yaml:17`

**Step 1: Read current file to confirm line content**

Run: `sed -n '17p' examples/brainstorm.yaml`
Expected: `model: claude-3-opus`

**Step 2: Fix the outdated model reference**

Replace line 17 from:

```yaml
    model: claude-3-opus
```

to:

```yaml
    model: claude-opus-4-5-20251101
```

**Step 3: Verify the change**

Run: `sed -n '17p' examples/brainstorm.yaml`
Expected: `model: claude-opus-4-5-20251101`

**Step 4: Commit**

```bash
git add examples/brainstorm.yaml
git commit -m "fix(examples): update brainstorm.yaml to latest Opus model

- Changed claude-3-opus to claude-opus-4-5-20251101"
```

---

## Task 4: Fix troubleshooting.md Model Reference

**Files:**

- Modify: `docs/troubleshooting.md:341`

**Step 1: Read current context around line 341**

Run: `sed -n '338,344p' docs/troubleshooting.md`
Expected: See `claude-3-haiku` in context

**Step 2: Fix the outdated model reference**

Replace `claude-3-haiku` with `claude-3-5-haiku-20241022` on line 341

**Step 3: Verify the change**

Run: `sed -n '341p' docs/troubleshooting.md`
Expected: `model: claude-3-5-haiku-20241022`

**Step 4: Commit**

```bash
git add docs/troubleshooting.md
git commit -m "fix(docs): update troubleshooting.md to valid Haiku model ID

- Changed claude-3-haiku to claude-3-5-haiku-20241022"
```

---

## Task 5: Run Tests and Build Verification

**Files:**

- Test: All existing tests

**Step 1: Run linting**

Run: `golangci-lint run --timeout=5m`
Expected: No new errors (some pre-existing may appear)

**Step 2: Run tests with race detection**

Run: `go test -v -race ./...`
Expected: All tests pass

**Step 3: Build the binary**

Run: `go build -o agentpipe .`
Expected: Build succeeds with no errors

**Step 4: Verify provider registry loads correctly**

Run: `./agentpipe providers list | head -20`
Expected: Shows updated provider list with new model counts

**Step 5: Verify specific models are accessible**

Run: `./agentpipe providers show anthropic | grep -A2 "claude-opus-4-5-20251101"`
Expected: Shows the new Opus 4.5 model with $5/$25 pricing

---

## Task 6: Update CHANGELOG.md

**Files:**

- Modify: `CHANGELOG.md`

**Step 1: Add entry for model updates**

Add under the next unreleased version section (or create one):

```markdown
### Changed
- Updated `providers.json` with latest models from Catwalk
  - Anthropic: Added `claude-opus-4-5-20251101` ($5/$25 per 1M tokens)
  - Gemini: Added `gemini-3-pro-preview`, `gemini-3-flash-preview`
  - OpenAI: Added `gpt-5.2`, `gpt-5.1-codex` variants (18 models total)
  - VertexAI: Added Gemini 3 previews and Claude models

### Fixed
- Fixed invalid model references in example configs
  - `simple-conversation.yaml`: `claude-4-sonnet` → `claude-sonnet-4-5-20250929`
  - `brainstorm.yaml`: `claude-3-opus` → `claude-opus-4-5-20251101`
  - `troubleshooting.md`: `claude-3-haiku` → `claude-3-5-haiku-20241022`
```

**Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG with model updates"
```

---

## Task 7: Final Verification and Summary Commit (Optional)

**Step 1: Review all changes**

Run: `git log --oneline -10`
Expected: See 5-6 commits for the model updates

**Step 2: Verify no model references use old format**

Run: `grep -rn "claude-[0-9]-" examples/ docs/ --include="*.yaml" --include="*.md" | grep -v "claude-3-5" | grep -v "claude-3-7"`
Expected: No results (all old format references fixed)

**Step 3: Run full test suite one more time**

Run: `go test -v -race ./...`
Expected: All tests pass

---

## Notes

### Model ID Format Reference

| Provider | Format | Example |
|----------|--------|---------|
| Anthropic | `claude-{variant}-{version}-{date}` | `claude-opus-4-5-20251101` |
| Gemini | `gemini-{version}` | `gemini-2.5-pro` |
| OpenAI | `gpt-{version}` or `o{version}` | `gpt-5.1-codex`, `o4-mini` |
| VertexAI | Same as source + `@{date}` suffix | `claude-sonnet-4-5@20250929` |

### Rollback Instructions

If issues arise, revert with:

```bash
git revert HEAD~5..HEAD  # Revert last 5 commits
```

Or restore old providers.json:

```bash
git checkout HEAD~6 -- internal/providers/providers.json
```

### Future Updates

To update models again in the future:

```bash
go run scripts/update-providers.go
```

Or via CLI:

```bash
agentpipe providers update
```
