package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryDeletePost string = `DELETE FROM posts WHERE id = $1;`

func (q *Queries) DeletePost(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, queryDeletePost, id)
	return err
}
