package mysqlblog

import (
	"context"
	"database/sql"
)

const queryIncrementPostViews string = `UPDATE posts SET view_count = view_count + 1 WHERE id = ?;`

func (q *Queries) IncrementPostViews(ctx context.Context, id int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryIncrementPostViews, id)
}
