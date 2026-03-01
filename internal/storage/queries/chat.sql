-- Chat Messages

-- name: CreateChatMessage :one
INSERT INTO chat_messages (user_id, role, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListChatMessages :many
SELECT * FROM chat_messages
WHERE user_id = $1
ORDER BY created_at ASC
LIMIT $2;

-- name: DeleteChatHistory :exec
DELETE FROM chat_messages WHERE user_id = $1;
