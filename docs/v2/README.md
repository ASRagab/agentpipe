---
type: reference
title: AgentPipe v2 Documentation
created: 2026-01-18
tags:
  - v2
  - documentation
  - index
---

# AgentPipe v2 Documentation

Welcome to the AgentPipe v2 documentation. This directory contains comprehensive guides for using and extending AgentPipe v2.

## Getting Started

| Document | Description |
|----------|-------------|
| [[quickstart]] | 5-minute guide to get up and running |
| [[configuration]] | Complete configuration reference |
| [[migration]] | Migrate from v1 to v2 |

## User Guides

| Document | Description |
|----------|-------------|
| [[adapters]] | Guide to available AI adapters |
| [[tui]] | Terminal UI usage and shortcuts |
| [[troubleshooting]] | Common issues and solutions |

## Developer Documentation

| Document | Description |
|----------|-------------|
| [[architecture]] | Technical overview for contributors |

## Example Configurations

Ready-to-use configurations for common use cases (in `examples/v2/`):

| Example | Description |
|---------|-------------|
| `minimal.yaml` | Simplest working config with one agent |
| `two-agents.yaml` | Two agents for side-by-side comparison |
| `multi-model.yaml` | Compare multiple models from same provider |
| `code-review.yaml` | Code review with security, performance, maintainability focus |
| `brainstorm.yaml` | Creative brainstorming with diverse perspectives |
| `research.yaml` | Research and fact-finding with verification |

## Code Examples

Working Go code examples for extending and integrating AgentPipe v2 (in `examples/v2/code/`):

| Example | Description |
|---------|-------------|
| [custom-adapter/](../../examples/v2/code/custom-adapter/) | Implement a custom `AgentAdapter` for any AI provider |
| [programmatic/](../../examples/v2/code/programmatic/) | Use v2 as a library: conversations, streaming, persistence |
| [webhook/](../../examples/v2/code/webhook/) | Forward events to webhooks for analytics and integration |

Each example includes:
- Complete, runnable Go code
- Detailed README with explanations
- Best practices and patterns from the v2 codebase

## Quick Links

- **Installation**: See [[quickstart#installation]]
- **API Keys**: See [[quickstart#set-your-api-key]]
- **Keyboard Shortcuts**: See [[tui#keyboard-shortcuts]]
- **Breaking Changes**: See [[migration#breaking-changes]]

## What's New in v2

- **Parallel Execution** - All agents respond simultaneously (2-4x faster)
- **Real-Time Streaming** - See responses as they're generated
- **Event-Driven Architecture** - Cleaner, more extensible design
- **API-First Adapters** - Direct API integration for lower latency
- **Modern TUI** - 60fps updates, better layout, richer features

## Document Map

```
docs/v2/
├── README.md          ← You are here
├── quickstart.md      ← Start here
├── configuration.md   ← Full config reference
├── adapters.md        ← AI provider adapters
├── tui.md             ← Terminal UI guide
├── troubleshooting.md ← Common issues and fixes
├── architecture.md    ← Technical internals
└── migration.md       ← v1 → v2 migration

examples/v2/code/
├── custom-adapter/    ← Create custom AI adapters
├── programmatic/      ← Library usage patterns
└── webhook/           ← External integrations
```

## Need Help?

- **GitHub Issues**: [github.com/ASRagab/agentpipe/issues](https://github.com/ASRagab/agentpipe/issues)
- **Discussions**: [github.com/ASRagab/agentpipe/discussions](https://github.com/ASRagab/agentpipe/discussions)
