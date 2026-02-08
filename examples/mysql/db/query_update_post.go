package mysqlblog

import (
	"context"
	"database/sql"
)

const queryUpdatePost string = `UPDATE posts SET title = ?, content = ?, status = ? WHERE id = ?;`

func (q *Queries) UpdatePost(ctx context.Context, title string, content sql.NullString, status sql.NullString, id int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryUpdatePost, title, content, status, id)
}
