# Phase 12: Repository Migration and Security

This phase updates all GitHub references from the original repository to the forked repository and adds security hooks using the pre-commit framework with detect-secrets to prevent accidental credential leaks.

## Tasks

- [ ] Update all GitHub references from `kevinelliott/agentpipe` to `ASRagab/agentpipe`:
  - Update go.mod module path
  - Update all import statements across all Go files
  - Update README.md repository links
  - Update CHANGELOG.md references
  - Update any CI/CD workflow files (.github/workflows/)
  - Update Homebrew formula if present
  - Update any documentation with repository URLs
  - Run `go mod tidy` after changes
  - Verify build and tests pass after migration

- [ ] Set up pre-commit framework with detect-secrets:
  - Install pre-commit: `pip install pre-commit` or `brew install pre-commit`
  - Create `.pre-commit-config.yaml` with detect-secrets hook:
    ```yaml
    repos:
      - repo: https://github.com/Yelp/detect-secrets
        rev: v1.4.0
        hooks:
          - id: detect-secrets
            args: ['--baseline', '.secrets.baseline']
    ```
  - Generate initial baseline: `detect-secrets scan > .secrets.baseline`
  - Review and audit baseline: `detect-secrets audit .secrets.baseline`
  - Install hooks: `pre-commit install`
  - Add additional useful hooks (golangci-lint, gofmt, govet)
  - Test that secrets are detected and blocked
  - Document setup in CLAUDE.md

- [ ] Create `.secrets.baseline` file:
  - Scan existing codebase for false positives
  - Audit and mark false positives in baseline
  - Exclude test fixtures and example files with placeholder secrets
  - Document how to update baseline when adding new files

- [ ] Add additional pre-commit hooks for Go:
  - golangci-lint for linting
  - go-fmt for formatting
  - go-imports for import organization
  - go-mod-tidy for dependency verification
  - End-of-file fixer
  - Trailing whitespace removal

- [ ] Update Phase-11 release tasks now that we're on the fork:
  - Update release URL references from kevinelliott to ASRagab
  - Verify GitHub Actions workflows work on fork
  - Update Homebrew tap references if applicable
  - Re-verify release tag and GitHub release are accessible

- [ ] Merge worktree branch and clean up:
  - Check if multi-window-tui worktree has changes to preserve
  - Merge any needed changes back to feature/artifact-collection
  - Remove worktree: `git worktree remove .worktrees/multi-window-tui`
  - Prune worktree references: `git worktree prune`

- [ ] Clean up intermediate state files:
  - Remove .swarm/state.json modifications
  - Remove .claude/settings.local.json local overrides
  - Remove .claude/memory.db if not needed
  - Remove .swarm/memory.db if not needed
  - Remove coverage.html
  - Clean up any temporary Working folder files
  - Remove v2demo and v2tui binaries if present

- [ ] Update CLAUDE.md with new repository information:
  - Update any hardcoded repository paths
  - Document the pre-commit hook setup
  - Add instructions for running detect-secrets
  - Add security best practices section

- [ ] Verify all changes work correctly:
  - Run full test suite
  - Verify build succeeds
  - Test pre-commit hook blocks secrets
  - Confirm git operations work with new remote
