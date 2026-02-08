package mysqlblog

import "context"

const queryGetTagsForPost string = `SELECT t.* FROM tags t
JOIN post_tags pt ON t.id = pt.tag_id
WHERE pt.post_id = ? ORDER BY t.name;`

func (q *Queries) GetTagsForPost(ctx context.Context, postId int32) ([]GetTagsForPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetTagsForPost, postId)
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
