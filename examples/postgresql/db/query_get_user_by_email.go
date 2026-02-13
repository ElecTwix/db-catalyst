package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgtype"
)

const queryGetUserByEmail string = `SELECT * FROM users WHERE useremail = $1;`

func (q *Queries) GetUserByEmail(ctx context.Context, useremail pgtype.Text) (GetUserByEmailRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetUserByEmail, useremail)
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
