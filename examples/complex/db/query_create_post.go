package complexdb

import (
	"context"
	"database/sql"
)

const queryCreatePost string = `INSERT INTO posts (author_id, title, content, published)
VALUES (?, ?, ?, ?)
RETURNING id, author_id, title, content, published, view_count, created_at, updated_at;`

func (q *Queries) CreatePost(ctx context.Context, authorId int64, title string, content string, published int64) (CreatePostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreatePost, authorId, title, content, published)
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
