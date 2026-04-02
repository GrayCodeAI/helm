-- name: CreateCostRecord :execresult
INSERT INTO cost_records (id, session_id, project, provider, model, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, total_cost)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCostBySession :one
SELECT CAST(COALESCE(SUM(total_cost), 0) AS REAL) AS total_cost, CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens, CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens
FROM cost_records WHERE session_id = ?;

-- name: GetCostByProject :one
SELECT CAST(COALESCE(SUM(total_cost), 0) AS REAL) AS total_cost, CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens, CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens
FROM cost_records WHERE project = ?;

-- name: GetCostByProjectToday :one
SELECT CAST(COALESCE(SUM(total_cost), 0) AS REAL) AS total_cost, CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens, CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens
FROM cost_records WHERE project = ? AND date(recorded_at) = date('now');

-- name: GetCostByProjectWeek :one
SELECT CAST(COALESCE(SUM(total_cost), 0) AS REAL) AS total_cost, CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens, CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens
FROM cost_records WHERE project = ? AND recorded_at >= date('now', '-7 days');

-- name: GetCostByProjectMonth :one
SELECT CAST(COALESCE(SUM(total_cost), 0) AS REAL) AS total_cost, CAST(COALESCE(SUM(input_tokens), 0) AS INTEGER) AS input_tokens, CAST(COALESCE(SUM(output_tokens), 0) AS INTEGER) AS output_tokens
FROM cost_records WHERE project = ? AND recorded_at >= date('now', '-30 days');

-- name: ListCostRecords :many
SELECT * FROM cost_records
WHERE project = ?
ORDER BY recorded_at DESC
LIMIT ?;

-- name: ListCostRecordsByDate :many
SELECT date(recorded_at) as day, SUM(total_cost) as total, SUM(input_tokens) as input, SUM(output_tokens) as output
FROM cost_records
WHERE project = ? AND recorded_at >= date('now', '-30 days')
GROUP BY day
ORDER BY day DESC;

-- name: GetBudget :one
SELECT * FROM budgets WHERE project = ?;

-- name: UpsertBudget :exec
INSERT INTO budgets (project, daily_limit, weekly_limit, monthly_limit, warning_pct)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(project) DO UPDATE SET
    daily_limit = excluded.daily_limit,
    weekly_limit = excluded.weekly_limit,
    monthly_limit = excluded.monthly_limit,
    warning_pct = excluded.warning_pct,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now');
