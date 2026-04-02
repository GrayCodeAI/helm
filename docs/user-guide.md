# HELM User Guide

## Overview

HELM is a unified TUI-first control plane for managing AI coding agents across all providers (Claude, Codex, Gemini, Ollama, etc.).

**You steer. Agents row.**

## Quick Start

### Installation

```bash
# From source
go install github.com/yourname/helm/cmd/helm@latest

# Or clone and build
git clone https://github.com/yourname/helm.git
cd helm
go build -o helm ./cmd/helm
```

### First Run

```bash
# Initialize HELM in your project
helm init

# Launch the TUI dashboard
helm
```

## Core Concepts

### Sessions

A session is a single agent interaction. Each session tracks:
- Provider and model used
- Token usage (input, output, cache)
- Cost
- Status (running, done, failed, paused)
- Summary of what was done

```bash
# Run a session
helm run "implement a feature"

# List sessions
helm session list

# View session details
helm session view <id>
```

### Cost Tracking

HELM automatically tracks costs for every session using per-model pricing.

```bash
# View cost summary
helm cost

# View detailed cost report
helm report cost --period month
```

### Project Memory

HELM learns from your sessions and stores conventions, patterns, and preferences.

```bash
# View memory entries
helm memory list

# Search memory
helm memory search "naming"
```

### Prompt Library

Organize and reuse prompts across sessions.

```bash
# List prompts
helm prompts list

# Run with a named prompt
helm run --prompt add-feature
```

## Configuration

HELM uses a TOML config file at `~/.helm/helm.toml`:

```toml
[providers.anthropic]
api_key = "sk-ant-..."
default_model = "claude-sonnet-4-20250514"

[providers.openai]
api_key = "sk-..."
default_model = "gpt-4o"

[router]
fallback_chain = ["anthropic", "openai", "openrouter"]
max_retries = 3

[budget]
daily_limit = 10.0
weekly_limit = 50.0
monthly_limit = 150.0
warning_pct = 0.80
```

### Environment Variables

All API keys can also be set via environment variables:
- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`
- `GOOGLE_API_KEY`
- `OPENROUTER_API_KEY`

## TUI Dashboard

The TUI provides:
- **Dashboard** - Overview of sessions, costs, and memory
- **Sessions** - Browse and manage all sessions
- **Cost** - Detailed cost breakdown and budget tracking
- **Memory** - Browse and search project memory
- **Prompts** - Manage prompt library

### Navigation

| Key | Action |
|-----|--------|
| `1` / `d` | Dashboard |
| `2` / `s` | Sessions |
| `3` / `c` | Cost |
| `4` / `m` | Memory |
| `5` / `p` | Prompts |
| `q` | Quit |
| `j`/`k` | Navigate |
| `Enter` | Select/View |

## Web Dashboard

HELM also provides a web dashboard:

```bash
# Start the web server
helm serve

# Open in browser
open http://localhost:8080
```

## Advanced Features

### Session Forking

Fork a session to try a different approach:

```bash
helm session fork <session-id> --prompt "try a different approach"
```

### Diff Review

Review changes made by agent sessions:

```bash
helm diff <session-id>
```

### Export/Import

```bash
# Export all data
helm export full backup.tar.gz

# Import data
helm import full backup.tar.gz
```

### Reports

```bash
# Generate cost report
helm report cost --period month --format json

# Generate session report
helm report sessions --limit 100

# Generate model performance report
helm report models
```

## Troubleshooting

### Common Issues

**Provider not configured:**
```bash
# Set API key via environment
export ANTHROPIC_API_KEY="sk-ant-..."

# Or add to config
helm config set providers.anthropic.api_key "sk-ant-..."
```

**Database issues:**
```bash
# Check database status
helm status health
```

**Cost tracking not working:**
```bash
# Verify cost tracking is enabled
helm config get cost.enabled
```

## Support

- GitHub Issues: https://github.com/yourname/helm/issues
- Documentation: https://github.com/yourname/helm/docs
