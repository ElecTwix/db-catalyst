-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES (?, ?)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = ?, email = ?
WHERE id = ?
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;
