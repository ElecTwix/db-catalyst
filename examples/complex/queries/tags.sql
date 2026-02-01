-- name: CreateTag :one
INSERT INTO tags (name, description)
VALUES (?, ?)
RETURNING *;

-- name: GetTag :one
SELECT * FROM tags WHERE id = ?;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ?;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;
