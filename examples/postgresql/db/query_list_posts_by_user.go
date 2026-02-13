package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryListPostsByUser string = `SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC;`

func (q *Queries) ListPostsByUser(ctx context.Context, userId *uuid.UUID) ([]ListPostsByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListPostsByUser, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPostsByUserRow
	for rows.Next() {
		item, err := scanListPostsByUserRow(rows)
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
