package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryRemoveTagFromPost string = `DELETE FROM post_tags WHERE post_id = $1 AND tag_id = $2;`

func (q *Queries) RemoveTagFromPost(ctx context.Context, postId *uuid.UUID, tagId *uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryRemoveTagFromPost, postId, tagId)
}
