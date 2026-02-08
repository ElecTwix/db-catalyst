package mysqlblog

import (
	"context"
	"database/sql"
)

const queryGetUserByEmail string = `SELECT * FROM users WHERE email = ?;`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetUserByEmail, email)
	if err != nil {
		return GetUserByEmailRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetUserByEmailRow{}, err
		}
		return GetUserByEmailRow{}, sql.ErrNoRows
	}
	item, err := scanGetUserByEmailRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
