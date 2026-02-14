package mysqlblog

import (
	"context"
	"database/sql"
)

type ListPostsByUserParams struct {
	UserId int32
	Status sql.NullString
	Limit  any
}

const queryListPostsByUser string = `SELECT * FROM posts WHERE user_id = ? AND status = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListPostsByUser(ctx context.Context, arg ListPostsByUserParams) ([]ListPostsByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListPostsByUser, arg.UserId, arg.Status, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPostsByUserRow
	for rows.Next() {
		item, err := scanListPostsByUserRow(rows)
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
