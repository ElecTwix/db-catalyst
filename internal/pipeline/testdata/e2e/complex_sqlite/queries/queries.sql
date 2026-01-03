-- name: GetItemWithTags :one
WITH RECURSIVE tag_list AS (
    SELECT tag FROM item_tags WHERE item_id = :id
)
SELECT i.*, (SELECT GROUP_CONCAT(tag) FROM tag_list) as tags
FROM items i
WHERE i.id = :id;

-- name: DeleteTag :exec
DELETE FROM tags WHERE tag = :tag;
