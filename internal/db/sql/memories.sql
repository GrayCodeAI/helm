-- name: CreateMemory :execresult
INSERT INTO memories (id, project, type, key, value, source, confidence)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetMemory :one
SELECT * FROM memories WHERE project = ? AND key = ?;

-- name: ListMemories :many
SELECT * FROM memories
WHERE project = ?
ORDER BY usage_count DESC, updated_at DESC;

-- name: ListMemoriesByType :many
SELECT * FROM memories
WHERE project = ? AND type = ?
ORDER BY usage_count DESC;

-- name: UpdateMemory :exec
UPDATE memories SET value = ?, confidence = ?, usage_count = usage_count + 1,
    last_used_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE id = ?;

-- name: DeleteMemory :exec
DELETE FROM memories WHERE id = ?;

-- name: SearchMemories :many
SELECT * FROM memories
WHERE project = ? AND (key LIKE '%' || ? || '%' OR value LIKE '%' || ? || '%')
ORDER BY usage_count DESC
LIMIT ?;

-- name: UpsertMemory :exec
INSERT INTO memories (id, project, type, key, value, source, confidence)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    value = excluded.value,
    confidence = excluded.confidence,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now');
