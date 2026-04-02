# HELM — Personal Coding Agent Control Plane

> You steer. Agents row.

A unified TUI-first control plane for personal coders to manage, monitor, and optimize AI coding agents across all providers.

## Features

- **Provider Router** — Claude, OpenAI, Gemini, Ollama, OpenRouter with auto-fallback
- **Prompt Library** — YAML templates, fuzzy search, one-keystroke launch
- **Project Memory** — SQLite-backed, survives sessions, auto-learns conventions
- **Session Dashboard** — All sessions, status, cost at a glance
- **Diff Review** — Side-by-side viewer, accept/reject per-file or per-hunk
- **Cost Tracker** — Per-session, per-project spend with budget alerts
- **Session Archive** — Full-text search past sessions across all providers
- **Auto-Retry** — Agent fails → retry with corrected context from mistake journal

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/helm/main/install.sh | bash

# Or build from source
git clone https://github.com/GrayCodeAI/helm.git
cd helm
go build ./cmd/helm

# Initialize a project
helm init

# Launch the TUI
helm
```

## Commands

```
helm              Launch the TUI dashboard
helm run          Start a new agent session
helm init         Initialize HELM in a project
helm memory       Manage project memory
helm cost         View cost tracking and budgets
helm prompts      Browse and manage prompt library
helm status       Show session status
helm diff         Review pending changes
```

## Tech Stack

- **Go** — Single binary, no dependencies
- **Bubbletea** — Terminal UI framework
- **SQLite** — Local-first storage (pure Go, no CGO)
- **sqlc** — Type-safe SQL queries

## License

MIT
