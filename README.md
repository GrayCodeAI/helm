# HELM — Personal Coding Agent Control Plane

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge">
  <img src="https://img.shields.io/badge/TUI-Bubbletea-purple?style=for-the-badge">
</p>

> **Helm** — You steer. Agents row.

A unified TUI-first control plane for personal coders to manage, monitor, and optimize AI coding agents across all providers.

---

## Features

| Category | Features |
|----------|----------|
| **Providers** | Anthropic (Claude), OpenAI (Codex), Google (Gemini), Ollama, OpenRouter |
| **Session** | Dashboard, Archive with FTS5 search, Fork, Replay |
| **Memory** | SQLite-backed project memory, Auto-learn conventions, Global memory |
| **Cost** | Per-session/project tracking, Budget alerts, Cost reports |
| **Diff** | Side-by-side viewer, Accept/reject per-file, Smart triage |
| **Intelligence** | Mistake journal, Auto-retry, Context pruning, AI summaries |
| **Automation** | Nightly scheduler, CI integration, Release automation |
| **Analytics** | ROI dashboard, Waste detection, Hotspot analysis, Trends |

---

## Quick Start

```bash
# Install via script
curl -fsSL https://raw.githubusercontent.com/GrayCodeAI/helm/main/install.sh | bash

# Or build from source
git clone https://github.com/GrayCodeAI/helm.git
cd helm
go build -o helm ./cmd/helm

# Initialize
./helm init

# Launch TUI
./helm
```

---

## Commands

| Command | Description |
|---------|-------------|
| `helm` | Launch TUI dashboard |
| `helm run` | Start new agent session |
| `helm init` | Initialize project |
| `helm memory` | Manage project memory |
| `helm cost` | View cost tracking |
| `helm prompts` | Manage prompts |
| `helm status` | Show session status |
| `helm diff` | Review changes |

---

## Architecture

```
┌─────────────────────────────────────┐
│           TUI (Bubbletea)           │
├─────────────────────────────────────┤
│              CLI                    │
├─────────────────────────────────────┤
│     Provider Router (7 adapters)    │
├─────────────────────────────────────┤
│  Session │ Memory │ Cost │ Diff     │
├─────────────────────────────────────┤
│        SQLite (modernc.org)         │
└─────────────────────────────────────┘
```

---

## Tech Stack

- **Go** — Single binary, no dependencies
- **Bubbletea v2** — Terminal UI framework
- **SQLite** — Pure Go (no CGO)
- **sqlc** — Type-safe SQL

---

## Configuration

Create `helm.toml`:

```toml
[provider]
default = "anthropic"

[provider.anthropic]
api_key = "${ANTHROPIC_API_KEY}"

[provider.openai]
api_key = "${OPENAI_API_KEY}"

[cost]
budget = 100.00
alert_threshold = 0.8

[memory]
auto_learn = true
```

---

## License

MIT — [GrayCodeAI](https://github.com/GrayCodeAI)
