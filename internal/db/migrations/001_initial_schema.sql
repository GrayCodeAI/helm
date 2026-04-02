-- Initial schema for HELM database

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    project TEXT NOT NULL,
    prompt TEXT,
    status TEXT NOT NULL DEFAULT 'running',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cache_read_tokens INTEGER DEFAULT 0,
    cache_write_tokens INTEGER DEFAULT 0,
    cost REAL DEFAULT 0,
    summary TEXT,
    tags TEXT,
    raw_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_calls TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);

-- Memories table
CREATE TABLE IF NOT EXISTS memories (
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

CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_project_key ON memories(project, key);

-- FTS5 virtual table for memories
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    key,
    value,
    content='memories',
    content_rowid='rowid'
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS memories_fts_insert AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, key, value)
    VALUES (new.rowid, new.key, new.value);
END;

CREATE TRIGGER IF NOT EXISTS memories_fts_update AFTER UPDATE ON memories BEGIN
    UPDATE memories_fts SET key = new.key, value = new.value
    WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS memories_fts_delete AFTER DELETE ON memories BEGIN
    DELETE FROM memories_fts WHERE rowid = old.rowid;
END;

-- Prompts table
CREATE TABLE IF NOT EXISTS prompts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    tags TEXT,
    complexity TEXT,
    template TEXT NOT NULL,
    variables TEXT,
    source TEXT DEFAULT 'builtin',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prompts_source ON prompts(source);

-- Cost records table
CREATE TABLE IF NOT EXISTS cost_records (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS idx_cost_project ON cost_records(project);
CREATE INDEX IF NOT EXISTS idx_cost_date ON cost_records(recorded_at);

-- Budgets table
CREATE TABLE IF NOT EXISTS budgets (
    project TEXT PRIMARY KEY,
    daily_limit REAL,
    weekly_limit REAL,
    monthly_limit REAL,
    warning_pct REAL DEFAULT 0.8,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Mistakes table
CREATE TABLE IF NOT EXISTS mistakes (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    description TEXT NOT NULL,
    context TEXT,
    correction TEXT,
    file_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mistakes_session ON mistakes(session_id);
CREATE INDEX IF NOT EXISTS idx_mistakes_type ON mistakes(type);

-- Model performance table
CREATE TABLE IF NOT EXISTS model_performance (
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

CREATE INDEX IF NOT EXISTS idx_model_performance_model ON model_performance(model);
CREATE INDEX IF NOT EXISTS idx_model_performance_task ON model_performance(task_type);

-- File changes table
CREATE TABLE IF NOT EXISTS file_changes (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    additions INTEGER,
    deletions INTEGER,
    classification TEXT,
    accepted INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_file_changes_session ON file_changes(session_id);
CREATE INDEX IF NOT EXISTS idx_file_changes_path ON file_changes(file_path);

-- FTS5 virtual table for sessions
CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
    prompt,
    summary,
    tags,
    content='sessions',
    content_rowid='rowid'
);

-- Triggers to keep sessions FTS index in sync
CREATE TRIGGER IF NOT EXISTS sessions_fts_insert AFTER INSERT ON sessions BEGIN
    INSERT INTO sessions_fts(rowid, prompt, summary, tags)
    VALUES (new.rowid, new.prompt, new.summary, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS sessions_fts_update AFTER UPDATE ON sessions BEGIN
    UPDATE sessions_fts SET prompt = new.prompt, summary = new.summary, tags = new.tags
    WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS sessions_fts_delete AFTER DELETE ON sessions BEGIN
    DELETE FROM sessions_fts WHERE rowid = old.rowid;
END;
