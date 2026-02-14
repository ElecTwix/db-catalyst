package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryDeleteComment string = `DELETE FROM comments WHERE id = $1;`

func (q *Queries) DeleteComment(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, queryDeleteComment, id)
	return err
}
