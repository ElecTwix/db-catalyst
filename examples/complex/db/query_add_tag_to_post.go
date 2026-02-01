package complexdb

import (
	"context"
	"database/sql"
)

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;`

func (q *Queries) AddTagToPost(ctx context.Context, arg1 int32, arg2 int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryAddTagToPost, arg1, arg2)
}
