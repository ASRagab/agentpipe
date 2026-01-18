---
type: analysis
title: V1 Code Location Audit
created: 2026-01-18
tags:
  - migration
  - v1-removal
  - phase-13
related:
  - "[[Phase-13-V1-Code-Removal]]"
---

# V1 Code Location Audit

This document identifies all v1 code locations that need to be removed as part of Phase 13: V1 Code Removal.

## Summary

| Category | V1 Items | V2 Items | Notes |
|----------|----------|----------|-------|
| pkg/ directories | 16 | 10 | V1 at pkg/*, V2 at pkg/v2/* |
| Go files in pkg/ | 62 | 78 | V2 has more test coverage |
| Example configs | 21 | 8 | V1 examples use old schema |
| Documentation files | ~20 | 8 | Many docs reference v1 architecture |
| cmd/ files using v1 | 1 (run.go) | 4 | run.go uses v1, run_v2.go uses v2 |
| Test directories | 2 | 1 (pkg/v2/e2e) | test/integration + test/benchmark use v1 |

## V1 Package Directories (pkg/*)

These directories contain v1 code and should be removed:

| Directory | File Count | Status | Notes |
|-----------|------------|--------|-------|
| pkg/adapters | 19 | REMOVE | V2 has pkg/v2/adapters (20 files) |
| pkg/agent | 3 | REMOVE | Interface definitions only |
| pkg/artifact | 4 | EVALUATE | May not have v2 equivalent |
| pkg/client | 2 | REMOVE | HTTP client for v1 |
| pkg/config | 4 | REMOVE | V2 has pkg/v2/config (5 files) |
| pkg/conversation | 2 | REMOVE | Merged into v2 core |
| pkg/errors | 2 | REMOVE | V2 has pkg/v2/errors |
| pkg/export | 2 | EVALUATE | May need to port to v2 |
| pkg/log | 2 | EVALUATE | Used by cmd/root.go |
| pkg/logger | 2 | REMOVE | Duplicate with pkg/log |
| pkg/metrics | 4 | EVALUATE | Prometheus metrics |
| pkg/middleware | 4 | EVALUATE | Middleware system |
| pkg/orchestrator | 2 | REMOVE | V2 has pkg/v2/manager |
| pkg/ratelimit | 4 | EVALUATE | Rate limiting |
| pkg/tui | 6 | REMOVE | V2 has pkg/v2/tui (12 files) |
| pkg/utils | 2 | EVALUATE | Utility functions |

### V1 Packages Analysis

**Definitely Remove (have v2 equivalents):**
- pkg/adapters -> pkg/v2/adapters
- pkg/config -> pkg/v2/config
- pkg/orchestrator -> pkg/v2/manager
- pkg/tui -> pkg/v2/tui
- pkg/errors -> pkg/v2/errors
- pkg/conversation -> pkg/v2/core
- pkg/agent -> merged into v2 adapters
- pkg/client -> v2 uses different HTTP approach
- pkg/logger -> consolidate with pkg/log

**Need Evaluation (may need porting):**
- pkg/artifact - artifact collection functionality
- pkg/export - export functionality
- pkg/log - logging (used by root.go)
- pkg/metrics - Prometheus metrics integration
- pkg/middleware - middleware system
- pkg/ratelimit - rate limiting
- pkg/utils - utility functions

## V1 cmd/ Files

Files using v1 packages:

| File | V1 Imports | Action |
|------|------------|--------|
| cmd/run.go | pkg/adapters, pkg/orchestrator, pkg/tui | REPLACE with v2 or DELETE |
| cmd/run_test.go | Associated tests | DELETE with run.go |

Files already using v2:

| File | V2 Imports | Action |
|------|------------|--------|
| cmd/run_v2.go | pkg/v2/* | RENAME to run.go |
| cmd/run_v2_test.go | pkg/v2/* | RENAME to run_test.go |
| cmd/run_v2_integration_test.go | pkg/v2/* | RENAME |
| cmd/doctor.go | pkg/v2/* | KEEP (update imports after restructure) |
| cmd/version.go | pkg/v2/* | KEEP (update imports after restructure) |

Other cmd/ files needing update:
- cmd/root.go - uses pkg/log (needs evaluation)
- cmd/bridge.go - internal/bridge (keep as-is)
- cmd/providers.go - internal/providers (keep as-is)

## V1 Example Configurations

Files in examples/ using v1 format (21 files):
- aider-coding.yaml
- aider-team-coding.yaml
- amp-coding.yaml
- artifact-test.yaml
- brainstorm.yaml
- claude-coding.yaml
- codex-brainstorm.yaml
- collaborative-planning-test.yaml
- config-hot-reload-demo.yaml
- continue-coding.yaml
- continue-team-coding.yaml
- copilot-dev.yaml
- cursor-brainstorm.yaml
- cursor-solo.yaml
- debate.yaml
- middleware.yaml
- openrouter-conversation.yaml
- openrouter-solo.yaml
- prometheus-metrics.yaml
- qoder-coding.yaml
- simple-conversation.yaml

V2 examples to keep (examples/v2/):
- brainstorm.yaml
- code-review.yaml
- demo-config.yaml
- legacy-config.yaml
- minimal.yaml
- multi-model.yaml
- research.yaml
- two-agents.yaml

## V1 Documentation

Files in docs/ that reference v1 architecture:
- architecture.md
- architecture-v2-proposal.md (rename after migration)
- refactoring-examples.md
- refactoring-roadmap.md
- v2-*.md files (many can be consolidated into docs/v2/)

Files to keep (docs/v2/):
- README.md
- adapters.md
- architecture.md
- configuration.md
- migration.md
- quickstart.md
- troubleshooting.md
- tui.md

## V1 Test Files

| Directory | Files | Action |
|-----------|-------|--------|
| test/benchmark/ | 4 files | EVALUATE - may port to v2 |
| test/integration/ | 2 files | REMOVE - uses v1 orchestrator |

V1 test imports found:
- test/benchmark/orchestrator_bench_test.go - uses pkg/orchestrator
- test/integration/error_scenarios_test.go - uses pkg/orchestrator
- test/integration/conversation_test.go - uses pkg/orchestrator

## internal/ Directory

| Directory | Status | Notes |
|-----------|--------|-------|
| internal/bridge/ | KEEP | Streaming bridge, not v1-specific |
| internal/branding/ | KEEP | Branding assets |
| internal/providers/ | KEEP | Provider pricing data |
| internal/registry/ | KEEP | Agent registry |
| internal/version/ | KEEP | Version management |

## Files Using pkg/log (needs evaluation)

cmd/root.go imports pkg/log - need to determine if this should:
1. Be kept and shared
2. Be replaced with zerolog directly
3. Be moved to internal/log

## Migration Order

Recommended order for Phase 13 execution:

1. **Identify** (this document) - DONE
2. **Remove v1 adapter packages** - Delete pkg/adapters, pkg/client, pkg/agent
3. **Update cmd/** - Replace run.go with run_v2.go, update imports
4. **Remove v1 internal packages** - Only if not shared
5. **Update main.go** - Should be minimal changes
6. **Remove v1 documentation** - Move relevant docs to docs/v2/, delete others
7. **Remove v1 example configurations** - Delete, keep examples/v2/
8. **Clean up test files** - Delete v1 tests, update integration tests
9. **Update go.mod** - Run go mod tidy
10. **Restructure pkg/v2 to pkg/** - Move all pkg/v2/* to pkg/*
11. **Update all import statements** - Replace pkg/v2/* with pkg/*
12. **Verify build and tests** - Full validation
13. **Update CLAUDE.md** - Remove v1/v2 distinction
14. **Final cleanup** - Orphaned files, .gitignore, commit
