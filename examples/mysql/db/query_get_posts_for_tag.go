package mysqlblog

import "context"

const queryGetPostsForTag string = `SELECT p.* FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = ? AND p.status = 'published'
ORDER BY p.created_at DESC LIMIT ?;`

func (q *Queries) GetPostsForTag(ctx context.Context, tagId int32, limit *any) ([]GetPostsForTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetPostsForTag, tagId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPostsForTagRow
	for rows.Next() {
		item, err := scanGetPostsForTagRow(rows)
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
