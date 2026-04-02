-- name: CreatePrompt :execresult
INSERT INTO prompts (id, name, description, tags, complexity, template, variables, source)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPrompt :one
SELECT * FROM prompts WHERE name = ?;

-- name: ListPrompts :many
SELECT * FROM prompts ORDER BY name ASC;

-- name: ListPromptsBySource :many
SELECT * FROM prompts WHERE source = ? ORDER BY name ASC;

-- name: UpdatePrompt :exec
UPDATE prompts SET description = ?, tags = ?, complexity = ?, template = ?, variables = ?
WHERE name = ?;

-- name: DeletePrompt :exec
DELETE FROM prompts WHERE name = ?;

-- name: CreateMistake :execresult
INSERT INTO mistakes (id, session_id, type, description, context, correction, file_path)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListMistakes :many
SELECT * FROM mistakes
WHERE session_id = ?
ORDER BY created_at DESC;

-- name: ListMistakesByType :many
SELECT * FROM mistakes
WHERE type = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: CountMistakesByType :one
SELECT COUNT(*) FROM mistakes WHERE type = ?;

-- name: CreateFileChange :execresult
INSERT INTO file_changes (id, session_id, file_path, additions, deletions, classification, accepted)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListFileChanges :many
SELECT * FROM file_changes WHERE session_id = ? ORDER BY file_path ASC;

-- name: UpsertModelPerformance :exec
INSERT INTO model_performance (id, model, task_type, attempts, successes, total_cost, avg_tokens, avg_time_seconds)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(model, task_type) DO UPDATE SET
    attempts = excluded.attempts,
    successes = excluded.successes,
    total_cost = excluded.total_cost,
    avg_tokens = excluded.avg_tokens,
    avg_time_seconds = excluded.avg_time_seconds,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now');

-- name: GetModelPerformance :many
SELECT * FROM model_performance
WHERE model = ?
ORDER BY task_type ASC;

-- name: ListModelPerformance :many
SELECT * FROM model_performance
ORDER BY successes DESC;

-- Sessions
-- name: CreateSession :execresult
INSERT INTO sessions (id, provider, model, project, prompt, status)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM sessions
WHERE project = ?
ORDER BY started_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateSessionStatus :exec
UPDATE sessions SET status = ?, ended_at = ? WHERE id = ?;

-- name: UpdateSessionCost :exec
UPDATE sessions SET
    input_tokens = ?, output_tokens = ?, cache_read_tokens = ?, cache_write_tokens = ?,
    cost = ?
WHERE id = ?;

-- Memories
-- name: CreateMemory :execresult
INSERT INTO memories (id, project, type, key, value, source, confidence)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetMemory :one
SELECT * FROM memories WHERE project = ? AND key = ?;

-- name: ListMemories :many
SELECT * FROM memories WHERE project = ? ORDER BY updated_at DESC;

-- name: ListMemoriesByType :many
SELECT * FROM memories WHERE project = ? AND type = ? ORDER BY updated_at DESC;

-- name: UpdateMemoryUsage :exec
UPDATE memories SET usage_count = usage_count + 1, last_used_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?;

-- name: DeleteMemory :exec
DELETE FROM memories WHERE project = ? AND key = ?;

-- Cost Records
-- name: RecordCost :execresult
INSERT INTO cost_records (id, session_id, project, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, total_cost)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListCostRecords :many
SELECT * FROM cost_records WHERE project = ? ORDER BY recorded_at DESC LIMIT ? OFFSET ?;

-- name: GetDailyCost :one
SELECT COALESCE(SUM(total_cost), 0) FROM cost_records WHERE project = ? AND date(recorded_at) = date('now');

-- name: GetWeeklyCost :one
SELECT COALESCE(SUM(total_cost), 0) FROM cost_records WHERE project = ? AND recorded_at >= datetime('now', '-7 days');

-- name: GetMonthlyCost :one
SELECT COALESCE(SUM(total_cost), 0) FROM cost_records WHERE project = ? AND recorded_at >= datetime('now', '-30 days');

-- Budgets
-- name: SetBudget :exec
INSERT INTO budgets (project, daily_limit, weekly_limit, monthly_limit, warning_pct)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(project) DO UPDATE SET
    daily_limit = excluded.daily_limit,
    weekly_limit = excluded.weekly_limit,
    monthly_limit = excluded.monthly_limit,
    warning_pct = excluded.warning_pct,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now');

-- name: GetBudget :one
SELECT * FROM budgets WHERE project = ?;
