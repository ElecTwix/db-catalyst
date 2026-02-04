package basic

import (
	"context"
	"database/sql"
)

const queryCreateUser string = `INSERT INTO users (username, email) VALUES (?, ?) RETURNING id;`

func (q *Queries) CreateUser(ctx context.Context, username string, email string) (CreateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateUser, username, email)
	if err != nil {
		return CreateUserRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateUserRow{}, err
		}
		return CreateUserRow{}, sql.ErrNoRows
	}
	item, err := scanCreateUserRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
