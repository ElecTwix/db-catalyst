package mysqlblog

import "context"

type ListCommentsByPostParams struct {
	PostId int32
	Limit  any
}

const queryListCommentsByPost string = `SELECT * FROM comments WHERE post_id = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListCommentsByPost(ctx context.Context, arg ListCommentsByPostParams) ([]ListCommentsByPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListCommentsByPost, arg.PostId, arg.Limit)
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
