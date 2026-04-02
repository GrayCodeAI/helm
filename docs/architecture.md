# HELM Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         HELM CLI/TUI                         │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Dashboard│  │ Sessions │  │   Cost   │  │  Memory  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
├─────────────────────────────────────────────────────────────┤
│                    Application Layer                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Provider │  │ Session  │  │  Prompt  │  │  Diff    │   │
│  │  Router  │  │ Manager  │  │ Library  │  │  Engine  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   Cost   │  │  Memory  │  │  Mistake │  │  Retry   │   │
│  │ Tracker  │  │  Engine  │  │ Journal  │  │  Engine  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
├─────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Database │  │  Config  │  │  Logger  │  │  Cache   │   │
│  │ (SQLite) │  │  System  │  │  (slog)  │  │  Layer   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Trace   │  │ Breaker  │  │ Feature  │  │  Auth    │   │
│  │  System  │  │ Pattern  │  │  Flags   │  │  (JWT)   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
├─────────────────────────────────────────────────────────────┤
│                    External Integrations                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │Anthropic │  │  OpenAI  │  │  Google  │  │  Ollama  │   │
│  │   API    │  │   API    │  │   API    │  │  Local   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │OpenRouter│  │ GitHub   │  │  GitLab  │  │   MCP    │   │
│  │   API    │  │   API    │  │   API    │  │  Server  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
internal/
├── app/              # Application entry point
├── auth/             # JWT authentication & RBAC
├── automation/       # Nightly maintenance, CI, issue-to-PR
├── backup/           # Database backup & restore
├── breaker/          # Circuit breaker pattern
├── cache/            # Caching layer (memory/Redis)
├── cmd/              # CLI commands (cobra)
├── config/           # Configuration management
├── cost/             # Cost tracking & budgets
├── db/               # SQLite database layer
├── diff/             # Diff engine & triage
├── errors/           # Error handling with codes
├── feature/          # Feature flags
├── git/              # Git operations
├── health/           # Health checks & probes
├── logger/           # Structured logging (slog)
├── memory/           # Project memory engine
├── middleware/       # HTTP middleware
├── mistake/          # Mistake journal
├── plugin/           # Plugin system
├── prompt/           # Prompt library
├── provider/         # Provider adapters
├── pprof/            # Profiling endpoints
├── retention/        # Data retention policies
├── retry/            # Retry with exponential backoff
├── session/          # Session management
├── shutdown/         # Graceful shutdown
├── sync/             # Team synchronization
├── trace/            # Distributed tracing
├── ui/               # TUI dashboard (bubbletea)
├── voice/            # Voice notes
├── watch/            # File watching & stuck detection
└── webhook/          # Webhook notifications
```

## Data Flow

```
User Input → CLI Command → App Layer → Provider Router → External API
    ↓              ↓           ↓           ↓              ↓
  TUI/Web      Parse Args   Validate   Select Model   Get Response
    ↓              ↓           ↓           ↓              ↓
  Display      Execute     Process     Route/Fallback  Parse Response
    ↓              ↓           ↓           ↓              ↓
  Update UI    Store Data  Update Cost  Track Tokens   Store Session
```

## Key Design Decisions

1. **TUI-First**: Primary interface is terminal-based using Bubbletea
2. **SQLite Storage**: Zero-dependency local storage with modernc.org/sqlite
3. **Provider Agnostic**: Canonical session model with provider-specific adapters
4. **Auto-Learning**: Memory engine learns from sessions automatically
5. **Cost Tracking**: Real-time cost calculation with budget enforcement
6. **Extensible**: Plugin system for custom tools and providers

## Security

- API keys stored in environment variables, never in config files
- JWT authentication for web API
- RBAC for team features
- Circuit breaker pattern for external API calls
- Rate limiting middleware

## Performance

- WAL mode for SQLite concurrent access
- Connection pooling for database
- Caching layer for hot data
- Exponential backoff for retries
- Graceful shutdown handling
