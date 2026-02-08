package mysqlblog

import (
	"context"
	"database/sql"
)

const queryDeleteComment string = `DELETE FROM comments WHERE id = ?;`

func (q *Queries) DeleteComment(ctx context.Context, id int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteComment, id)
}
