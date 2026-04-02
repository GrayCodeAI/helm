-- 001_initial.sql — Core schema for HELM

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    provider        TEXT NOT NULL DEFAULT 'anthropic',
    model           TEXT NOT NULL,
    project         TEXT NOT NULL,
    prompt          TEXT,
    status          TEXT NOT NULL DEFAULT 'running',
    started_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    ended_at        TEXT,
    input_tokens    INTEGER NOT NULL DEFAULT 0,
    output_tokens   INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    cost            REAL NOT NULL DEFAULT 0,
    summary         TEXT,
    tags            TEXT,
    raw_path        TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    tool_calls      TEXT,
    timestamp       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_session_ts ON messages(session_id, timestamp);

-- Memories table
CREATE TABLE IF NOT EXISTS memories (
    id              TEXT PRIMARY KEY,
    project         TEXT NOT NULL,
    type            TEXT NOT NULL,
    key             TEXT NOT NULL,
    value           TEXT NOT NULL,
    source          TEXT NOT NULL DEFAULT 'manual',
    confidence      REAL NOT NULL DEFAULT 0.5,
    usage_count     INTEGER NOT NULL DEFAULT 0,
    last_used_at    TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_project_type ON memories(project, type);

-- Prompts table
CREATE TABLE IF NOT EXISTS prompts (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    description     TEXT,
    tags            TEXT,
    complexity      TEXT,
    template        TEXT NOT NULL,
    variables       TEXT,
    source          TEXT NOT NULL DEFAULT 'builtin',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_prompts_name ON prompts(name);
CREATE INDEX IF NOT EXISTS idx_prompts_source ON prompts(source);

-- Cost records table
CREATE TABLE IF NOT EXISTS cost_records (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project         TEXT NOT NULL,
    provider        TEXT NOT NULL,
    model           TEXT NOT NULL,
    input_tokens    INTEGER,
    output_tokens   INTEGER,
    cache_read_tokens INTEGER,
    cache_write_tokens INTEGER,
    total_cost      REAL,
    recorded_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_cost_project ON cost_records(project);
CREATE INDEX IF NOT EXISTS idx_cost_date ON cost_records(recorded_at);
CREATE INDEX IF NOT EXISTS idx_cost_session ON cost_records(session_id);

-- Mistakes table
CREATE TABLE IF NOT EXISTS mistakes (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,
    description     TEXT NOT NULL,
    context         TEXT,
    correction      TEXT,
    file_path       TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_mistakes_session ON mistakes(session_id);
CREATE INDEX IF NOT EXISTS idx_mistakes_type ON mistakes(type);

-- Budgets table
CREATE TABLE IF NOT EXISTS budgets (
    project         TEXT PRIMARY KEY,
    daily_limit     REAL,
    weekly_limit    REAL,
    monthly_limit   REAL,
    warning_pct     REAL NOT NULL DEFAULT 0.8,
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Model performance table
CREATE TABLE IF NOT EXISTS model_performance (
    id              TEXT PRIMARY KEY,
    model           TEXT NOT NULL,
    task_type       TEXT NOT NULL,
    attempts        INTEGER NOT NULL DEFAULT 0,
    successes       INTEGER NOT NULL DEFAULT 0,
    total_cost      REAL NOT NULL DEFAULT 0,
    avg_tokens      INTEGER NOT NULL DEFAULT 0,
    avg_time_seconds INTEGER NOT NULL DEFAULT 0,
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    UNIQUE(model, task_type)
);

-- File changes table
CREATE TABLE IF NOT EXISTS file_changes (
    id              TEXT PRIMARY KEY,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    file_path       TEXT NOT NULL,
    additions       INTEGER,
    deletions       INTEGER,
    classification  TEXT,
    accepted        INTEGER,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_file_changes_session ON file_changes(session_id);
CREATE INDEX IF NOT EXISTS idx_file_changes_file ON file_changes(file_path);

-- Session indexes
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_provider ON sessions(provider);

-- Full-text search for sessions
CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(prompt, summary, tags, content='sessions', content_rowid='rowid');

-- Full-text search for memories
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(key, value, content='memories');

-- Triggers to keep FTS indexes in sync
CREATE TRIGGER IF NOT EXISTS sessions_ai AFTER INSERT ON sessions BEGIN
    INSERT INTO sessions_fts(rowid, prompt, summary, tags)
    VALUES (new.rowid, new.prompt, new.summary, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS sessions_ad AFTER DELETE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, prompt, summary, tags)
    VALUES ('delete', old.rowid, old.prompt, old.summary, old.tags);
END;

CREATE TRIGGER IF NOT EXISTS sessions_au AFTER UPDATE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, prompt, summary, tags)
    VALUES ('delete', old.rowid, old.prompt, old.summary, old.tags);
    INSERT INTO sessions_fts(rowid, prompt, summary, tags)
    VALUES (new.rowid, new.prompt, new.summary, new.tags);
END;
