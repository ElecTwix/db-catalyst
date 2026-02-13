package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const queryUpdatePost string = `UPDATE posts
SET title = $2, postbody = $3, categories = $4, is_published = $5, 
    published_at = CASE WHEN $5 AND published_at IS NULL THEN NOW() ELSE published_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;`

func (q *Queries) UpdatePost(ctx context.Context, title pgtype.Text, postbody pgtype.Text, categories pgtype.Text, isPublished pgtype.Bool, publishedAt *any, id uuid.UUID) (UpdatePostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdatePost, title, postbody, categories, isPublished, publishedAt, id)
	if err != nil {
		return UpdatePostRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdatePostRow{}, err
		}
		return UpdatePostRow{}, sql.ErrNoRows
	}
	item, err := scanUpdatePostRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
