package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryListCommentsByUser string = `SELECT * FROM comments WHERE user_id = $1 ORDER BY created_at DESC;`

func (q *Queries) ListCommentsByUser(ctx context.Context, userId *uuid.UUID) ([]ListCommentsByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListCommentsByUser, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCommentsByUserRow
	for rows.Next() {
		item, err := scanListCommentsByUserRow(rows)
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
