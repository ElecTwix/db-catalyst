-- Example queries with cache annotations

-- GetUser retrieves a user by ID with 5-minute cache.
-- The cache key includes the user ID.
-- @cache ttl=5m key=user:{id}
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- ListActiveUsers retrieves all active users with 1-hour cache.
-- No custom key pattern, so cache key is auto-generated.
-- @cache ttl=1h
-- name: ListActiveUsers :many
SELECT * FROM users WHERE active = true ORDER BY created_at DESC;

-- GetUserByEmail retrieves a user by email with caching.
-- @cache ttl=5m key=user:email:{email}
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- CreateUser inserts a new user (no caching, but invalidates users list).
-- This is a write operation so we don't cache it, but we invalidate
-- the cached list of users.
-- name: CreateUser :exec
INSERT INTO users (name, email, active) VALUES ($1, $2, true);

-- UpdateUser updates a user and invalidates related caches.
-- @cache invalidate=user:{id},users
-- name: UpdateUser :exec
UPDATE users SET name = $1, email = $2 WHERE id = $3;

-- GetUserPosts retrieves posts for a specific user.
-- @cache ttl=10m key=user:{userId}:posts
-- name: GetUserPosts :many
SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC;

-- GetPopularPosts retrieves popular posts with longer cache TTL.
-- @cache ttl=30m key=posts:popular:{limit}
-- name: GetPopularPosts :many
SELECT * FROM posts WHERE likes > 100 ORDER BY created_at DESC LIMIT $1;
