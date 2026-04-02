-- name: CreateMessage :execresult
INSERT INTO messages (session_id, role, content, tool_calls, timestamp)
VALUES (?, ?, ?, ?, ?);

-- name: GetMessagesBySession :many
SELECT * FROM messages
WHERE session_id = ?
ORDER BY timestamp ASC;

-- name: GetMessage :one
SELECT * FROM messages WHERE id = ?;

-- name: CountMessagesBySession :one
SELECT COUNT(*) FROM messages WHERE session_id = ?;

-- name: DeleteMessagesBySession :exec
DELETE FROM messages WHERE session_id = ?;
