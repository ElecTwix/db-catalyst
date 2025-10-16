-- name: FindUser :one
SELECT id, name FROM users WHERE id = ?;
