package complexdb

import (
	"context"
	"database/sql"
)

const queryDeleteAuthor string = `DELETE FROM authors WHERE id = ?;`

func (q *Queries) DeleteAuthor(ctx context.Context, id int64) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteAuthor, id)
}
