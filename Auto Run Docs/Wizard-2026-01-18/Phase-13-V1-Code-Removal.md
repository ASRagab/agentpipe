# Phase 13: V1 Code Removal

This phase performs a wholesale removal of all v1 code, documentation, and references, completing the migration to the v2 architecture.

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
  - Remove pkg/adapters/ (v1 adapters - claude, gemini, qwen, etc.)
  - Remove pkg/client/ (v1 HTTP client)
  - Remove pkg/config/ (v1 configuration)
  - Remove pkg/log/ (if not shared with v2) - **NOTE: used by cmd/root.go, evaluate first**
  - Remove pkg/orchestrator/ (v1 orchestrator)
  - Remove pkg/tui/ (v1 TUI - replaced by pkg/v2/tui/)
  - Remove pkg/agent/ (v1 agent interfaces)
  - Remove pkg/conversation/ (v1 conversation types)
  - Remove pkg/logger/ (duplicate logging package)
  - ~~Remove pkg/providers/~~ - **Does not exist; providers are in internal/providers/**
  - ~~Remove pkg/types/~~ - **Does not exist**
  - **Additional packages to evaluate:** pkg/artifact, pkg/export, pkg/metrics, pkg/middleware, pkg/ratelimit, pkg/utils

- [ ] Update cmd/ to use v2 only:
  - Update cmd/run.go to use pkg/v2/ packages
  - Update cmd/run_v2.go (rename to cmd/run.go if separate)
  - Remove any v1-specific commands
  - Update cmd/doctor.go for v2
  - Update cmd/root.go imports

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
