package complexdb

import (
	"context"
	"database/sql"
)

const queryGetPost string = `SELECT * FROM posts WHERE id = ?;`

func (q *Queries) GetPost(ctx context.Context, id int32) (GetPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetPost, id)
	if err != nil {
		return GetPostRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetPostRow{}, err
		}
		return GetPostRow{}, sql.ErrNoRows
	}
	item, err := scanGetPostRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
