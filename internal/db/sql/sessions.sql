-- name: CreateSession :execresult
INSERT INTO sessions (
    id, provider, model, project, prompt, status, input_tokens, output_tokens,
    cache_read_tokens, cache_write_tokens, cost, summary, tags, raw_path
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM sessions
WHERE project = ?
ORDER BY started_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateSessionStatus :exec
UPDATE sessions SET status = ?, ended_at = COALESCE(?, ended_at)
WHERE id = ?;

-- name: UpdateSessionCost :exec
UPDATE sessions SET
    input_tokens = ?, output_tokens = ?,
    cache_read_tokens = ?, cache_write_tokens = ?,
    cost = ?
WHERE id = ?;

-- name: UpdateSessionSummary :exec
UPDATE sessions SET summary = ? WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: CountSessions :one
SELECT COUNT(*) FROM sessions WHERE project = ?;

-- name: ListSessionsByStatus :many
SELECT * FROM sessions
WHERE project = ? AND status = ?
ORDER BY started_at DESC;

-- name: ListRecentSessions :many
SELECT * FROM sessions
ORDER BY started_at DESC
LIMIT ?;

-- name: SearchSessions :many
SELECT * FROM sessions
WHERE prompt LIKE '%' || ? || '%' OR summary LIKE '%' || ? || '%'
LIMIT ?;
