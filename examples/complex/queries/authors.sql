-- name: CreateAuthor :one
INSERT INTO authors (name, email, bio)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetAuthor :one
SELECT * FROM authors WHERE id = ?;

-- name: ListAuthors :many
SELECT * FROM authors ORDER BY name;

-- name: UpdateAuthor :one
UPDATE authors
SET name = ?, email = ?, bio = ?
WHERE id = ?
RETURNING *;

-- name: DeleteAuthor :exec
DELETE FROM authors WHERE id = ?;
