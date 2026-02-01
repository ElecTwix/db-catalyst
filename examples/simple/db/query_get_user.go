package simpledb

import (
	"context"
	"database/sql"
)

const queryGetUser string = `SELECT * FROM users WHERE id = ?;`

func (q *Queries) GetUser(ctx context.Context, id int32) (GetUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetUser, id)
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
