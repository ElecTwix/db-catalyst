package complex

import "context"

const queryGetItemWithTags string = `WITH RECURSIVE tag_list AS (
    SELECT tag FROM item_tags WHERE item_id = :id
)
SELECT i.*, (SELECT GROUP_CONCAT(tag) FROM tag_list) as tags
FROM items i
WHERE i.id = :id;`

func (q *Queries) GetItemWithTags(ctx context.Context, id *int32) (GetItemWithTagsRow, error) {
	row := q.db.QueryRowContext(ctx, queryGetItemWithTags, id)
	if err := row.Err(); err != nil {
		return GetItemWithTagsRow{}, err
	}
	var item GetItemWithTagsRow
	err := row.Scan(&item.Id, &item.Name, &item.Metadata, &item.Tags)
	if err != nil {
		return GetItemWithTagsRow{}, err
	}
	return item, nil
}
