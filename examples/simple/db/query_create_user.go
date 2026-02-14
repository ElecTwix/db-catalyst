package simpledb

import (
	"context"
	"database/sql"
)

type CreateUserParams struct {
	Name  string
	Email sql.NullString
}

const queryCreateUser string = `INSERT INTO users (name, email)
VALUES (?, ?)
RETURNING *;`

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateUser, arg.Name, arg.Email)
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
