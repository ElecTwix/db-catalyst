package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryListCommentsByPost string = `SELECT * FROM comments WHERE post_id = $1 ORDER BY created_at DESC;`

func (q *Queries) ListCommentsByPost(ctx context.Context, arg1 *uuid.UUID) ([]ListCommentsByPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListCommentsByPost, arg1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCommentsByPostRow
	for rows.Next() {
		item, err := scanListCommentsByPostRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
