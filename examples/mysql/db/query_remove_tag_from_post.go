package mysqlblog

import (
	"context"
	"database/sql"
)

const queryRemoveTagFromPost string = `DELETE FROM post_tags WHERE post_id = ? AND tag_id = ?;`

func (q *Queries) RemoveTagFromPost(ctx context.Context, postId int32, tagId int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryRemoveTagFromPost, postId, tagId)
}
