-- name: GetPost :one
SELECT * FROM posts WHERE id = ?;

-- name: CreatePost :execresult
INSERT INTO posts (user_id, title, content, status)
VALUES (?, ?, ?, ?);

-- name: UpdatePost :exec
UPDATE posts SET title = ?, content = ?, status = ? WHERE id = ?;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = ?;

-- name: ListPosts :many
SELECT * FROM posts WHERE status = ? ORDER BY created_at DESC LIMIT ?;

-- name: ListPostsByUser :many
SELECT * FROM posts WHERE user_id = ? AND status = ? ORDER BY created_at DESC LIMIT ?;

-- name: IncrementPostViews :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = ?;
