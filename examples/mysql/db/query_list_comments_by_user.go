package mysqlblog

import "context"

type ListCommentsByUserParams struct {
	UserId int32
	Limit  any
}

const queryListCommentsByUser string = `SELECT * FROM comments WHERE user_id = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListCommentsByUser(ctx context.Context, arg ListCommentsByUserParams) ([]ListCommentsByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListCommentsByUser, arg.UserId, arg.Limit)
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
