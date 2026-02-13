package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryIncrementPostViews string = `UPDATE posts SET view_count = view_count + 1 WHERE id = $1;`

func (q *Queries) IncrementPostViews(ctx context.Context, id uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryIncrementPostViews, id)
}
