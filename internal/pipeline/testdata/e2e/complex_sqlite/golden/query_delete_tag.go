package complex

import (
	"context"
	"database/sql"
)

const queryDeleteTag string = `DELETE FROM tags WHERE tag = :tag;`

func (q *Queries) DeleteTag(ctx context.Context, tag string) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteTag, tag)
}
