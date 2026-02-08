-- name: GetTag :one
SELECT * FROM tags WHERE id = ?;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ?;

-- name: CreateTag :execresult
INSERT INTO tags (name, description) VALUES (?, ?);

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: AddTagToPost :exec
INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?);

-- name: RemoveTagFromPost :exec
DELETE FROM post_tags WHERE post_id = ? AND tag_id = ?;

-- name: GetTagsForPost :many
SELECT t.* FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = ? ORDER BY t.name;

-- name: GetPostsForTag :many
SELECT p.* FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = ? AND p.status = 'published'
ORDER BY p.created_at DESC LIMIT ?;
