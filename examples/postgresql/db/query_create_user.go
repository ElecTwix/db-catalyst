package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgtype"
)

const queryCreateUser string = `INSERT INTO users (username, useremail, metadata, tags)
VALUES ($1, $2, $3, $4)
RETURNING *;`

func (q *Queries) CreateUser(ctx context.Context, arg1 pgtype.Text, arg2 pgtype.Text, arg3 *[]byte, arg4 pgtype.Text) (CreateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateUser, arg1, arg2, arg3, arg4)
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
