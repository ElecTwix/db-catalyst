-- name: GetComment :one
SELECT * FROM comments WHERE id = ?;

-- name: CreateComment :execresult
INSERT INTO comments (post_id, user_id, content)
VALUES (?, ?, ?);

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = ?;

-- name: ListCommentsByPost :many
SELECT * FROM comments WHERE post_id = ? ORDER BY created_at DESC LIMIT ?;

-- name: ListCommentsByUser :many
SELECT * FROM comments WHERE user_id = ? ORDER BY created_at DESC LIMIT ?;

-- name: CountCommentsByPost :one
SELECT COUNT(*) AS count FROM comments WHERE post_id = ?;
