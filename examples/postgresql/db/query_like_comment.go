package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryLikeComment string = `UPDATE comments SET likes = likes + 1 WHERE id = $1;

-- Tag queries`

func (q *Queries) LikeComment(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, queryLikeComment, id)
	return err
}
