package mysqlblog

import (
	"context"
	"database/sql"
)

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?);`

func (q *Queries) AddTagToPost(ctx context.Context, postId int32, tagId int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryAddTagToPost, postId, tagId)
}
