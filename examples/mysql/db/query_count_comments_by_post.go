package mysqlblog

import (
	"context"
	"database/sql"
)

const queryCountCommentsByPost string = `SELECT COUNT(*) AS count FROM comments WHERE post_id = ?;`

func (q *Queries) CountCommentsByPost(ctx context.Context, postId int32) (int64, error) {
	rows, err := q.db.QueryContext(ctx, queryCountCommentsByPost, postId)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, sql.ErrNoRows
	}
	var item int64
	err = rows.Scan(&item)
	if err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
