# Phase 13: V1 Code Removal

This phase performs a wholesale removal of all v1 code, documentation, and references, completing the migration to the v2 architecture.

## ⚠️ Critical Execution Notes (Added 2026-01-18)

**Task Interdependencies:**
Tasks 2 and 3 in this phase are TIGHTLY COUPLED and must be executed together:
- **Task 2 (Remove v1 adapter packages)** cannot proceed while cmd/run.go still imports v1 packages
- **Task 3 (Update cmd/ to use v2 only)** must be done BEFORE or SIMULTANEOUSLY with Task 2

**Recommended Execution Order:**
1. ✅ Task 1 (Identify v1 code) - COMPLETED
2. 🔄 Tasks 2+3 (Update cmd/ AND Remove v1 packages) - **Execute as ONE atomic operation**
3. ➡️ Task 4 onwards - Can proceed after 2+3 are complete

**pkg/log Decision: KEEP**
The pkg/log package is NOT v1-specific. It is shared logging infrastructure used by both v1 and v2 code paths, including cmd/root.go, cmd/bridge.go, cmd/providers.go, and other commands. It should remain in the codebase.

**High-Risk Tasks:**
- Merging cmd/run.go and cmd/run_v2.go requires careful handling to preserve all v2 functionality
- Deleting v1 packages MUST only happen after cmd/ updates are complete and verified
- Build and test verification is mandatory after each major change

## Tasks

- [x] Identify all v1 code locations:
  - List all pkg/ directories that are v1 (not under pkg/v2/)
  - List all cmd/ files using v1 packages
  - Identify v1-specific configuration files
  - Find v1 documentation in docs/
  - Locate v1 example configurations

  **Completed 2026-01-18**: Full audit documented in `Working/v1-code-audit.md`

  **V1 Package Directories (16 dirs, 62 Go files):**
  - pkg/adapters (19 files) - REMOVE (v2 equivalent exists)
  - pkg/agent (3 files) - REMOVE
  - pkg/artifact (4 files) - EVALUATE (may need porting)
  - pkg/client (2 files) - REMOVE
  - pkg/config (4 files) - REMOVE (v2 equivalent exists)
  - pkg/conversation (2 files) - REMOVE
  - pkg/errors (2 files) - REMOVE (v2 equivalent exists)
  - pkg/export (2 files) - EVALUATE (may need porting)
  - pkg/log (2 files) - EVALUATE (used by cmd/root.go)
  - pkg/logger (2 files) - REMOVE
  - pkg/metrics (4 files) - EVALUATE
  - pkg/middleware (4 files) - EVALUATE
  - pkg/orchestrator (2 files) - REMOVE (v2 has pkg/v2/manager)
  - pkg/ratelimit (4 files) - EVALUATE
  - pkg/tui (6 files) - REMOVE (v2 equivalent exists)
  - pkg/utils (2 files) - EVALUATE

  **cmd/ Files Using V1:**
  - cmd/run.go - uses pkg/adapters, pkg/orchestrator, pkg/tui (REPLACE)
  - cmd/run_test.go - associated tests (DELETE with run.go)

  **V1 Examples (21 files in examples/):**
  All YAML files in examples/ except examples/v2/

  **V1 Documentation:**
  ~20 files in docs/ reference v1 architecture
  docs/v2/ has 8 files to keep

  **V1 Tests:**
  - test/benchmark/ (4 files) - EVALUATE
  - test/integration/ (2 files) - REMOVE (uses v1 orchestrator)

  **internal/ Status:**
  - internal/bridge/ - KEEP (not v1-specific)
  - internal/branding/ - KEEP
  - internal/providers/ - KEEP
  - internal/registry/ - KEEP
  - internal/version/ - KEEP

  **Note:** pkg/providers/ does not exist (task list error) - providers are in internal/providers/

- [ ] Remove v1 adapter packages:

  **⚠️ BLOCKING DEPENDENCY: This task CANNOT be completed until "Update cmd/ to use v2 only" (task 3) is done first.**

  **Dependency Analysis (2026-01-18):**
  - cmd/run.go imports: pkg/adapters, pkg/agent, pkg/artifact, pkg/config, pkg/conversation, pkg/log, pkg/logger, pkg/orchestrator, pkg/tui
  - cmd/init.go imports: pkg/agent, pkg/config
  - cmd/export.go imports: pkg/agent, pkg/export
  - cmd/resume.go imports: pkg/conversation, pkg/log
  - cmd/bridge.go imports: pkg/log
  - cmd/providers.go imports: pkg/log
  - cmd/model_validation.go imports: pkg/log
  - test/integration/conversation_test.go imports: pkg/logger, pkg/orchestrator

  **pkg/log Status: KEEP** - This package is shared infrastructure used by both v1 and v2 code paths. It provides zerolog wrapper functions and is used by cmd/root.go, cmd/bridge.go, cmd/providers.go, cmd/resume.go, cmd/model_validation.go, and cmd/v2demo/main.go.

  **Packages to Remove (after cmd/ update):**
  - Remove pkg/adapters/ (v1 adapters - claude, gemini, qwen, etc.)
  - Remove pkg/client/ (v1 HTTP client)
  - Remove pkg/config/ (v1 configuration)
  - Remove pkg/orchestrator/ (v1 orchestrator)
  - Remove pkg/tui/ (v1 TUI - replaced by pkg/v2/tui/)
  - Remove pkg/agent/ (v1 agent interfaces)
  - Remove pkg/conversation/ (v1 conversation types)
  - Remove pkg/logger/ (duplicate logging package - used by v1 orchestrator)
  - Remove pkg/errors/ (v2 has pkg/v2/errors)
  - ~~Remove pkg/providers/~~ - **Does not exist; providers are in internal/providers/**
  - ~~Remove pkg/types/~~ - **Does not exist**
  - ~~Remove pkg/log/~~ - **KEEP: Shared logging infrastructure**

  **Packages to Evaluate (may need porting to v2):**
  - pkg/artifact - Artifact collection functionality (used by cmd/run.go)
  - pkg/export - Export functionality (used by cmd/export.go)
  - pkg/metrics - Prometheus metrics integration
  - pkg/middleware - Middleware system
  - pkg/ratelimit - Rate limiting
  - pkg/utils - Utility functions (token estimation)

- [ ] Update cmd/ to use v2 only:

  **⚠️ CRITICAL: This task and "Remove v1 adapter packages" (task 2) must be done together as a cohesive unit.**

  **Architecture Notes (2026-01-18):**
  - cmd/run.go defines `runCmd` and contains the v1 execution path
  - cmd/run_v2.go adds v2 flags to `runCmd` and provides `runV2Conversation()` function
  - The `shouldUseV2()` function in run_v2.go determines which path to take
  - cmd/run.go calls `runV2Conversation()` when v2 is enabled
  - These files are NOT independent - run_v2.go depends on runCmd from run.go

  **Migration Strategy:**
  1. **DO NOT simply delete run.go and rename run_v2.go** - this will break the build
  2. Instead, create a new merged run.go that:
     - Keeps the runCmd definition and flag setup from run.go
     - Removes the v1 execution path (runConversation function body that uses v1 packages)
     - Always routes to v2 execution (remove shouldUseV2 check, always use v2)
     - Removes v1-specific imports
  3. Delete the now-empty run_v2.go (merged into run.go)
  4. Delete run_test.go (v1 tests) and rename run_v2_test.go to run_test.go

  **Files to Update:**
  - cmd/run.go - Merge with run_v2.go, remove v1 imports and execution path
  - cmd/run_v2.go - Delete after merge
  - cmd/run_test.go - Replace with run_v2_test.go
  - cmd/init.go - Update to use pkg/v2/config and pkg/v2/core (or remove if v1-only)
  - cmd/export.go - Port to use v2 core or remove
  - cmd/resume.go - Port to use v2 persistence or remove
  - cmd/doctor.go - Already uses v2 packages, keep as-is
  - cmd/root.go - Keep pkg/log import (shared infrastructure)

- [ ] Remove v1 internal packages:
  - ~~Remove internal/bridge/~~ - **KEEP: Streaming bridge is shared, not v1-specific**
  - Update any shared internal/ code
  - **NOTE: All internal/ packages (bridge, branding, providers, registry, version) should be KEPT**

- [ ] Update main.go:
  - Ensure only v2 code paths
  - Remove v1 command registrations
  - Clean up imports

- [ ] Remove v1 documentation:
  - Remove docs/ files specific to v1 architecture
  - Keep docs/v2/ documentation
  - Update README.md to remove v1 references
  - Update CHANGELOG.md to note v1 removal

- [ ] Remove v1 example configurations:
  - Remove examples/ that use v1 format
  - Keep examples/v2/ configurations
  - Update any example references in docs

- [ ] Clean up test files:
  - Remove v1 test files (pkg/*/..._test.go for v1 packages)
  - Keep pkg/v2/**/*_test.go
  - Update integration tests

- [ ] Update go.mod and dependencies:
  - Run go mod tidy to remove unused dependencies
  - Verify no v1 package imports remain
  - Check for orphaned dependencies

- [ ] Restructure pkg/v2 to pkg/:
  - Move pkg/v2/adapters/ to pkg/adapters/
  - Move pkg/v2/core/ to pkg/core/
  - Move pkg/v2/config/ to pkg/config/
  - Move pkg/v2/errors/ to pkg/errors/
  - Move pkg/v2/events/ to pkg/events/
  - Move pkg/v2/manager/ to pkg/manager/
  - Move pkg/v2/persistence/ to pkg/persistence/
  - Move pkg/v2/pool/ to pkg/pool/
  - Move pkg/v2/tui/ to pkg/tui/
  - Update all import paths from pkg/v2/* to pkg/*
  - Remove empty pkg/v2/ directory

- [ ] Update all import statements:
  - Replace github.com/ASRagab/agentpipe/pkg/v2/* with github.com/ASRagab/agentpipe/pkg/*
  - Update cmd/ imports
  - Update internal/ imports
  - Update test imports

- [ ] Verify build and tests:
  - Run go build
  - Run go test ./...
  - Run golangci-lint
  - Verify TUI works
  - Test with example configurations

- [ ] Update CLAUDE.md:
  - Remove any v1/v2 distinction language
  - Update package paths in documentation
  - Simplify architecture documentation

- [ ] Final cleanup:
  - Remove any orphaned files
  - Update .gitignore if needed
  - Commit with descriptive message
  - Push to fork
