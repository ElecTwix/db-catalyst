package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryDeleteUser string = `DELETE FROM users WHERE id = $1;`

func (q *Queries) DeleteUser(ctx context.Context, id uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteUser, id)
}
