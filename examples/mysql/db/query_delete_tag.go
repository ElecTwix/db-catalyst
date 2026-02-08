package mysqlblog

import (
	"context"
	"database/sql"
)

const queryDeleteTag string = `DELETE FROM tags WHERE id = ?;`

func (q *Queries) DeleteTag(ctx context.Context, id int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteTag, id)
}
