-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: CreateUser :execresult
INSERT INTO users (email, username, password_hash, status)
VALUES (?, ?, ?, ?);

-- name: ListUsers :many
SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ?;
