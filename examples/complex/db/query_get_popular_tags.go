package complexdb

import "context"

const queryGetPopularTags string = `SELECT 
    t.id, t.name, t.description,
    (SELECT COUNT(*) FROM post_tags pt WHERE pt.tag_id = t.id) as post_count
FROM tags t
ORDER BY post_count DESC
LIMIT ?;`

func (q *Queries) GetPopularTags(ctx context.Context, arg1 *int32) ([]GetPopularTagsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetPopularTags, arg1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPopularTagsRow
	for rows.Next() {
		item, err := scanGetPopularTagsRow(rows)
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
