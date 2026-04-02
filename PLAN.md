# HELM — Personal Coding Agent Control Plane

> **Helm** — You steer. Agents row.

A unified TUI-first control plane for personal coders to manage, monitor, and optimize AI coding agents across all providers.

---

## Table of Contents

1. [Vision](#vision)
2. [Architecture](#architecture)
3. [Tech Stack](#tech-stack)
4. [Feature Roadmap](#feature-roadmap)
5. [Phase 1: Foundation](#phase-1-foundation)
6. [Phase 2: Intelligence](#phase-2-intelligence)
7. [Phase 3: Autonomy](#phase-3-autonomy)
8. [Phase 4: Analytics](#phase-4-analytics)
9. [Phase 5: Experience](#phase-5-experience)
10. [Phase 6: Ecosystem](#phase-6-ecosystem)
11. [Data Model](#data-model)
12. [Project Structure](#project-structure)
13. [Provider Integration](#provider-integration)
14. [Session Format Parsing](#session-format-parsing)
15. [Build & Release](#build--release)
16. [Testing Strategy](#testing-strategy)
17. [Security Model](#security-model)
18. [Performance Targets](#performance-targets)
19. [Migration Path](#migration-path)
20. [Competitive Analysis](#competitive-analysis)

---

## Vision

**Problem:** Personal developers using AI coding agents face:
- Context loss between sessions (agent forgets conventions)
- Diff overload (agent touches 30 files, you needed 5)
- Cost blindness (no visibility into token burn)
- Session chaos (tmux hell, lost context on terminal close)
- No memory (agents repeat same mistakes session after session)
- Fragmented tooling (separate tools for sessions, memory, cost, prompts)

**Solution:** HELM — a single unified TUI that combines:
- Session management across all providers
- Persistent project memory (SQLite)
- Prompt library (YAML templates)
- Cost tracking with budget alerts
- Smart diff review with triage
- Auto-retry with learning from mistakes

**Design Principles:**
1. **Local-first** — No cloud dependency, all data in SQLite
2. **Agent-agnostic** — Works with Claude Code, Codex, Gemini, OpenCode, Crush, etc.
3. **TUI-first** — Terminal-native, keyboard-driven, fast
4. **CLI-also** — Scriptable commands for automation/CI
5. **Progressive disclosure** — Simple defaults, deep configuration when needed
6. **Zero-config startup** — `helm init` auto-detects everything

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         HELM TUI                                │
│                    (Bubbletea v2)                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │Dashboard │ │ Sessions │ │  Diffs   │ │  Cost    │          │
│  │  Screen  │ │  Screen  │ │  Screen  │ │  Screen  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
├─────────────────────────────────────────────────────────────────┤
│                        CLI Commands                             │
│  helm run  helm memory  helm cost  helm prompts  helm status   │
├─────────────────────────────────────────────────────────────────┤
│                     Core Services                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐              │
│  │Provider │ │ Session │ │ Memory  │ │  Cost   │              │
│  │ Router  │ │ Manager │ │ Engine  │ │ Tracker │              │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘              │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐              │
│  │ Prompt  │ │  Diff   │ │  Auto   │ │ Quality │              │
│  │ Library │ │ Engine  │ │  Retry  │ │  Gates  │              │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘              │
├─────────────────────────────────────────────────────────────────┤
│                     Provider Adapters                           │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐      │
│  │Anthropic│ │OpenAI  │ │Google  │ │Ollama  │ │Open    │      │
│  │(Claude)│ │(Codex) │ │(Gemini)│ │(Local) │ │Router  │      │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘      │
├─────────────────────────────────────────────────────────────────┤
│                     Session Parsers                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │Claude    │ │Codex     │ │Gemini    │ │OpenCode  │          │
│  │JSONL     │ │JSONL     │ │JSONL     │ │JSONL     │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
├─────────────────────────────────────────────────────────────────┤
│                     Storage Layer                               │
│  ┌─────────────────────────────────────────────────────┐       │
│  │              SQLite (modernc.org/sqlite)             │       │
│  │  sessions | memories | prompts | costs | mistakes   │       │
│  └─────────────────────────────────────────────────────┘       │
│  ┌─────────────────────────────────────────────────────┐       │
│  │              File System                             │       │
│  │  ~/.helm/  (config, global memory, prompt library)  │       │
│  │  .helm/    (project memory, local prompts)          │       │
│  └─────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

---

## Tech Stack

| Component | Choice | Why |
|-----------|--------|-----|
| **Language** | Go 1.25+ | Single binary, fast, proven by agentsview/engram/crush |
| **TUI Framework** | Bubbletea v2 (charm.land/bubbletea/v2) | Best Go TUI, used by crush/agent-deck/engram |
| **TUI Components** | Bubbles v2 (charm.land/bubbles/v2) | Pre-built components (list, table, textinput) |
| **Styling** | Lipgloss v2 (charm.land/lipgloss/v2) | Terminal styling, theming |
| **Markdown** | Glamour v2 (charm.land/glamour/v2) | Render markdown in terminal |
| **Database** | modernc.org/sqlite (pure Go) | No CGO, single binary, FTS5 support |
| **Schema Gen** | sqlc | Type-safe SQL → Go code generation |
| **CLI Framework** | Cobra + Viper | Standard Go CLI, used by crush |
| **Fuzzy Search** | sahilm/fuzzy | Fast fuzzy matching for prompts/sessions |
| **Diff Engine** | aymanbagabas/go-udiff | Pure Go unified diff |
| **Syntax Highlight** | alecthomas/chroma/v2 | Terminal code highlighting |
| **Terminal PTY** | creack/pty | Terminal session management |
| **File Watching** | fsnotify/fsnotify | Real-time session file monitoring |
| **Config** | TOML (BurntSushi/toml) | Human-readable, typed config |
| **Prompt Templates** | text/template (stdlib) | Go templates for prompts |
| **UUID** | google/uuid | Session/message IDs |
| **Humanize** | dustin/go-humanize | Readable sizes, costs, times |
| **Testing** | testify | Assertions, require, suite |
| **Linting** | golangci-lint | Comprehensive Go linting |
| **Formatting** | gofumpt | Stricter gofmt |
| **Release** | GoReleaser | Cross-platform binaries |
| **Install** | Shell script + Homebrew | Easy installation |

---

## Feature Roadmap

**Total: 48 features across 6 phases**

| Phase | Duration | Features | Focus |
|-------|----------|----------|-------|
| 1. Foundation | Weeks 1-4 | 8 | Core loop works |
| 2. Intelligence | Weeks 5-8 | 8 | Agent learns |
| 3. Autonomy | Weeks 9-12 | 8 | Works while away |
| 4. Analytics | Weeks 13-16 | 8 | Data-driven |
| 5. Experience | Weeks 17-20 | 8 | Frictionless |
| 6. Ecosystem | Weeks 21-24 | 8 | Connects to everything |

---

## Phase 1: Foundation (Weeks 1-4)

> **Goal:** You can launch an agent, see what it did, review changes, track cost, and restart with context.

### F1.1 — Provider Router

**Description:** Unified interface to all LLM providers with per-session switching and auto-fallback.

**Supported Providers:**
- Anthropic (Claude Sonnet, Opus, Haiku)
- OpenAI (GPT-4o, GPT-4o-mini, o1, o3)
- Google (Gemini 2.5 Pro, Gemini 2.5 Flash)
- OpenRouter (any model via OpenRouter API)
- Ollama (any local model)
- Custom (any OpenAI-compatible endpoint)

**Implementation:**
```
internal/provider/
  router.go          # ProviderRouter — selects provider per session
  interface.go       # Provider interface (Chat, Stream, Models, Cost)
  anthropic.go       # Anthropic API adapter
  openai.go          # OpenAI API adapter
  google.go          # Google AI adapter
  ollama.go          # Ollama local adapter
  openrouter.go      # OpenRouter adapter
  custom.go          # OpenAI-compatible custom endpoint
  models.go          # Model catalog with pricing, context windows
  fallback.go        # Auto-fallback chain on errors/rate limits
  pricing.go         # Per-model pricing (input/output/cache tokens)
```

**Config (helm.toml):**
```toml
[providers.anthropic]
api_key = "sk-ant-..."
default_model = "claude-sonnet-4-20250514"

[providers.openai]
api_key = "sk-..."
default_model = "gpt-4o"

[providers.ollama]
base_url = "http://localhost:11434"
default_model = "qwen2.5-coder:32b"

[router]
fallback_chain = ["anthropic", "openai", "openrouter"]
rate_limit_retry = true
max_retries = 3
```

**Key Design Decisions:**
- Use a unified `Provider` interface — all adapters implement the same methods
- Model catalog includes pricing, context window, max output tokens
- Fallback chain is configurable — if Anthropic rate-limits, try OpenAI
- Pricing data refreshed from a static file (updated monthly)

### F1.2 — Prompt Library

**Description:** YAML-based prompt templates, searchable in TUI, one-keystroke launch.

**Implementation:**
```
internal/prompt/
  library.go         # PromptLibrary — load, search, render templates
  template.go        # PromptTemplate struct, variable injection
  discover.go        # Auto-discover prompts from ~/.helm/prompts/ and .helm/prompts/
  builtin.go         # Built-in prompts (add-feature, fix-bug, refactor, write-tests, etc.)
```

**Prompt Format (YAML):**
```yaml
name: add-feature
description: Add a new feature to the codebase
tags: [feature, implementation]
complexity: medium
context:
  - project_memory    # Auto-inject project memory
  - file_structure    # Auto-inject current file structure
template: |
  Add the following feature: {{.Task}}
  
  Project context:
  {{.ProjectMemory}}
  
  Guidelines:
  - Follow existing conventions
  - Write tests for new code
  - Use existing patterns
variables:
  - name: Task
    description: Describe the feature to add
    required: true
```

**Built-in Prompts:**
- `add-feature` — Implement a new feature
- `fix-bug` — Fix a specific bug
- `refactor` — Refactor code with specific goals
- `write-tests` — Generate tests for existing code
- `review` — Review code for issues
- `explain` — Explain complex code
- `docs` — Generate/update documentation
- `deps` — Update dependencies
- `security` — Security audit and fixes
- `performance` — Performance optimization

**TUI Integration:**
- Fuzzy-searchable list with tags, complexity, description
- Variable input form before launch
- One-keystroke launch (number keys or vim-style navigation)

### F1.3 — Project Memory

**Description:** SQLite-backed persistent memory that survives sessions, auto-learns conventions.

**Implementation:**
```
internal/memory/
  engine.go          # MemoryEngine — store, retrieve, consolidate
  store.go           # SQLite operations for memory entries
  auto.go            # Auto-learn from sessions (conventions, patterns)
  recall.go          # Context-aware recall for session start
  forget.go          # Forgetting curve — old/unused info fades
  types.go           # MemoryEntry, MemoryType enums
```

**Memory Types:**
- `convention` — Coding conventions (naming, patterns, structure)
- `decision` — Architecture decisions, past choices
- `preference` — User preferences (framework, style, tools)
- `fact` — Project facts (tech stack, deployment, structure)
- `correction` — Past mistakes and their fixes
- `skill` — Reusable skills extracted from sessions

**Storage Schema:**
```sql
CREATE TABLE memories (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL,
    type TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    source TEXT,              -- 'auto' | 'manual' | 'extracted'
    confidence REAL DEFAULT 0.5,
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_memories_project ON memories(project);
CREATE INDEX idx_memories_type ON memories(type);
CREATE VIRTUAL TABLE memories_fts USING fts5(key, value, content='memories');
```

**Auto-Learn Sources:**
- Rejected diffs → correction entries
- Manual edits after agent output → pattern entries
- Successful session outcomes → skill entries
- Repeated patterns across sessions → convention entries

**CLI:**
```bash
helm memory list              # List all memories
helm memory set "use tailwind"  # Manual entry
helm memory get               # Show current project memory
helm memory forget "old rule"  # Remove a memory
```

### F1.4 — Session Dashboard

**Description:** Central TUI screen showing all sessions, status, and key metrics.

**Implementation:**
```
internal/ui/
  dashboard.go         # Main dashboard screen
  session_list.go      # Session list component with status badges
  status_bar.go        # Global status bar (provider, cost, time)
  keybindings.go       # Global keybinding handler
```

**Dashboard Layout:**
```
┌────────────────────────────────────────────────────────────┐
│ HELM v0.1.0                              Today: $2.47 spent│
├────────────────────────────────────────────────────────────┤
│ Sessions (7)                            [q]uit [n]ew [s]earch│
│ ┌──────────────────────────────────────────────────────┐   │
│ │ ● add-auth-feature    Claude Sonnet   Running   3m    │   │
│ │ ✓ fix-login-bug       GPT-4o        Done      12m    │   │
│ │ ✗ refactor-api        Gemini        Failed    8m     │   │
│ │ ⏸ write-tests         Claude Haiku  Paused    5m     │   │
│ │ ● update-deps         Ollama        Running   1m     │   │
│ └──────────────────────────────────────────────────────┘   │
│ ─────────────────────────────────────────────────────────  │
│ Quick Launch                            [p]rompts [m]emory │
│ [1] add-feature  [2] fix-bug  [3] refactor  [4] tests     │
└────────────────────────────────────────────────────────────┘
```

**Session States:**
- `running` — Agent actively working
- `done` — Completed successfully
- `failed` — Ended with error
- `paused` — User paused
- `stuck` — Detected looping/repeating (Phase 2)

**Session Data Source:**
- Parse JSONL files from provider session directories
- Real-time updates via fsnotify file watching
- Canonical session model stored in SQLite

### F1.5 — Diff Review

**Description:** Side-by-side diff viewer in TUI, accept/reject per-file or per-hunk.

**Implementation:**
```
internal/diff/
  viewer.go          # DiffViewer — render and navigate diffs
  engine.go          # DiffEngine — generate unified diffs
  classify.go        # Auto-classify changes (essential vs incidental)
  apply.go           # Apply/reject changes to working tree
  group.go           # Group changes by intent/file
```

**Diff Review Layout:**
```
┌────────────────────────────────────────────────────────────┐
│ Diff Review — fix-login-bug (3 files, 47 changes)          │
├────────────────────────────────────────────────────────────┤
│ [✓] auth/login.go    (+12/-3)  Essential                   │
│ [ ] auth/session.go  (+8/-2)   Essential                   │
│ [ ] go.mod           (+1/-0)   Incidental                  │
│                                                            │
│ ── auth/login.go ───────────────────────────────────────── │
│   42  func Login(w, r) {                                   │
│ - 43      token := generateToken(user.ID)                  │
│ + 43      token, err := generateSecureToken(user.ID)       │
│   44      if err != nil {                                  │
│ + 45          http.Error(w, "token error", 500)            │
│   46          return                                       │
│                                                            │
│ [a]ccept  [r]eject  [h]unk  [n]ext  [p]rev  [q]uit        │
└────────────────────────────────────────────────────────────┘
```

**Features:**
- Side-by-side or unified diff view
- Syntax-highlighted code (chroma)
- Accept/reject per-file or per-hunk
- Auto-classification: essential vs incidental
- Group changes by intent (not just file)
- Quick-generate focused PR from accepted changes

### F1.6 — Cost Tracker

**Description:** Per-session, per-project cost tracking with daily totals.

**Implementation:**
```
internal/cost/
  tracker.go         # CostTracker — record and aggregate costs
  calculator.go      # Token → cost calculation with per-model pricing
  budget.go          # Budget enforcement (warning, hard stop)
  parser.go          # Parse cost data from session JSONL files
  report.go          # Cost reports and summaries
```

**Cost Data Sources:**
- Parse session JSONL files (token counts per turn)
- Provider API responses (usage metadata)
- Real-time file watching for in-progress sessions

**Storage Schema:**
```sql
CREATE TABLE cost_records (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    project TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cache_read_tokens INTEGER,
    cache_write_tokens INTEGER,
    total_cost REAL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE budgets (
    project TEXT PRIMARY KEY,
    daily_limit REAL,
    weekly_limit REAL,
    monthly_limit REAL,
    warning_pct REAL DEFAULT 0.8
);
```

**TUI Display:**
```
┌────────────────────────────────────────┐
│ Cost Today: $2.47 / $10.00 (24.7%)    │
│ ─────────────────────────────────────  │
│ Session              Tokens    Cost    │
│ add-auth-feature     12,450   $0.87   │
│ fix-login-bug        34,200   $1.23   │
│ refactor-api         8,100    $0.37   │
│ ─────────────────────────────────────  │
│ Total                54,750   $2.47   │
└────────────────────────────────────────┘
```

### F1.7 — Session Archive

**Description:** Full-text search past sessions, export, tag by project/task.

**Implementation:**
```
internal/session/
  archive.go         # SessionArchive — store and retrieve sessions
  search.go          # Full-text search across session content
  export.go          # Export sessions to HTML, JSON, markdown
  canonical.go       # Canonical session model (provider-agnostic)
  parser_claude.go   # Parse Claude Code JSONL sessions
  parser_codex.go    # Parse Codex JSONL sessions
  parser_gemini.go   # Parse Gemini JSONL sessions
  parser_opencode.go # Parse OpenCode JSONL sessions
```

**Canonical Session Model:**
```go
type Session struct {
    ID           string
    Provider     string    // "anthropic", "openai", "google", etc.
    Model        string    // "claude-sonnet-4-20250514"
    Project      string    // Project path or name
    Prompt       string    // User's initial prompt
    Status       string    // "running", "done", "failed", "paused"
    StartedAt    time.Time
    EndedAt      time.Time
    InputTokens  int
    OutputTokens int
    Cost         float64
    FilesChanged []string
    Tags         []string
    Summary      string    // AI-generated summary (Phase 2)
}

type Message struct {
    SessionID string
    Role      string    // "user", "assistant", "system"
    Content   string
    ToolCalls []ToolCall
    Timestamp time.Time
}
```

**Session Parser Strategy:**
- Each provider stores sessions differently (JSONL format varies)
- Build a parser per provider that normalizes to canonical model
- Store canonical data in SQLite for fast search
- Keep raw JSONL references for replay

### F1.8 — One-Command Setup

**Description:** `helm init` auto-detects repo, builds memory, configures providers.

**Implementation:**
```
internal/setup/
  init.go            # HelmInit — full project initialization
  detect.go          # Detect tech stack, framework, language
  memory_build.go    # Auto-build initial project memory from codebase
  provider_config.go # Guide through provider configuration
  prompt_suggest.go  # Suggest relevant prompts based on stack
```

**What `helm init` Does:**
1. Detect language, framework, package manager
2. Scan project structure (key directories, config files)
3. Read existing docs (README, AGENTS.md, CLAUDE.md, CONTRIBUTING.md)
4. Build initial project memory (conventions, structure, tech stack)
5. Check for provider API keys in environment
6. Generate suggested prompts based on project type
7. Create `.helm/` directory with config

**Output:**
```
$ helm init

🔍 Analyzing project...
  Language: Go 1.25
  Framework: Bubbletea TUI
  Package Manager: go modules
  Test Framework: testify

📋 Building project memory...
  Found 12 conventions from existing code
  Found 3 architecture decisions from docs
  Found 5 tool preferences from config files

⚙️ Configuring providers...
  ✓ Anthropic (from ANTHROPIC_API_KEY)
  ✓ OpenAI (from OPENAI_API_KEY)
  ○ Google (not configured)
  ○ Ollama (not running)

💡 Suggested prompts:
  - add-feature (Go + Bubbletea patterns)
  - write-tests (testify patterns detected)
  - refactor (go modules structure)

✅ HELM ready! Run `helm` to start.
```

---

## Phase 2: Intelligence (Weeks 5-8)

> **Goal:** Less babysitting. Agent learns from mistakes. You review signal, not noise.

### F2.1 — Mistake Journal

**Description:** Auto-capture every agent error, rejected diff, failed test — build a personal failure database.

**Implementation:**
```
internal/mistake/
  journal.go         # MistakeJournal — record and query mistakes
  capture.go         # Auto-capture from session events
  patterns.go        # Detect recurring mistake patterns
  rules.go           # Generate correction rules from mistakes
```

**Mistake Types:**
- `rejected_diff` — User rejected agent's changes
- `test_failure` — Agent's code failed tests
- `lint_error` — Agent introduced lint violations
- `compile_error` — Agent's code doesn't compile
- `timeout` — Agent exceeded token/time budget
- `loop_detected` — Agent repeated same action
- `wrong_file` — Agent modified wrong files

**Storage:**
```sql
CREATE TABLE mistakes (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT NOT NULL,
    context TEXT,              -- Relevant code/context
    correction TEXT,           -- How it was fixed
    file_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### F2.2 — Auto-Retry + Learning

**Description:** Agent fails → retry with corrected context from mistake journal.

**Implementation:**
```
internal/retry/
  engine.go          # RetryEngine — decide when and how to retry
  context.go         # Build corrected context from mistake journal
  strategy.go        # Retry strategies (same model, different model, adjusted prompt)
```

**Retry Logic:**
1. Detect failure (error, rejected diff, timeout)
2. Check mistake journal for similar past failures
3. Build corrected context: "Last time X failed, fix was Y"
4. Retry with same model (up to 2 times)
5. If still failing, try fallback model
6. If all retries fail, pause and notify user

### F2.3 — Session Fork

**Description:** Branch a session, try different approach, compare later.

**Implementation:**
```
internal/session/
  fork.go            # SessionFork — create session branches
  compare.go         # Compare two session outcomes
```

**Fork Workflow:**
1. Select a session in dashboard
2. Press `f` to fork
3. Inherits: project memory, prompt context, file state
4. New prompt or modified prompt
5. Both sessions tracked side-by-side
6. Compare results in diff view

### F2.4 — Budget Alerts

**Description:** 80% warning, 100% hard stop, daily/weekly/monthly limits.

**Implementation:**
```
internal/cost/
  budget.go          # Budget enforcement engine
  alert.go           # Alert system (terminal notification, sound)
  enforcement.go     # Hard stop — prevent new sessions when over budget
```

**Config:**
```toml
[budget]
daily_limit = 10.00
weekly_limit = 50.00
monthly_limit = 150.00
warning_pct = 0.80
action_on_limit = "pause"  # "pause" | "stop" | "notify"
```

### F2.5 — Smart Context Pruning

**Description:** Auto-remove irrelevant files from context window, stay within token budget.

**Implementation:**
```
internal/context/
  pruner.go          # ContextPruner — select relevant files
  relevance.go       # Score file relevance to current task
  budget.go          # Token budget management
```

**Strategy:**
- Score files by relevance to prompt (keyword matching, file type, recent changes)
- Prioritize: project memory > recent changes > relevant files > rest
- Hard cap: never exceed model's context window
- Soft cap: stay within 80% for safety margin

### F2.6 — AI Session Summary

**Description:** One-line summary of what agent did — not raw logs.

**Implementation:**
```
internal/summary/
  generator.go       # Generate session summaries
  template.go        # Summary templates
```

**Summary Format:**
```
✓ fix-login-bug: Updated token generation to use secure method, added error handling in login.go and session.go. 2 files changed, +20/-5 lines. Cost: $1.23
```

**Generation:**
- On session completion, send session transcript to LLM with summary prompt
- Cache summary in SQLite
- Display in dashboard instead of raw prompt

### F2.7 — Smart Diff Triage

**Description:** Auto-classify: essential vs incidental changes, group by intent.

**Implementation:**
```
internal/diff/
  triage.go          # SmartDiffTriage — classify changes
  intent.go          # Group changes by intent
  score.go           # Score change importance
```

**Classification:**
- `essential` — Directly related to the task/prompt
- `incidental` — Side effects (formatting, imports, deps)
- `suspicious` — Unrelated changes (potential hallucination)

**Heuristics:**
- Changes to files mentioned in prompt → essential
- Import/formatting changes → incidental
- Changes to unrelated files → suspicious
- Large changes in small-task sessions → suspicious

### F2.8 — Pattern Learning

**Description:** Detect your conventions — naming, error handling, test style, structure preferences.

**Implementation:**
```
internal/pattern/
  detector.go        # PatternDetector — analyze codebase for patterns
  conventions.go     # Extract coding conventions
  style.go           # Learn user's coding style
```

**What It Learns:**
- Naming conventions (camelCase, snake_case, PascalCase)
- Error handling style (return errors, panic, log)
- Test patterns (table-driven, BDD, simple)
- File organization (by feature, by type, by layer)
- Import ordering and grouping
- Comment style and frequency

---

## Phase 3: Autonomy (Weeks 9-12)

> **Goal:** Set tasks and walk away. Come back to results, not process.

### F3.1 — Nightly Maintenance

**Description:** Scheduled runs: dep updates, lint fixes, type sync, doc regeneration.

**Implementation:**
```
internal/automation/
  scheduler.go       # TaskScheduler — cron-like scheduling
  nightly.go         # NightlyMaintenance — scheduled tasks
  tasks.go           # Built-in maintenance tasks
```

**Scheduled Tasks:**
- Update dependencies (go get -u, npm update, etc.)
- Fix lint violations
- Regenerate types/prototypes
- Update documentation from code
- Run tests and report failures

**Config:**
```toml
[automation.nightly]
enabled = true
time = "02:00"
timezone = "America/New_York"
tasks = ["deps", "lint", "docs"]
max_cost_per_run = 2.00
```

### F3.2 — Issue-to-PR Pipeline

**Description:** Pick up GitHub/GitLab issue → implement → open PR → you just review.

**Implementation:**
```
internal/automation/
  issue.go           # IssueFetcher — get issues from GitHub/GitLab
  pipeline.go        # IssueToPR — full pipeline
  pr.go              # PRCreator — open pull requests
```

**Pipeline:**
1. Fetch open issues from repo
2. Select issue (by label, priority, or manual selection)
3. Create branch from main
4. Run agent with issue description as prompt
5. Run tests, lint, build
6. If all pass, commit and push
7. Open PR with description
8. Notify user for review

### F3.3 — Self-Healing CI

**Description:** CI fails → agent investigates, fixes, pushes → loops until green.

**Implementation:**
```
internal/automation/
  ci.go              # CIWatcher — monitor CI status
  healer.go          # SelfHealer — investigate and fix CI failures
```

**Workflow:**
1. Webhook from GitHub Actions/GitLab CI on failure
2. Fetch CI logs and error output
3. Agent analyzes failure
4. Agent fixes issue
5. Push fix, re-run CI
6. Loop until green (max 3 iterations)
7. If still failing, notify user with analysis

### F3.4 — Red Team Mode

**Description:** One agent writes code, another tries to break it — surface edge cases.

**Implementation:**
```
internal/agent/
  redteam.go         # RedTeam — adversarial testing
  breaker.go         # CodeBreaker — find edge cases
```

**Workflow:**
1. Agent A writes code for task
2. Agent B reviews and tries to break it (edge cases, security, performance)
3. Agent B reports issues
4. Agent A fixes issues
5. Present final result to user with red team notes

### F3.5 — A/B Comparison

**Description:** Two agents on same task, side-by-side output, pick winner.

**Implementation:**
```
internal/agent/
  abtest.go          # ABComparison — run parallel agents
  compare.go         # Compare outputs side-by-side
```

**Workflow:**
1. Select task/prompt
2. Choose two models (e.g., Claude Sonnet vs GPT-4o)
3. Run both in parallel (separate branches)
4. Show side-by-side comparison
5. User picks winner, other branch discarded

### F3.6 — Progressive Autonomy

**Description:** Agent earns trust per task type → gradually more autonomy over time.

**Implementation:**
```
internal/autonomy/
  trust.go           # TrustScore — track agent reliability per task type
  levels.go          # Autonomy levels (supervised → semi → full)
  escalation.go      # When to escalate to human
```

**Autonomy Levels:**
- `supervised` — Every change requires approval (default for new task types)
- `semi` — Batch approval (review all changes at once)
- `full` — Auto-apply changes, notify after (earned after 5+ successful runs)

### F3.7 — Specialist Routing

**Description:** Auto-route task to best model based on historical success data.

**Implementation:**
```
internal/router/
  specialist.go      # SpecialistRouter — route to best model
  history.go         # Historical performance data
```

**Routing Logic:**
- Frontend tasks → model with best frontend success rate
- Backend tasks → model with best backend success rate
- Quick fixes → cheapest model that can handle it
- Complex refactors → most capable model
- Data-driven: based on your historical success rates

### F3.8 — Release Automation

**Description:** Auto-generate changelog, bump version, create tag, draft release notes.

**Implementation:**
```
internal/automation/
  release.go         # ReleaseAutomation — full release pipeline
  changelog.go       # Changelog generation from commits
  version.go         # Version bumping (semver)
```

**Pipeline:**
1. Analyze commits since last release
2. Categorize (feat, fix, chore, breaking)
3. Generate changelog
4. Bump version (patch/minor/major)
5. Create git tag
6. Draft GitHub release
7. User reviews and publishes

---

## Phase 4: Analytics (Weeks 13-16)

> **Goal:** Data-driven decisions. Agent knows your style. You see what's working.

### F4.1 — Model ROI Dashboard

**Description:** Cost vs success rate per model per task type.

**Data Tracked:**
- Cost per task per model
- Success rate per model per task type
- Time to completion per model
- Rejection rate (user rejected changes)
- Retry rate (needed multiple attempts)

**Display:**
```
┌─────────────────────────────────────────────────────────────┐
│ Model ROI Dashboard                                         │
├──────────────┬──────────┬──────────┬──────────┬─────────────┤
│ Model        │ Frontend │ Backend  │ Tests    │ Avg Cost    │
├──────────────┼──────────┼──────────┼──────────┼─────────────┤
│ Claude Sonnet│ 92% ✓    │ 78% ✓    │ 85% ✓    │ $0.42/task  │
│ GPT-4o       │ 85% ✓    │ 88% ✓    │ 72% ⚠    │ $0.31/task  │
│ Gemini 2.5   │ 78% ⚠    │ 82% ✓    │ 90% ✓    │ $0.18/task  │
│ Claude Haiku │ 65% ⚠    │ 60% ✗    │ 70% ⚠    │ $0.05/task  │
└──────────────┴──────────┴──────────┴──────────┴─────────────┘
```

### F4.2 — Waste Detection

**Description:** "You spent $12 this week on discarded sessions" — identify inefficiency.

**Waste Categories:**
- Discarded sessions (started but never accepted)
- Rejected diffs (agent output user didn't use)
- Retry loops (same task attempted multiple times)
- Over-engineered (agent did more than asked)

**Display:**
```
┌────────────────────────────────────────┐
│ Waste Report — This Week               │
├────────────────────────────────────────┤
│ Discarded sessions: 3 ($4.20)          │
│ Rejected diffs: 12 changes ($2.80)     │
│ Retry loops: 2 sessions ($3.10)        │
│ Over-engineered: 1 session ($1.90)     │
├────────────────────────────────────────┤
│ Total waste: $12.00 (32% of spend)     │
│ Trend: ↓ 15% from last week            │
└────────────────────────────────────────┘
```

### F4.3 — Skill Auto-Extraction

**Description:** When you fix agent output, extract the correction as a reusable skill.

**Workflow:**
1. User manually edits agent output
2. HELM captures the diff (agent output → user's version)
3. LLM analyzes the diff to extract the rule/pattern
4. Suggests saving as a skill or memory entry
5. User approves → saved to prompt library or project memory

### F4.4 — Hotspot Analysis

**Description:** Files most likely to break based on change frequency, complexity, error history.

**Data Sources:**
- Git history (change frequency per file)
- Mistake journal (errors per file)
- Code complexity metrics
- Test coverage per file

**Display:**
```
┌────────────────────────────────────────────────────────────┐
│ Hotspot Analysis                                            │
├──────────────────────────┬─────────┬────────┬──────────────┤
│ File                     │ Changes │ Errors │ Risk Score   │
├──────────────────────────┼─────────┼────────┼──────────────┤
│ internal/auth/login.go   │ 47      │ 12     │ 🔴 High      │
│ internal/api/handler.go  │ 32      │ 8      │ 🟡 Medium    │
│ internal/db/query.go     │ 15      │ 2      │ 🟢 Low       │
└──────────────────────────┴─────────┴────────┴──────────────┘
```

### F4.5 — Architecture Map

**Description:** Auto-generate codebase structure, dependency graph, data flow visualization.

**Implementation:**
- Parse import statements across all files
- Build dependency graph
- Detect layers (handlers → services → repositories)
- Detect circular dependencies
- Visualize in TUI (tree view with dependency indicators)

### F4.6 — Trend Analytics

**Description:** Cost trends, productivity trends, model performance over time.

**Metrics:**
- Daily/weekly/monthly cost trends
- Tasks completed per week
- Average cost per task over time
- Success rate trends per model
- Time savings estimate

### F4.7 — Drift Detection

**Description:** Alert when actual code diverges from documented architecture or design patterns.

**Implementation:**
- Parse architecture docs (ARCHITECTURE.md, design docs)
- Compare actual code structure against documented patterns
- Flag deviations (new layers, missing abstractions, pattern violations)
- Suggest corrections or doc updates

### F4.8 — Cross-Project Memory

**Description:** Share learned patterns across your repos.

**Implementation:**
- Global memory store in `~/.helm/memory/`
- Project-specific memory in `.helm/memory/`
- When starting a new project, suggest relevant global memories
- "You always use Zod for validation" → apply to new Go project as "you always use go-playground/validator"

---

## Phase 5: Experience (Weeks 17-20)

> **Goal:** Frictionless experience. Feels like a teammate, not a tool.

### F5.1 — Session Replay

**Description:** Rewind any session, see what prompt caused what change.

**Implementation:**
- Reconstruct session timeline from JSONL
- Step through each turn: prompt → thinking → tool call → result
- See file state at any point in time
- Identify which prompt led to which change

### F5.2 — Voice Notes

**Description:** Record voice memo → transcribe → use as prompt.

**Implementation:**
- Record via terminal (arecord, sox) or paste audio file
- Transcribe via Whisper (local or API)
- Use transcription as session prompt
- Optional: attach to existing session as follow-up

### F5.3 — Natural Language Git

**Description:** `helm undo`, `helm save this approach`, `helm show what changed`.

**Commands:**
```bash
helm undo              # Revert last agent changes
helm save "approach-a" # Tag current state
helm show              # Show what changed since last save
helm compare a b       # Compare two saved states
helm restore "approach-a"  # Restore a saved state
```

### F5.4 — Mood/Auto-Pause

**Description:** Detect stuck agent → auto-pause and notify.

**Detection Signals:**
- Same tool call repeated 3+ times
- Same error message 2+ times
- No file changes in last 5 turns
- Token usage spiking without progress
- Session duration exceeding expected for task type

**Action:**
- Pause session
- Show what went wrong
- Suggest fixes (different model, adjusted prompt, manual intervention)

### F5.5 — Quality Gates

**Description:** Pre-accept validation: lint, tests, security scan, complexity check.

**Gates (configurable):**
- `lint` — Run linter, reject if violations
- `test` — Run tests, reject if failures
- `security` — Run semgrep/trivy, reject if findings
- `complexity` — Check cyclomatic complexity, reject if increased
- `build` — Run build, reject if fails

**Config:**
```toml
[quality_gates]
lint = true
test = true
security = false
complexity = false
build = true
```

### F5.6 — Token Budget

**Description:** Set max tokens per session — agent self-manages context.

**Implementation:**
- Hard cap on input + output tokens per session
- Agent receives budget info in system prompt
- Auto-prune context when approaching limit
- Alert user when budget is tight

### F5.7 — Context Inheritance

**Description:** Fork a session and inherit only relevant context.

**Implementation:**
- When forking, select which context to inherit:
  - Full context (everything)
  - Relevant files only
  - Just project memory
  - Custom selection
- Reduces token waste from unnecessary context

### F5.8 — Dependency Graph

**Description:** Visualize import chains, detect circular deps, suggest decoupling.

**Implementation:**
- Parse imports across all files
- Build directed graph
- Detect cycles
- Identify highly-coupled modules
- Suggest refactoring opportunities

---

## Phase 6: Ecosystem (Weeks 21-24)

> **Goal:** Extensible, connectable, team-ready if you want it.

### F6.1 — MCP Server

**Description:** Expose HELM as an MCP server — other agents can call it.

**Tools Exposed:**
- `helm_memory_get` — Retrieve project memory
- `helm_memory_set` — Store project memory
- `helm_session_list` — List sessions
- `helm_session_get` — Get session details
- `helm_cost_get` — Get cost data
- `helm_prompt_get` — Get prompt template

### F6.2 — Plugin System

**Description:** Custom tools, community plugins, skill marketplace.

**Plugin Types:**
- Tools (new capabilities for agents)
- Skills (prompt templates + context)
- Parsers (new session format support)
- Providers (new LLM provider adapters)

### F6.3 — CI/CD Integration

**Description:** GitHub Actions, GitLab CI — run HELM tasks as pipeline steps.

**Usage:**
```yaml
# .github/workflows/helm.yml
- name: Run HELM maintenance
  run: helm run nightly --headless
  env:
    HELM_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

### F6.4 — Team Sync (Optional)

**Description:** Share memory, prompts, benchmarks with teammates.

**Implementation:**
- PostgreSQL sync backend (optional)
- Share project memory across team
- Shared prompt library
- Team-wide model performance data

### F6.5 — Web Dashboard

**Description:** Browser-accessible view of everything.

**Implementation:**
- Embedded HTTP server
- HTMX-based dashboard
- Real-time session monitoring via SSE
- Mobile-friendly for checking status remotely

### F6.6 — API

**Description:** REST/GraphQL API for programmatic access.

**Endpoints:**
- `GET /sessions` — List sessions
- `POST /sessions` — Start new session
- `GET /sessions/:id` — Session details
- `GET /cost` — Cost data
- `GET /memory` — Project memory
- `POST /memory` — Update memory
- `GET /prompts` — Prompt library

### F6.7 — Session Comparison

**Description:** Side-by-side diff of two agent approaches to the same task.

**Implementation:**
- Select two sessions (or forks)
- Show side-by-side file changes
- Highlight differences in approach
- Metrics comparison (cost, time, quality)

### F6.8 — Performance Budget

**Description:** Warn when agent adds heavy deps, increases bundle size, adds N+1 queries.

**Checks:**
- New dependencies added
- Bundle size increase
- Database query count increase
- API response time impact
- Memory usage increase

---

## Data Model

### Core Tables

```sql
-- Sessions
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    project TEXT NOT NULL,
    prompt TEXT,
    status TEXT NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cache_read_tokens INTEGER DEFAULT 0,
    cache_write_tokens INTEGER DEFAULT 0,
    cost REAL DEFAULT 0,
    summary TEXT,
    tags TEXT,  -- JSON array
    raw_path TEXT  -- Path to raw JSONL
);

-- Messages
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_calls TEXT,  -- JSON array
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Memories
CREATE TABLE memories (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL,
    type TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    source TEXT DEFAULT 'manual',
    confidence REAL DEFAULT 0.5,
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Prompts
CREATE TABLE prompts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    tags TEXT,  -- JSON array
    complexity TEXT,
    template TEXT NOT NULL,
    variables TEXT,  -- JSON array
    source TEXT DEFAULT 'builtin',  -- 'builtin' | 'user' | 'extracted'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Cost Records
CREATE TABLE cost_records (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    project TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cache_read_tokens INTEGER,
    cache_write_tokens INTEGER,
    total_cost REAL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Mistakes
CREATE TABLE mistakes (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    type TEXT NOT NULL,
    description TEXT NOT NULL,
    context TEXT,
    correction TEXT,
    file_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Budgets
CREATE TABLE budgets (
    project TEXT PRIMARY KEY,
    daily_limit REAL,
    weekly_limit REAL,
    monthly_limit REAL,
    warning_pct REAL DEFAULT 0.8,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Model Performance
CREATE TABLE model_performance (
    id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    task_type TEXT NOT NULL,
    attempts INTEGER DEFAULT 0,
    successes INTEGER DEFAULT 0,
    total_cost REAL DEFAULT 0,
    avg_tokens INTEGER DEFAULT 0,
    avg_time_seconds INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model, task_type)
);

-- File Changes
CREATE TABLE file_changes (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    file_path TEXT NOT NULL,
    additions INTEGER,
    deletions INTEGER,
    classification TEXT,  -- 'essential' | 'incidental' | 'suspicious'
    accepted BOOLEAN
);

-- Indexes
CREATE INDEX idx_sessions_project ON sessions(project);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_started ON sessions(started_at);
CREATE INDEX idx_memories_project ON memories(project);
CREATE INDEX idx_cost_project ON cost_records(project);
CREATE INDEX idx_cost_date ON cost_records(recorded_at);
CREATE INDEX idx_mistakes_session ON mistakes(session_id);
CREATE INDEX idx_mistakes_type ON mistakes(type);

-- Full-text search
CREATE VIRTUAL TABLE sessions_fts USING fts5(prompt, summary, content='sessions');
CREATE VIRTUAL TABLE memories_fts USING fts5(key, value, content='memories');
```

---

## Project Structure

```
helm/
├── cmd/
│   └── helm/
│       └── main.go              # CLI entry point
├── internal/
│   ├── app/
│   │   └── app.go               # Top-level wiring: DB, config, services, TUI
│   ├── cmd/
│   │   ├── root.go              # Root command
│   │   ├── run.go               # helm run <prompt>
│   │   ├── init.go              # helm init
│   │   ├── memory.go            # helm memory subcommands
│   │   ├── cost.go              # helm cost
│   │   ├── prompts.go           # helm prompts
│   │   ├── status.go            # helm status
│   │   └── diff.go              # helm diff
│   ├── config/
│   │   ├── config.go            # Config struct
│   │   ├── load.go              # Load helm.toml
│   │   └── provider.go          # Provider configuration
│   ├── provider/
│   │   ├── router.go            # ProviderRouter
│   │   ├── interface.go         # Provider interface
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   ├── google.go
│   │   ├── ollama.go
│   │   ├── openrouter.go
│   │   ├── custom.go
│   │   ├── models.go
│   │   ├── fallback.go
│   │   └── pricing.go
│   ├── session/
│   │   ├── manager.go           # Session CRUD
│   │   ├── archive.go           # Session archive/search
│   │   ├── fork.go              # Session forking
│   │   ├── canonical.go         # Canonical session model
│   │   ├── parser_claude.go
│   │   ├── parser_codex.go
│   │   ├── parser_gemini.go
│   │   └── parser_opencode.go
│   ├── memory/
│   │   ├── engine.go            # MemoryEngine
│   │   ├── store.go
│   │   ├── auto.go
│   │   ├── recall.go
│   │   ├── forget.go
│   │   └── types.go
│   ├── prompt/
│   │   ├── library.go           # PromptLibrary
│   │   ├── template.go
│   │   ├── discover.go
│   │   ├── builtin.go
│   │   └── render.go
│   ├── cost/
│   │   ├── tracker.go           # CostTracker
│   │   ├── calculator.go
│   │   ├── budget.go
│   │   ├── parser.go
│   │   └── report.go
│   ├── diff/
│   │   ├── viewer.go            # DiffViewer
│   │   ├── engine.go
│   │   ├── classify.go
│   │   ├── apply.go
│   │   ├── triage.go
│   │   └── group.go
│   ├── mistake/
│   │   ├── journal.go
│   │   ├── capture.go
│   │   ├── patterns.go
│   │   └── rules.go
│   ├── retry/
│   │   ├── engine.go
│   │   ├── context.go
│   │   └── strategy.go
│   ├── context/
│   │   ├── pruner.go
│   │   ├── relevance.go
│   │   └── budget.go
│   ├── summary/
│   │   ├── generator.go
│   │   └── template.go
│   ├── pattern/
│   │   ├── detector.go
│   │   ├── conventions.go
│   │   └── style.go
│   ├── automation/
│   │   ├── scheduler.go
│   │   ├── nightly.go
│   │   ├── tasks.go
│   │   ├── issue.go
│   │   ├── pipeline.go
│   │   ├── pr.go
│   │   ├── ci.go
│   │   ├── healer.go
│   │   ├── release.go
│   │   ├── changelog.go
│   │   └── version.go
│   ├── agent/
│   │   ├── redteam.go
│   │   ├── breaker.go
│   │   ├── abtest.go
│   │   └── compare.go
│   ├── autonomy/
│   │   ├── trust.go
│   │   ├── levels.go
│   │   └── escalation.go
│   ├── router/
│   │   ├── specialist.go
│   │   └── history.go
│   ├── db/
│   │   ├── db.go                # Database connection
│   │   ├── migrations/          # SQL migrations
│   │   └── sql/                 # sqlc queries
│   ├── ui/
│   │   ├── app.go               # Bubbletea app
│   │   ├── dashboard.go
│   │   ├── session_list.go
│   │   ├── diff_view.go
│   │   ├── cost_view.go
│   │   ├── memory_view.go
│   │   ├── prompt_view.go
│   │   ├── status_bar.go
│   │   ├── keybindings.go
│   │   └── theme.go
│   ├── setup/
│   │   ├── init.go
│   │   ├── detect.go
│   │   ├── memory_build.go
│   │   ├── provider_config.go
│   │   └── prompt_suggest.go
│   ├── git/
│   │   ├── worktree.go          # Git worktree management
│   │   ├── branch.go
│   │   └── diff.go
│   ├── watch/
│   │   └── watcher.go           # File watching for session updates
│   └── version/
│       └── version.go           # Version info
├── sqlc.yaml                    # sqlc configuration
├── Taskfile.yaml                # Task runner commands
├── go.mod
├── go.sum
├── .goreleaser.yml              # Release configuration
├── install.sh                   # Install script
├── README.md
├── PLAN.md                      # This file
└── AGENTS.md                    # Agent execution instructions
```

---

## Provider Integration

### Provider Interface

```go
type Provider interface {
    Name() string
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    Stream(ctx context.Context, req ChatRequest) (<chan StreamEvent, error)
    Models() ([]ModelInfo, error)
    Cost(model string, input, output, cacheRead, cacheWrite int) float64
}

type ChatRequest struct {
    Model    string
    Messages []Message
    Tools    []Tool
    MaxTokens int
}

type ChatResponse struct {
    Message     Message
    Usage       Usage
    FinishReason string
}

type StreamEvent struct {
    Type   string // "text", "tool_call", "done", "error"
    Text   string
    ToolCall *ToolCall
    Usage  *Usage
}

type Usage struct {
    InputTokens  int
    OutputTokens int
    CacheRead    int
    CacheWrite   int
}
```

---

## Session Format Parsing

### Claude Code
- Location: `~/.claude/projects/<project-id>/`
- Format: JSONL, one event per line
- Events: user message, assistant message, tool call, tool result
- Token counts in assistant message metadata

### Codex
- Location: `~/.codex/sessions/`
- Format: JSONL, similar to Claude
- Events: conversation turns with tool executions

### Gemini CLI
- Location: `~/.gemini/sessions/`
- Format: JSONL
- Events: conversation turns

### OpenCode
- Location: `~/.opencode/sessions/`
- Format: JSONL
- Events: conversation turns

### Parser Strategy
1. Detect provider from session file location/format
2. Parse JSONL line by line
3. Normalize to canonical Message model
4. Extract token counts and cost
5. Store in SQLite
6. Keep raw file reference for replay

---

## Build & Release

### Commands

```bash
# Build
go build -o helm ./cmd/helm

# Run
go run ./cmd/helm

# Test
go test ./...

# Test single
go test ./internal/provider -run TestProviderRouter

# Lint
golangci-lint run

# Format
gofumpt -w .

# Generate SQL code
sqlc generate

# Release
goreleaser release --clean
```

### Cross-Platform Builds

Targets:
- darwin/amd64, darwin/arm64
- linux/amd64, linux/arm64
- windows/amd64

### Installation

```bash
# Homebrew (macOS)
brew install helm

# Shell script
curl -fsSL https://get.helm.sh | sh

# Go install
go install github.com/yourname/helm/cmd/helm@latest
```

---

## Testing Strategy

### Unit Tests
- Every package has `_test.go` files
- Mock providers for testing without API calls
- SQLite in-memory database for testing
- Table-driven tests for parsers

### Integration Tests
- Test session parsing against real JSONL samples
- Test provider adapters with mock responses
- Test TUI components with golden files

### Golden File Testing
- TUI rendering tests with charm.land/catwalk
- Compare rendered output against `.golden` files
- Update with `go test ./... -update`

### Test Data
- Sample JSONL files from each provider
- Sample project structures for `helm init`
- Sample diff files for diff review testing

---

## Security Model

### API Key Management
- Read from environment variables (never store in config)
- Support `.env` files (gitignored)
- Never log or display keys
- Keychain integration (macOS) for optional storage

### Data Privacy
- All data stored locally in SQLite
- No telemetry by default (opt-in)
- Session content never leaves local machine
- Provider API calls use HTTPS only

### Safe Defaults
- Budget limits prevent runaway costs
- Quality gates prevent bad code from being applied
- Auto-pause on stuck detection
- No auto-push without user approval (Phase 1-2)

---

## Performance Targets

| Metric | Target |
|--------|--------|
| TUI render latency | < 16ms (60fps) |
| Session search (10K sessions) | < 100ms |
| Memory recall | < 10ms |
| Cost calculation | < 5ms |
| Diff rendering (100 files) | < 200ms |
| Startup time | < 500ms |
| Binary size | < 30MB |
| Memory usage (idle) | < 50MB |

---

## Migration Path

### From Existing Tools

| From | Migration Path |
|------|---------------|
| Claude Code sessions | Auto-detect `~/.claude/`, import sessions |
| Codex sessions | Auto-detect `~/.codex/`, import sessions |
| Gemini sessions | Auto-detect `~/.gemini/`, import sessions |
| OpenCode sessions | Auto-detect `~/.opencode/`, import sessions |
| CLAUDE.md/AGENTS.md | Import as project memory entries |
| .cursor/rules | Import as project memory entries |

### Backward Compatibility
- HELM never modifies existing agent session files
- HELM reads but doesn't write to provider directories
- All HELM data is in `~/.helm/` and `.helm/`

---

## Competitive Analysis

### Why HELM Wins

| Dimension | Agent Deck | agentsview | engram | **HELM** |
|-----------|-----------|------------|--------|----------|
| Session Management | ✅ | ❌ | ❌ | ✅ |
| Agent Memory | ❌ | ❌ | ✅ | ✅ |
| Cost Tracking | ✅ | ❌ | ❌ | ✅ |
| Prompt Library | ❌ | ❌ | ❌ | ✅ |
| Diff Review | ⚠️ | ❌ | ❌ | ✅ |
| Smart Triage | ❌ | ❌ | ❌ | ✅ |
| Auto-Retry | ❌ | ❌ | ❌ | ✅ |
| Mistake Learning | ❌ | ❌ | ❌ | ✅ |
| Budget Alerts | ⚠️ | ❌ | ❌ | ✅ |
| Session Archive | ❌ | ✅ | ❌ | ✅ |
| Model ROI | ❌ | ❌ | ❌ | ✅ |
| Agent Benchmarking | ❌ | ❌ | ❌ | ✅ |
| Nightly Automation | ❌ | ❌ | ❌ | ✅ |
| Issue-to-PR | ❌ | ❌ | ❌ | ✅ |
| Self-Healing CI | ❌ | ❌ | ❌ | ✅ |
| **Total** | **3/15** | **1/15** | **1/15** | **15/15** |

### Unique Positioning
- **Personal-first** — Not enterprise, not team — built for solo developers
- **Unified** — One tool instead of 6 separate tools
- **Intelligent** — Learns from mistakes, auto-retries, smart triage
- **Agent-agnostic** — Works with any provider, not locked to one
- **Local-first** — No cloud, no subscription, no data leaving your machine
