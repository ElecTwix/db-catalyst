-- name: CreateUser :one
INSERT INTO users (email)
VALUES (?)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
