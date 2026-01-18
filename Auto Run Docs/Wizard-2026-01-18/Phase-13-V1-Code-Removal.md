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

- [x] Remove v1 adapter packages:

  **Completed 2026-01-18**: Removed all v1 packages as part of atomic operation with task 3.

  **Packages Removed:**
  - pkg/adapters/ (20 files) - v1 CLI adapters
  - pkg/agent/ (4 files) - v1 agent interfaces
  - pkg/client/ (3 files) - v1 HTTP client
  - pkg/config/ (5 files) - v1 configuration
  - pkg/orchestrator/ (3 files) - v1 orchestrator
  - pkg/tui/ (7 files) - v1 TUI
  - pkg/conversation/ (2 files) - v1 conversation types
  - pkg/logger/ (2 files) - duplicate logging
  - pkg/errors/ (2 files) - v1 errors

  **Packages KEPT:**
  - pkg/log/ - Shared logging infrastructure (used by cmd/root.go, bridge.go, etc.)
  - pkg/artifact/, pkg/export/, pkg/metrics/, pkg/middleware/, pkg/ratelimit/, pkg/utils/ - May need porting in later phase

- [x] Update cmd/ to use v2 only:

  **Completed 2026-01-18**: cmd/run.go now uses v2 packages exclusively.

  **Changes Made:**
  - cmd/run.go - Already updated to import only pkg/v2/* packages
  - cmd/run_v2.go - Deleted (duplicate code causing build failure)
  - cmd/run_v2_test.go - Deleted (obsolete v2 flag tests)
  - cmd/run_test.go - Created new test file without v2 flag references
  - cmd/run_v2_integration_test.go - Updated to use renamed helper functions
  - cmd/init.go, cmd/export.go, cmd/resume.go - Deleted (v1-only commands)

  **Build & Test Status:**
  - `go build` passes
  - `go test ./cmd/...` passes
  - Main packages (pkg/v2/*, internal/*) tests pass
  - Some peripheral packages (pkg/export, pkg/middleware) fail due to v1 dependencies (to be addressed in later task)

- [x] Remove v1 internal packages:
  - ~~Remove internal/bridge/~~ - **KEEP: Streaming bridge is shared, not v1-specific**
  - Update any shared internal/ code
  - **NOTE: All internal/ packages (bridge, branding, providers, registry, version) should be KEPT**

  **Completed 2026-01-18**: Verified all internal/ packages are clean and should be kept.

  **Verified Packages:**
  - internal/bridge/ - KEEP: Streaming bridge, no v1 dependencies
  - internal/branding/ - KEEP: Logo/branding assets, no v1 dependencies
  - internal/providers/ - KEEP: Provider registry, only depends on pkg/log (shared infrastructure)
  - internal/registry/ - KEEP: Version registry, no v1 dependencies
  - internal/version/ - KEEP: Version info, no v1 dependencies

  **pkg/log Status:** KEEP as shared logging infrastructure (used by internal/providers and cmd/)

- [x] Update main.go:
  - Ensure only v2 code paths
  - Remove v1 command registrations
  - Clean up imports

  **Completed 2026-01-18**: Verified main.go and cmd/ are v2-only.

  **Verification Results:**
  - main.go: Clean (only imports cmd package)
  - cmd/root.go: Uses shared infrastructure (pkg/log, internal/bridge, internal/version)
  - All cmd/*.go files: Only use pkg/v2/* and internal/* packages
  - pkg/log: KEPT as shared logging infrastructure (per Phase-13 notes)

  **Commands Registered (all v2):**
  - runCmd - Uses pkg/v2/*
  - doctorCmd - Uses pkg/v2/adapters, pkg/v2/config, pkg/v2/core
  - versionCmd - Uses pkg/v2/adapters
  - bridgeCmd - Uses internal/bridge
  - providersCmd - Uses internal/providers
  - agentsCmd - Uses internal/registry

  **Previously Removed v1 Commands:**
  - cmd/init.go (deleted in task 3)
  - cmd/export.go (deleted in task 3)
  - cmd/resume.go (deleted in task 3)
  - cmd/run_v2.go (merged into run.go in task 3)

  **Build & Tests:** All passing

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
