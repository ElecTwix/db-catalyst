-- User queries

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE useremail = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: ListActiveUsers :many
SELECT * FROM users WHERE is_active = true ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (username, useremail, metadata, tags)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET username = $2, useremail = $3, metadata = $4, tags = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: SearchUsersByTag :many
SELECT * FROM users WHERE $1 = ANY(tags);

-- name: SearchUsersByMetadata :many
SELECT * FROM users WHERE metadata @> $1;

-- Post queries

-- name: GetPostByID :one
SELECT * FROM posts WHERE id = $1;

-- name: ListPosts :many
SELECT * FROM posts ORDER BY created_at DESC;

-- name: ListPublishedPosts :many
SELECT * FROM posts WHERE is_published = true ORDER BY published_at DESC;

-- name: ListPostsByUser :many
SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (user_id, title, postbody, categories, is_published, published_at)
VALUES ($1, $2, $3, $4, $5, CASE WHEN $5 THEN NOW() ELSE NULL END)
RETURNING *;

-- name: UpdatePost :one
UPDATE posts
SET title = $2, postbody = $3, categories = $4, is_published = $5, 
    published_at = CASE WHEN $5 AND published_at IS NULL THEN NOW() ELSE published_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- name: IncrementPostViews :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = $1;

-- name: SearchPostsByCategory :many
SELECT * FROM posts WHERE $1 = ANY(categories) AND is_published = true;

-- Comment queries

-- name: GetCommentByID :one
SELECT * FROM comments WHERE id = $1;

-- name: ListCommentsByPost :many
SELECT * FROM comments WHERE post_id = $1 ORDER BY created_at DESC;

-- name: ListCommentsByUser :many
SELECT * FROM comments WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateComment :one
INSERT INTO comments (post_id, user_id, commentbody)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateComment :one
UPDATE comments SET commentbody = $2 WHERE id = $1 RETURNING *;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: LikeComment :exec
UPDATE comments SET likes = likes + 1 WHERE id = $1;

-- Tag queries

-- name: GetTagByID :one
SELECT * FROM tags WHERE id = $1;

-- name: GetTagByName :one
SELECT * FROM tags WHERE tagname = $1;

-- name: ListTags :many
SELECT * FROM tags ORDER BY tagname;

-- name: CreateTag :one
INSERT INTO tags (tagname, tagdescription) VALUES ($1, $2) RETURNING *;

-- name: UpdateTag :one
UPDATE tags SET tagname = $2, tagdescription = $3 WHERE id = $1 RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- Post-Tag relationship queries

-- name: AddTagToPost :exec
INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2);

-- name: RemoveTagFromPost :exec
DELETE FROM post_tags WHERE post_id = $1 AND tag_id = $2;

-- name: GetTagsForPost :many
SELECT t.* FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = $1;

-- name: GetPostsForTag :many
SELECT p.* FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = $1 AND p.is_published = true;

-- Complex queries

-- name: GetUserStats :one
SELECT 
    u.id,
    u.username,
    COUNT(DISTINCT p.id) as post_count,
    COUNT(DISTINCT c.id) as comment_count,
    COALESCE(SUM(p.view_count), 0) as total_views
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
LEFT JOIN comments c ON u.id = c.user_id
WHERE u.id = $1
GROUP BY u.id, u.username;

-- name: GetPopularPosts :many
SELECT * FROM posts 
WHERE is_published = true 
ORDER BY view_count DESC, published_at DESC
LIMIT $1;
