package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryGetTagsForPost string = `SELECT t.* FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = $1;`

func (q *Queries) GetTagsForPost(ctx context.Context, arg1 *uuid.UUID) ([]GetTagsForPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetTagsForPost, arg1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetTagsForPostRow
	for rows.Next() {
		item, err := scanGetTagsForPostRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
