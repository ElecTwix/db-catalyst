package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateUserParams struct {
	Username  pgtype.Text
	Username2 pgtype.Text
	Username3 *[]byte
	Username4 pgtype.Text
}

const queryCreateUser string = `INSERT INTO users (username, useremail, metadata, tags)
VALUES ($1, $2, $3, $4)
RETURNING *;`

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateUser, arg.Username, arg.Username2, arg.Username3, arg.Username4)
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
