-- name: GetUser :one
SELECT id, name, trigger_type FROM users WHERE id = ?;

-- name: CreateUser :exec
INSERT INTO users (id, name, trigger_type) VALUES (?, ?, ?);