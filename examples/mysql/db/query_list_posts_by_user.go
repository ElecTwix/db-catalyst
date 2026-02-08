package mysqlblog

import (
	"context"
	"database/sql"
)

const queryListPostsByUser string = `SELECT * FROM posts WHERE user_id = ? AND status = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListPostsByUser(ctx context.Context, userId int32, status sql.NullString, limit *any) ([]ListPostsByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListPostsByUser, userId, status, limit)
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
