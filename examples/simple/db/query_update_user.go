package simpledb

import (
	"context"
	"database/sql"
)

type UpdateUserParams struct {
	Name  string
	Email sql.NullString
	Id    int64
}

const queryUpdateUser string = `UPDATE users
SET name = ?, email = ?
WHERE id = ?
RETURNING *;`

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (UpdateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateUser, arg.Name, arg.Email, arg.Id)
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
