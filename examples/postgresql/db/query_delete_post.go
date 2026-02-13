package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryDeletePost string = `DELETE FROM posts WHERE id = $1;`

func (q *Queries) DeletePost(ctx context.Context, id uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeletePost, id)
}
