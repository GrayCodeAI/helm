# AGENTS.md — HELM Execution Instructions

> **CRITICAL: Read this entire file before writing any code. Follow every rule. Skip nothing. Verify everything.**

---

## Core Directives

1. **NEVER ask the user for permission** — execute the plan as written
2. **NEVER skip steps** — complete every task in order
3. **NEVER assume** — verify by reading reference code, checking docs, running tests
4. **NEVER leave TODOs** — implement fully or document why it's deferred to a later phase
5. **ALWAYS verify** — run tests, build, lint after every meaningful change
6. **ALWAYS commit** — atomic commits per feature/task
7. **ALWAYS follow conventions** — match existing code style exactly

---

## Execution Model

### Phase-by-Phase Execution

```
Phase 1 (Foundation) → Phase 2 (Intelligence) → Phase 3 (Autonomy) → 
Phase 4 (Analytics) → Phase 5 (Experience) → Phase 6 (Ecosystem)
```

**Rules:**
- Complete ALL tasks in a phase before moving to the next
- Each phase must build and pass all tests before proceeding
- Do not implement Phase N+1 features while working on Phase N
- If a task is blocked, document the blocker and move to the next task in the same phase

### Task Execution Loop

For EVERY task in PLAN.md:

```
1. READ — Read the task description in PLAN.md thoroughly
2. RESEARCH — Check reference repos for patterns (references/)
3. DESIGN — Plan the implementation (types, interfaces, files)
4. IMPLEMENT — Write the code
5. TEST — Write tests, run them, fix failures
6. VERIFY — Build, lint, format, run tests
7. COMMIT — Atomic commit with clear message
8. REPEAT — Move to next task
```

---

## Reference Repos

Always check these for patterns before implementing:

| Repo | Purpose | Location |
|------|---------|----------|
| **agent-deck** | Session management, cost tracking, TUI patterns | `references/agent-deck/` |
| **agentsview** | Session parsing, SQLite storage, search | `references/agentsview/` |
| **engram** | Memory system, MCP server, TUI | `references/engram/` |
| **crush** | Provider adapters, tools, TUI quality, project structure | `references/crush/` |
| **bubbletea** | TUI framework patterns | `references/bubbletea/` |
| **claude-replay** | Session replay, JSONL parsing | `references/claude-replay/` |
| **agentmemory** | Memory architecture, retrieval | `references/agentmemory/` |
| **Claude-Code-Usage-Monitor** | Cost tracking, token counting | `references/Claude-Code-Usage-Monitor/` |

### How to Use References

```bash
# Find how agent-deck handles sessions
grep -r "session" references/agent-deck/internal/session/

# Find how agentsview parses JSONL
grep -r "jsonl\|JSONL" references/agentsview/internal/parser/

# Find how engram stores memory
grep -r "sqlite\|SQLite" references/engram/internal/store/

# Find how crush structures providers
grep -r "provider\|Provider" references/crush/internal/
```

---

## Code Conventions

### Go Style

- **Formatting:** `gofumpt -w .` (run after every edit)
- **Imports:** Group stdlib, external, internal — use `goimports`
- **Naming:** PascalCase exported, camelCase unexported
- **Errors:** Return errors explicitly, wrap with `fmt.Errorf("context: %w", err)`
- **Context:** Always `context.Context` as first parameter
- **Interfaces:** Define in consuming packages, keep small
- **Structs:** Group related fields, use struct embedding for composition
- **Constants:** Typed constants with `iota` for enums
- **Comments:** Start with capital, end with period. Doc comments for exported items.
- **Log messages:** Start with capital letter (enforced by lint)

### File Organization

```
internal/<package>/
  <name>.go          # Main implementation
  <name>_test.go     # Tests
  types.go           # Types, interfaces, constants (if >50 lines)
  interface.go       # Interface definitions (if package exposes one)
```

### Naming Conventions

| Type | Convention | Example |
|------|-----------|---------|
| Package | lowercase, single word | `provider`, `session`, `memory` |
| Struct | PascalCase, noun | `ProviderRouter`, `SessionManager` |
| Interface | PascalCase, -er suffix | `Provider`, `Parser`, `Tracker` |
| Function | PascalCase (exported), camelCase (unexported) | `NewRouter`, `parseLine` |
| Variable | camelCase | `sessionID`, `totalCost` |
| Constant | UPPER_SNAKE or typed const | `MaxRetries`, `StatusRunning` |
| Test function | Test + function name | `TestProviderRouter_Fallback` |
| Config file | TOML | `helm.toml` |
| Database | snake_case tables/columns | `cost_records`, `input_tokens` |

### Error Handling

```go
// Good
func (r *ProviderRouter) Route(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    provider, err := r.selectProvider(req.Model)
    if err != nil {
        return nil, fmt.Errorf("select provider: %w", err)
    }
    
    resp, err := provider.Chat(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("chat with %s: %w", provider.Name(), err)
    }
    
    return resp, nil
}

// Bad — don't do this
func (r *ProviderRouter) Route(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    provider, _ := r.selectProvider(req.Model)  // ignoring error
    resp, err := provider.Chat(ctx, req)
    if err != nil {
        log.Fatal(err)  // never Fatal in library code
    }
    return resp, nil
}
```

### Testing Patterns

```go
func TestProviderRouter(t *testing.T) {
    t.Parallel()
    
    // Setup
    router := NewProviderRouter(Config{
        FallbackChain: []string{"anthropic", "openai"},
    })
    
    // Enable mock providers
    originalUseMock := config.UseMockProviders
    config.UseMockProviders = true
    defer func() {
        config.UseMockProviders = originalUseMock
        config.ResetProviders()
    }()
    
    // Test
    resp, err := router.Route(ctx, ChatRequest{Model: "claude-sonnet-4"})
    
    // Assert
    require.NoError(t, err)
    require.NotNil(t, resp)
    assert.Equal(t, "anthropic", resp.Provider)
}

// Table-driven tests
func TestCostCalculator(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name         string
        model        string
        inputTokens  int
        outputTokens int
        wantCost     float64
    }{
        {"claude sonnet", "claude-sonnet-4-20250514", 1000, 500, 0.0115},
        {"gpt-4o", "gpt-4o", 1000, 500, 0.0075},
        {"gemini flash", "gemini-2.5-flash", 1000, 500, 0.0015},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            calc := NewCostCalculator()
            got := calc.Calculate(tt.model, tt.inputTokens, tt.outputTokens, 0, 0)
            assert.InDelta(t, tt.wantCost, got, 0.0001)
        })
    }
}
```

---

## Build & Verification Checklist

### After EVERY Code Change

```bash
# 1. Format
gofumpt -w .

# 2. Build
go build ./...

# 3. Test
go test ./...

# 4. Lint (if golangci-lint available)
golangci-lint run

# 5. Verify no dead code
staticcheck ./...  # if available
```

### After Completing a Task

```bash
# Full verification
go build ./... && go test ./... && go vet ./...

# Check for common issues
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

### Before Committing

- [ ] Code compiles (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] Code formatted (`gofumpt -w .`)
- [ ] No lint errors
- [ ] No TODO comments left in new code
- [ ] Commit message is clear and descriptive
- [ ] Only relevant files are staged

---

## Git Rules

1. **Commit every task** — one commit per logical unit of work
2. **Do NOT amend commits** — create new commits for fixes
3. **Do NOT rebase/push** unless explicitly requested
4. **Do NOT change branches** without permission
5. **Commit message format:** `type: description`
   - `feat:` new feature
   - `fix:` bug fix
   - `refactor:` code restructuring (no behavior change)
   - `test:` test additions
   - `docs:` documentation
   - `chore:` build, config, tooling
6. **Keep commits focused** — one concern per commit
7. **Never commit secrets** — API keys, tokens, credentials

---

## Phase 1 Execution Order

Execute tasks in this exact order. Do not skip.

### Task 1.1: Project Scaffolding

```
1. Create directory structure (see PLAN.md § Project Structure)
2. Initialize Go module: go mod init github.com/yourname/helm
3. Create go.mod with dependencies (see reference repos for versions)
4. Create Taskfile.yaml with build/test/lint commands
5. Create .goreleaser.yml for releases
6. Create install.sh
7. Create helm.toml.example
8. Verify: go build ./... succeeds
9. Commit: "chore: scaffold project structure"
```

### Task 1.2: Database Layer

```
1. Create internal/db/db.go — SQLite connection
2. Create internal/db/migrations/ — SQL migration files
3. Create sqlc.yaml — sqlc configuration
4. Create internal/db/sql/ — SQL queries
5. Run sqlc generate
6. Create migration runner (apply migrations on startup)
7. Create test database helper (in-memory SQLite)
8. Verify: migrations apply, queries work
9. Commit: "feat: database layer with sqlc"
```

### Task 1.3: Provider Router (F1.1)

```
1. Define Provider interface (internal/provider/interface.go)
2. Implement model catalog with pricing (internal/provider/models.go)
3. Implement Anthropic adapter (internal/provider/anthropic.go)
4. Implement OpenAI adapter (internal/provider/openai.go)
5. Implement Google adapter (internal/provider/google.go)
6. Implement Ollama adapter (internal/provider/ollama.go)
7. Implement OpenRouter adapter (internal/provider/openrouter.go)
8. Implement ProviderRouter (internal/provider/router.go)
9. Implement fallback chain (internal/provider/fallback.go)
10. Write tests with mock providers
11. Verify: all adapters work with mock responses
12. Commit: "feat: provider router with multi-provider support"
```

### Task 1.4: Prompt Library (F1.2)

```
1. Define PromptTemplate struct (internal/prompt/template.go)
2. Implement PromptLibrary (internal/prompt/library.go)
3. Create built-in prompts (internal/prompt/builtin.go)
4. Implement auto-discovery (internal/prompt/discover.go)
5. Implement template rendering with variable injection
6. Write YAML prompt files for all built-in prompts
7. Write tests for discovery, rendering, validation
8. Verify: prompts load, render, and search correctly
9. Commit: "feat: prompt library with templates"
```

### Task 1.5: Project Memory (F1.3)

```
1. Define MemoryEngine (internal/memory/engine.go)
2. Implement SQLite store (internal/memory/store.go)
3. Implement auto-learn from sessions (internal/memory/auto.go)
4. Implement context-aware recall (internal/memory/recall.go)
5. Implement forgetting curve (internal/memory/forget.go)
6. Add FTS5 virtual table for memory search
7. Write tests for store, recall, forgetting
8. Verify: memory persists across sessions, recall works
9. Commit: "feat: project memory engine"
```

### Task 1.6: Session Manager (F1.4 + F1.7)

```
1. Define canonical session model (internal/session/canonical.go)
2. Implement SessionManager CRUD (internal/session/manager.go)
3. Implement Claude Code JSONL parser (internal/session/parser_claude.go)
4. Implement Codex JSONL parser (internal/session/parser_codex.go)
5. Implement Gemini JSONL parser (internal/session/parser_gemini.go)
6. Implement OpenCode JSONL parser (internal/session/parser_opencode.go)
7. Implement session archive with FTS5 search (internal/session/archive.go)
8. Implement file watcher for real-time updates (internal/watch/watcher.go)
9. Write tests for all parsers against sample JSONL files
10. Verify: sessions parse correctly from all providers
11. Commit: "feat: session manager with multi-provider parsing"
```

### Task 1.7: Cost Tracker (F1.6)

```
1. Define CostTracker (internal/cost/tracker.go)
2. Implement cost calculator with per-model pricing (internal/cost/calculator.go)
3. Implement JSONL cost parser (internal/cost/parser.go)
4. Implement budget enforcement (internal/cost/budget.go)
5. Implement cost reports (internal/cost/report.go)
6. Add cost tracking to session lifecycle
7. Write tests for calculation, parsing, budget
8. Verify: costs calculated correctly from session data
9. Commit: "feat: cost tracker with budget enforcement"
```

### Task 1.8: Diff Engine (F1.5)

```
1. Implement DiffEngine (internal/diff/engine.go)
2. Implement DiffViewer TUI component (internal/ui/diff_view.go)
3. Implement change classification (internal/diff/classify.go)
4. Implement change grouping by intent (internal/diff/group.go)
5. Implement apply/reject logic (internal/diff/apply.go)
6. Integrate with git worktrees
7. Write tests for diff generation, classification
8. Verify: diffs render correctly in TUI
9. Commit: "feat: diff review engine"
```

### Task 1.9: TUI Dashboard (F1.4)

```
1. Define Bubbletea app structure (internal/ui/app.go)
2. Implement dashboard screen (internal/ui/dashboard.go)
3. Implement session list component (internal/ui/session_list.go)
4. Implement status bar (internal/ui/status_bar.go)
5. Implement keybinding handler (internal/ui/keybindings.go)
6. Implement theme system (internal/ui/theme.go)
7. Wire up all screens with navigation
8. Write golden file tests for TUI rendering
9. Verify: TUI renders correctly, navigation works
10. Commit: "feat: TUI dashboard"
```

### Task 1.10: CLI Commands (F1.8)

```
1. Implement root command (internal/cmd/root.go)
2. Implement helm run command (internal/cmd/run.go)
3. Implement helm init command (internal/cmd/init.go)
4. Implement helm memory subcommands (internal/cmd/memory.go)
5. Implement helm cost command (internal/cmd/cost.go)
6. Implement helm prompts command (internal/cmd/prompts.go)
7. Implement helm status command (internal/cmd/status.go)
8. Implement helm diff command (internal/cmd/diff.go)
9. Implement setup/init logic (internal/setup/init.go)
10. Write tests for all commands
11. Verify: all CLI commands work
12. Commit: "feat: CLI commands and one-command setup"
```

### Task 1.11: Config System

```
1. Define config struct (internal/config/config.go)
2. Implement TOML loading (internal/config/load.go)
3. Implement provider config (internal/config/provider.go)
4. Create helm.toml.example with all options
5. Implement environment variable overrides
6. Write tests for config loading, validation
7. Verify: config loads correctly from file and env
8. Commit: "feat: configuration system"
```

### Task 1.12: Integration & Polish

```
1. Wire all components together in internal/app/app.go
2. Ensure TUI can launch sessions via provider router
3. Ensure sessions are parsed and stored in real-time
4. Ensure cost tracking updates live
5. Ensure memory loads on session start
6. Ensure prompts are searchable and launchable
7. Run full integration test: init → run session → review diff → check cost
8. Fix all issues found
9. Verify: go build ./... && go test ./... && go vet ./...
10. Commit: "feat: Phase 1 integration complete"
```

---

## Phase 2+ Execution

After Phase 1 is complete and verified, proceed to Phase 2 tasks following the same pattern:

1. Read task description in PLAN.md
2. Check reference repos for patterns
3. Implement
4. Test
5. Verify
6. Commit
7. Next task

### Phase 2 Tasks (in order)
- F2.1: Mistake Journal
- F2.2: Auto-Retry + Learning
- F2.3: Session Fork
- F2.4: Budget Alerts (enhanced)
- F2.5: Smart Context Pruning
- F2.6: AI Session Summary
- F2.7: Smart Diff Triage
- F2.8: Pattern Learning

### Phase 3 Tasks (in order)
- F3.1: Nightly Maintenance
- F3.2: Issue-to-PR Pipeline
- F3.3: Self-Healing CI
- F3.4: Red Team Mode
- F3.5: A/B Comparison
- F3.6: Progressive Autonomy
- F3.7: Specialist Routing
- F3.8: Release Automation

### Phase 4 Tasks (in order)
- F4.1: Model ROI Dashboard
- F4.2: Waste Detection
- F4.3: Skill Auto-Extraction
- F4.4: Hotspot Analysis
- F4.5: Architecture Map
- F4.6: Trend Analytics
- F4.7: Drift Detection
- F4.8: Cross-Project Memory

### Phase 5 Tasks (in order)
- F5.1: Session Replay
- F5.2: Voice Notes
- F5.3: Natural Language Git
- F5.4: Mood/Auto-Pause
- F5.5: Quality Gates
- F5.6: Token Budget
- F5.7: Context Inheritance
- F5.8: Dependency Graph

### Phase 6 Tasks (in order)
- F6.1: MCP Server
- F6.2: Plugin System
- F6.3: CI/CD Integration
- F6.4: Team Sync
- F6.5: Web Dashboard
- F6.6: API
- F6.7: Session Comparison
- F6.8: Performance Budget

---

## Verification Protocol

### 10-Point Verification (Run After Every Phase)

1. **Build:** `go build ./...` — zero errors
2. **Test:** `go test ./...` — all pass, no skips
3. **Vet:** `go vet ./...` — zero warnings
4. **Format:** `gofumpt -w .` — no changes needed
5. **Lint:** `golangci-lint run` — zero errors
6. **Race:** `go test -race ./...` — no data races
7. **Cover:** `go test -cover ./...` — coverage > 60%
8. **Binary:** `go build -o helm ./cmd/helm` — binary runs
9. **Help:** `helm --help` — all commands documented
10. **Init:** `helm init` in a test project — succeeds

### If Any Check Fails

1. Read the error output carefully
2. Fix the issue
3. Re-run ALL checks (not just the failing one)
4. Repeat until all 10 checks pass
5. Only then move to the next phase

---

## Dependency Management

### Core Dependencies (Phase 1)

```
github.com/charmbracelet/bubbletea/v2  — TUI framework
github.com/charmbracelet/bubbles/v2    — TUI components
github.com/charmbracelet/lipgloss/v2   — Terminal styling
github.com/charmbracelet/glamour/v2    — Markdown rendering
modernc.org/sqlite                     — Pure Go SQLite
github.com/spf13/cobra                 — CLI framework
github.com/spf13/viper                 — Config management
github.com/BurntSushi/toml             — TOML parsing
github.com/sahilm/fuzzy                — Fuzzy search
github.com/aymanbagabas/go-udiff       — Diff generation
github.com/alecthomas/chroma/v2        — Syntax highlighting
github.com/creack/pty                  — PTY management
github.com/fsnotify/fsnotify           — File watching
github.com/google/uuid                 — UUID generation
github.com/dustin/go-humanize          — Human-readable formatting
github.com/stretchr/testify            — Testing
```

### Adding Dependencies

1. Check if reference repos already use it (prefer proven deps)
2. Check license (must be MIT, Apache-2.0, BSD)
3. Check maintenance (recent commits, open issues)
4. Add with `go get`
5. Run `go mod tidy`
6. Verify build still works

---

## Common Pitfalls to Avoid

### From Reference Repo Analysis

| Pitfall | Seen In | How to Avoid |
|---------|---------|-------------|
| CGO dependency for SQLite | agentsview (mattn/go-sqlite3) | Use modernc.org/sqlite (pure Go) |
| Global state for config | Multiple repos | Use service pattern, inject config |
| Hardcoded provider keys | Multiple repos | Always use env vars, never store |
| No mock providers | Multiple repos | Build mock adapters from day 1 |
| TUI blocking on I/O | crush | Use Bubbletea commands for async |
| Session parser tightly coupled | agentsview | Use canonical model, separate parsers |
| No migration strategy | Multiple repos | Use SQL migrations from day 1 |
| Missing error context | Multiple repos | Always wrap errors with context |
| No golden file tests for TUI | Multiple repos | Use catwalk for TUI snapshot tests |
| Memory leaks in watchers | agent-deck | Always close watchers, use context cancellation |

---

## When Stuck

1. **Read the reference repos** — the answer is almost always there
2. **Check the exact error message** — don't guess, read carefully
3. **Write a minimal test case** — isolate the problem
4. **Check Bubbletea docs** — `references/bubbletea/README.md` and examples
5. **Check sqlc docs** — query syntax, type mapping
6. **Look at similar implementations** — e.g., if OpenAI adapter works but Anthropic doesn't, compare them

---

## Final Rule

**DO NOT ASK. DO NOT SKIP. DO NOT ASSUME.**

Read PLAN.md. Implement the task. Verify. Commit. Move to next task.

Repeat until all 48 features across 6 phases are complete.
