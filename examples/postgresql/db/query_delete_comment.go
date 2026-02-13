package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryDeleteComment string = `DELETE FROM comments WHERE id = $1;`

func (q *Queries) DeleteComment(ctx context.Context, id uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteComment, id)
}
