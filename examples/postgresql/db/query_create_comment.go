package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const queryCreateComment string = `INSERT INTO comments (post_id, user_id, commentbody)
VALUES ($1, $2, $3)
RETURNING *;`

func (q *Queries) CreateComment(ctx context.Context, postId *uuid.UUID, postId2 *uuid.UUID, postId3 pgtype.Text) (CreateCommentRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateComment, postId, postId2, postId3)
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
