package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const queryUpdateComment string = `UPDATE comments SET commentbody = $2 WHERE id = $1 RETURNING *;`

func (q *Queries) UpdateComment(ctx context.Context, arg2 pgtype.Text, arg1 uuid.UUID) (UpdateCommentRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateComment, arg2, arg1)
	if err != nil {
		return UpdateCommentRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdateCommentRow{}, err
		}
		return UpdateCommentRow{}, sql.ErrNoRows
	}
	item, err := scanUpdateCommentRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
