package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const queryUpdateUser string = `UPDATE users
SET username = $2, useremail = $3, metadata = $4, tags = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;`

func (q *Queries) UpdateUser(ctx context.Context, username pgtype.Text, useremail pgtype.Text, metadata *[]byte, tags pgtype.Text, id uuid.UUID) (UpdateUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateUser, username, useremail, metadata, tags, id)
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
