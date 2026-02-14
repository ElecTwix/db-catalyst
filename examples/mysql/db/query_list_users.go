package mysqlblog

import (
	"context"
	"database/sql"
)

type ListUsersParams struct {
	Status sql.NullString
	Limit  any
}

const queryListUsers string = `SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListUsers(ctx context.Context, arg ListUsersParams) ([]ListUsersRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListUsers, arg.Status, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListUsersRow
	for rows.Next() {
		item, err := scanListUsersRow(rows)
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
