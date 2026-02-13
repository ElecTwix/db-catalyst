package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2);`

func (q *Queries) AddTagToPost(ctx context.Context, postId *uuid.UUID, postId2 *uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryAddTagToPost, postId, postId2)
}
