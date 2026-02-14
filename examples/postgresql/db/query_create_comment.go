package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCommentParams struct {
	PostId  *uuid.UUID
	PostId2 *uuid.UUID
	PostId3 pgtype.Text
}

const queryCreateComment string = `INSERT INTO comments (post_id, user_id, commentbody)
VALUES ($1, $2, $3)
RETURNING *;`

func (q *Queries) CreateComment(ctx context.Context, arg CreateCommentParams) (CreateCommentRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateComment, arg.PostId, arg.PostId2, arg.PostId3)
	if err != nil {
		return CreateCommentRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateCommentRow{}, err
		}
		return CreateCommentRow{}, sql.ErrNoRows
	}
	item, err := scanCreateCommentRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
