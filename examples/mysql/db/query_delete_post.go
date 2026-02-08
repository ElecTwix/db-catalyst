package mysqlblog

import (
	"context"
	"database/sql"
)

const queryDeletePost string = `DELETE FROM posts WHERE id = ?;`

func (q *Queries) DeletePost(ctx context.Context, id int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeletePost, id)
}
