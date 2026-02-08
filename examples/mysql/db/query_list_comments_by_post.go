package mysqlblog

import "context"

const queryListCommentsByPost string = `SELECT * FROM comments WHERE post_id = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListCommentsByPost(ctx context.Context, postId int32, limit *any) ([]ListCommentsByPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListCommentsByPost, postId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCommentsByPostRow
	for rows.Next() {
		item, err := scanListCommentsByPostRow(rows)
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
