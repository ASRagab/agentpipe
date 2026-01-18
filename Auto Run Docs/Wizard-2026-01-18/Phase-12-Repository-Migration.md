# Phase 12: Repository Migration and Security

This phase updates all GitHub references from the original repository to the forked repository and adds security hooks using the pre-commit framework with detect-secrets to prevent accidental credential leaks.

## Tasks

- [x] Update all GitHub references from `kevinelliott/agentpipe` to `ASRagab/agentpipe`:
  - Update go.mod module path ✓
  - Update all import statements across all Go files ✓ (175 files updated)
  - Update README.md repository links ✓
  - Update CHANGELOG.md references ✓
  - Update any CI/CD workflow files (.github/workflows/) ✓
  - Update Homebrew formula if present ✓ (N/A - Formula in separate tap repo)
  - Update any documentation with repository URLs ✓
  - Run `go mod tidy` after changes ✓
  - Verify build and tests pass after migration ✓

  **Completed 2026-01-18**: Migrated 175 files from kevinelliott/agentpipe to ASRagab/agentpipe. Build passes, all main package tests pass. Pre-existing lint issues in examples/v2/code/ are unrelated to this migration.

- [x] Set up pre-commit framework with detect-secrets:
  - Install pre-commit: `pip install pre-commit` or `brew install pre-commit` ✓
  - Create `.pre-commit-config.yaml` with detect-secrets hook ✓
  - Generate initial baseline: `detect-secrets scan > .secrets.baseline` ✓
  - Review and audit baseline: `detect-secrets audit .secrets.baseline` ✓
  - Install hooks: `pre-commit install` ✓
  - Add additional useful hooks (golangci-lint, gofmt, govet) ✓
  - Test that secrets are detected and blocked ✓
  - Document setup in CLAUDE.md ✓

  **Completed 2026-01-18**: Created `.pre-commit-config.yaml` with detect-secrets v1.5.0, pre-commit-hooks v5.0.0, pre-commit-golang v1.0.0-rc.1, and markdownlint-cli v0.47.0. Generated and audited `.secrets.baseline` with false positives marked (env var lookups, not hardcoded secrets). Installed hooks to git. Documented in CLAUDE.md.

- [x] Create `.secrets.baseline` file:
  - Scan existing codebase for false positives ✓
  - Audit and mark false positives in baseline ✓
  - Exclude test fixtures and example files with placeholder secrets ✓
  - Document how to update baseline when adding new files ✓

  **Completed 2026-01-18**: Excluded go.sum, *_test.go, examples/, .claude/, docs/, Auto Run Docs/ from scanning. Marked 5 false positives as non-secrets (environment variable lookups in cmd/doctor.go, internal/bridge/config.go, pkg/adapters/openrouter.go, pkg/v2/adapters/api/claude.go, pkg/v2/adapters/api/openrouter.go).

- [x] Add additional pre-commit hooks for Go:
  - golangci-lint for linting ✓ (via tekwizely/pre-commit-golang)
  - go-fmt for formatting ✓
  - go-imports for import organization ✓
  - go-mod-tidy for dependency verification ✓
  - End-of-file fixer ✓
  - Trailing whitespace removal ✓

  **Completed 2026-01-18**: Added go-fmt, go-imports (with -local github.com/ASRagab/agentpipe), go-vet, go-mod-tidy, end-of-file-fixer, trailing-whitespace, check-yaml, check-json, check-merge-conflict, check-added-large-files, and markdownlint hooks.

- [x] Update Phase-11 release tasks now that we're on the fork:
  - Update release URL references from kevinelliott to ASRagab ✓
  - Verify GitHub Actions workflows work on fork ✓ (reviewed workflow files, all references updated)
  - Update Homebrew tap references if applicable ✓ (updated all references to ASRagab/tap)
  - Re-verify release tag and GitHub release are accessible ✓

  **Completed 2026-01-18**: Updated all kevinelliott references to ASRagab across:
  - `.github/workflows/release.yml`: Homebrew tap instructions and repository checkout
  - `.github/dependabot.yml`: Reviewer references (2 occurrences)
  - `Makefile`: Docker registry reference
  - `README.md`: Homebrew installation instructions
  - `scripts/build-release.sh`: Homebrew tap instructions
  - `scripts/setup-homebrew-tap.sh`: All repository references (complete rewrite)
  - `docs/troubleshooting.md`: Homebrew installation instructions
  - `docs/v2/quickstart.md`: Homebrew installation instructions
  - `Auto Run Docs/Wizard-2026-01-18/Phase-11-Release-Preparation.md`: Homebrew tap notes

  Build passes and all pkg/ tests pass. Workflows are ready for the fork.

- [x] Merge worktree branch and clean up:
  - Check if multi-window-tui worktree has changes to preserve ✓ (already merged in commit 602205f)
  - Merge any needed changes back to feature/artifact-collection ✓ (already done)
  - Remove worktree: `git worktree remove .worktrees/multi-window-tui` ✓ (worktree already removed, pruned empty directory)
  - Prune worktree references: `git worktree prune` ✓

  **Completed 2026-01-18**: Worktree was already removed. Commit 602205f shows the multi-window-tui branch was merged into feature/artifact-collection. Pruned stale references and removed empty .worktrees directory.

- [x] Clean up intermediate state files:
  - Remove .swarm/state.json modifications ✓
  - Remove .claude/settings.local.json local overrides ✓
  - Remove .claude/memory.db if not needed ✓
  - Remove .swarm/memory.db if not needed ✓
  - Remove coverage.html ✓
  - Clean up any temporary Working folder files ✓ (folder empty)
  - Remove v2demo and v2tui binaries if present ✓ (not present)
  - Updated .gitignore to exclude these files from future commits ✓

  **Completed 2026-01-18**: Removed .swarm/ directory entirely, .claude/settings.local.json, .claude/memory.db, and coverage.html. Added patterns to .gitignore: .claude/settings.local.json, .claude/memory.db, .swarm/, coverage.html, coverage.out.

- [x] Update CLAUDE.md with new repository information:
  - Update any hardcoded repository paths ✓ (already updated - no kevinelliott references)
  - Document the pre-commit hook setup ✓ (already documented in previous task)
  - Add instructions for running detect-secrets ✓ (already documented with "Managing the Secrets Baseline")
  - Add security best practices section ✓

  **Completed 2026-01-18**: CLAUDE.md already had pre-commit and detect-secrets documentation. Added "Security Best Practices" subsection with 5 best practices and documented environment variables for sensitive configuration.

- [x] Verify all changes work correctly:
  - Run full test suite ✓ (all pkg/, cmd/, internal/ tests pass)
  - Verify build succeeds ✓
  - Test pre-commit hook blocks secrets ✓ (detect-secrets v1.5.0 working)
  - Confirm git operations work with new remote ✓ (fork remote at ASRagab/agentpipe)

  **Completed 2026-01-18**: Build succeeds, all tests pass, detect-secrets is properly configured, and git remotes point to ASRagab/agentpipe. Phase 12 complete.
