package postgresqldb

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreatePostParams struct {
	UserId  *uuid.UUID
	UserId2 pgtype.Text
	UserId3 pgtype.Text
	UserId4 pgtype.Text
	UserId5 pgtype.Bool
	UserId6 *time.Time
}

const queryCreatePost string = `INSERT INTO posts (user_id, title, postbody, categories, is_published, published_at)
VALUES ($1, $2, $3, $4, $5, CASE WHEN $5 THEN NOW() ELSE NULL END)
RETURNING *;`

func (q *Queries) CreatePost(ctx context.Context, arg CreatePostParams) (CreatePostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreatePost, arg.UserId, arg.UserId2, arg.UserId3, arg.UserId4, arg.UserId5, arg.UserId6)
	if err != nil {
		return CreatePostRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreatePostRow{}, err
		}
		return CreatePostRow{}, sql.ErrNoRows
	}
	item, err := scanCreatePostRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
