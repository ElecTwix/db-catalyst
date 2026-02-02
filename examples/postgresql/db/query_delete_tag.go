package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryDeleteTag string = `DELETE FROM tags WHERE id = $1;

-- Post-Tag relationship queries`

func (q *Queries) DeleteTag(ctx context.Context, arg1 uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteTag, arg1)
}
