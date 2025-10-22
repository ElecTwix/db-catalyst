package test

import (
	"context"
	"database/sql"
)

const queryGetUser string = `SELECT id, name, trigger_type FROM users WHERE id = ?;`

func (q *Queries) GetUser(ctx context.Context, arg1 interface{}) (GetUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetUser, arg1)
	if err != nil {
		return GetUserRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetUserRow{}, err
		}
		return GetUserRow{}, sql.ErrNoRows
	}
	item, err := scanGetUserRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
