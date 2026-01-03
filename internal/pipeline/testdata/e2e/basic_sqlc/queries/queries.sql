-- name: GetUser :one
SELECT * FROM users WHERE id = :id;

-- name: ListPostsByAuthor :many
SELECT * FROM posts WHERE author_id = :author_id ORDER BY id DESC;

-- name: CreateUser :one
INSERT INTO users (username, email) VALUES (?, ?) RETURNING id;
