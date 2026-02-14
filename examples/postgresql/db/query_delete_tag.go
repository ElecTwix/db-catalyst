package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryDeleteTag string = `DELETE FROM tags WHERE id = $1;

-- Post-Tag relationship queries`

func (q *Queries) DeleteTag(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, queryDeleteTag, id)
	return err
}
