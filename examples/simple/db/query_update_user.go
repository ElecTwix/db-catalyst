package simpledb

import (
	"context"
	"database/sql"
)

const queryUpdateUser string = `UPDATE users
SET name = ?, email = ?
WHERE id = ?
RETURNING *;`

func (q *Queries) UpdateUser(ctx context.Context, name interface{}, arg2 *interface{}, arg3 int32) (UpdateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateUser, name, arg2, arg3)
	if err != nil {
		return UpdateUserRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdateUserRow{}, err
		}
		return UpdateUserRow{}, sql.ErrNoRows
	}
	item, err := scanUpdateUserRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
