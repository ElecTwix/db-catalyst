package basic

import (
	"context"
	"database/sql"
)

type CreateUserParams struct {
	Username string
	Email    string
}

const queryCreateUser string = `INSERT INTO users (username, email) VALUES (?, ?) RETURNING id;`

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (int32, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateUser, arg.Username, arg.Email)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, sql.ErrNoRows
	}
	var item int32
	err = rows.Scan(&item)
	if err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
