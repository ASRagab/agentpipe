# Phase 13: V1 Code Removal

This phase performs a wholesale removal of all v1 code, documentation, and references, completing the migration to the v2 architecture.

## Tasks

- [ ] Identify all v1 code locations:
  - List all pkg/ directories that are v1 (not under pkg/v2/)
  - List all cmd/ files using v1 packages
  - Identify v1-specific configuration files
  - Find v1 documentation in docs/
  - Locate v1 example configurations

- [ ] Remove v1 adapter packages:
  - Remove pkg/adapters/ (v1 adapters - claude, gemini, qwen, etc.)
  - Remove pkg/client/ (v1 HTTP client)
  - Remove pkg/config/ (v1 configuration)
  - Remove pkg/log/ (if not shared with v2)
  - Remove pkg/orchestrator/ (v1 orchestrator)
  - Remove pkg/providers/ (v1 providers)
  - Remove pkg/tui/ (v1 TUI - replaced by pkg/v2/tui/)
  - Remove pkg/types/ (v1 types)

- [ ] Update cmd/ to use v2 only:
  - Update cmd/run.go to use pkg/v2/ packages
  - Update cmd/run_v2.go (rename to cmd/run.go if separate)
  - Remove any v1-specific commands
  - Update cmd/doctor.go for v2
  - Update cmd/root.go imports

- [ ] Remove v1 internal packages:
  - Remove internal/bridge/ if v1-only
  - Update any shared internal/ code

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
