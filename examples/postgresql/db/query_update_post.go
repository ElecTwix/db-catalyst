package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UpdatePostParams struct {
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	IsPublished pgtype.Bool
	PublishedAt any
	Id          uuid.UUID
}

const queryUpdatePost string = `UPDATE posts
SET title = $2, postbody = $3, categories = $4, is_published = $5, 
    published_at = CASE WHEN $5 AND published_at IS NULL THEN NOW() ELSE published_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;`

func (q *Queries) UpdatePost(ctx context.Context, arg UpdatePostParams) (UpdatePostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdatePost, arg.Title, arg.Postbody, arg.Categories, arg.IsPublished, arg.PublishedAt, arg.Id)
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
