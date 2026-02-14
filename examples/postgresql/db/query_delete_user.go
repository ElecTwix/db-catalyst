package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryDeleteUser string = `DELETE FROM users WHERE id = $1;`

func (q *Queries) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, queryDeleteUser, id)
	return err
}
