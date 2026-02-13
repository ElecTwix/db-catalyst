package postgresqldb

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const queryCreatePost string = `INSERT INTO posts (user_id, title, postbody, categories, is_published, published_at)
VALUES ($1, $2, $3, $4, $5, CASE WHEN $5 THEN NOW() ELSE NULL END)
RETURNING *;`

func (q *Queries) CreatePost(ctx context.Context, userId *uuid.UUID, userId2 pgtype.Text, userId3 pgtype.Text, userId4 pgtype.Text, userId5 pgtype.Bool, userId6 *time.Time) (CreatePostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreatePost, userId, userId2, userId3, userId4, userId5, userId6)
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
