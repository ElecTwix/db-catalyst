package mysqlblog

import "context"

const queryListCommentsByUser string = `SELECT * FROM comments WHERE user_id = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListCommentsByUser(ctx context.Context, userId int32, limit *any) ([]ListCommentsByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListCommentsByUser, userId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCommentsByUserRow
	for rows.Next() {
		item, err := scanListCommentsByUserRow(rows)
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
