package complexdb

import "context"

const queryGetPostsByTag string = `SELECT p.id, p.author_id, p.title, p.content, p.published, p.view_count, p.created_at, p.updated_at
FROM posts p
WHERE p.id IN (
    SELECT pt.post_id 
    FROM post_tags pt 
    WHERE pt.tag_id = (SELECT t.id FROM tags t WHERE t.name = ?)
)
AND p.published = 1
ORDER BY p.created_at DESC;`

func (q *Queries) GetPostsByTag(ctx context.Context, name interface{}) ([]GetPostsByTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetPostsByTag, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPostsByTagRow
	for rows.Next() {
		item, err := scanGetPostsByTagRow(rows)
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
