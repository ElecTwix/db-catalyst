package complexdb

import (
	"context"
	"database/sql"
)

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;`

func (q *Queries) AddTagToPost(ctx context.Context, postId int64, tagId int64) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryAddTagToPost, postId, tagId)
}
