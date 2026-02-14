package mysqlblog

import (
	"context"
	"database/sql"
)

type ListPostsParams struct {
	Status sql.NullString
	Limit  any
}

const queryListPosts string = `SELECT * FROM posts WHERE status = ? ORDER BY created_at DESC LIMIT ?;`

func (q *Queries) ListPosts(ctx context.Context, arg ListPostsParams) ([]ListPostsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListPosts, arg.Status, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPostsRow
	for rows.Next() {
		item, err := scanListPostsRow(rows)
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
