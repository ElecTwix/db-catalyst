package mysqlblog

import "context"

type RemoveTagFromPostParams struct {
	PostId int32
	TagId  int32
}

const queryRemoveTagFromPost string = `DELETE FROM post_tags WHERE post_id = ? AND tag_id = ?;`

func (q *Queries) RemoveTagFromPost(ctx context.Context, arg RemoveTagFromPostParams) error {
	_, err := q.db.ExecContext(ctx, queryRemoveTagFromPost, arg.PostId, arg.TagId)
	return err
}
