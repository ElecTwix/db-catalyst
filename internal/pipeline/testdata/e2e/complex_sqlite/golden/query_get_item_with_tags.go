package complex

import (
	"context"
	"database/sql"
)

const queryGetItemWithTags string = `WITH RECURSIVE tag_list AS (
    SELECT tag FROM item_tags WHERE item_id = :id
)
SELECT i.*, (SELECT GROUP_CONCAT(tag) FROM tag_list) as tags
FROM items i
WHERE i.id = :id;`

func (q *Queries) GetItemWithTags(ctx context.Context, id *int32) (GetItemWithTagsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetItemWithTags, id)
	if err != nil {
		return GetItemWithTagsRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetItemWithTagsRow{}, err
		}
		return GetItemWithTagsRow{}, sql.ErrNoRows
	}
	item, err := scanGetItemWithTagsRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
