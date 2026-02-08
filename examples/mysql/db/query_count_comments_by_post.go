package mysqlblog

import (
	"context"
	"database/sql"
)

const queryCountCommentsByPost string = `SELECT COUNT(*) AS count FROM comments WHERE post_id = ?;`

func (q *Queries) CountCommentsByPost(ctx context.Context, postId int32) (CountCommentsByPostRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCountCommentsByPost, postId)
	if err != nil {
		return CountCommentsByPostRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CountCommentsByPostRow{}, err
		}
		return CountCommentsByPostRow{}, sql.ErrNoRows
	}
	item, err := scanCountCommentsByPostRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
