-- name: GetAuthorWithPostCount :one
SELECT 
    a.id,
    a.name,
    a.email,
    a.bio,
    (SELECT COUNT(*) FROM posts WHERE posts.author_id = a.id) as total_posts
FROM authors a
WHERE a.id = ?;

-- name: ListPosts :many
SELECT * FROM posts WHERE published = 1 ORDER BY created_at DESC;

-- name: GetPost :one
SELECT * FROM posts WHERE id = ?;

-- name: GetPostTags :many
SELECT t.id, t.name, t.description
FROM tags t
WHERE t.id IN (SELECT pt.tag_id FROM post_tags pt WHERE pt.post_id = ?);

-- name: SearchPosts :many
SELECT *
FROM posts
WHERE published = 1
  AND (title LIKE '%' || ? || '%' OR content LIKE '%' || ? || '%')
ORDER BY view_count DESC
LIMIT ? OFFSET ?;

-- name: GetPostsByTag :many
SELECT p.id, p.author_id, p.title, p.content, p.published, p.view_count, p.created_at, p.updated_at
FROM posts p
WHERE p.id IN (
    SELECT pt.post_id 
    FROM post_tags pt 
    WHERE pt.tag_id = (SELECT t.id FROM tags t WHERE t.name = ?)
)
AND p.published = 1
ORDER BY p.created_at DESC;

-- name: GetPopularTags :many
SELECT 
    t.id, t.name, t.description,
    (SELECT COUNT(*) FROM post_tags pt WHERE pt.tag_id = t.id) as post_count
FROM tags t
ORDER BY post_count DESC
LIMIT ?;

-- name: CreatePost :one
INSERT INTO posts (author_id, title, content, published)
VALUES (?, ?, ?, ?)
RETURNING id, author_id, title, content, published, view_count, created_at, updated_at;

-- name: AddTagToPost :exec
INSERT INTO post_tags (post_id, tag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: IncrementViewCount :exec
UPDATE posts
SET view_count = view_count + 1
WHERE id = ?;

-- name: GetAuthorStats :one
SELECT 
    a.id,
    a.name,
    (SELECT COUNT(*) FROM posts p WHERE p.author_id = a.id) as total_posts,
    (SELECT COALESCE(SUM(p.view_count), 0) FROM posts p WHERE p.author_id = a.id) as total_views
FROM authors a
WHERE a.id = ?;

-- name: ListUnpublishedPosts :many
SELECT * FROM posts WHERE published = 0 ORDER BY created_at DESC;
