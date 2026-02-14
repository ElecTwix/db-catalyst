package mysqlblog

import "context"

type AddTagToPostParams struct {
	PostId int32
	TagId  int32
}

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?);`

func (q *Queries) AddTagToPost(ctx context.Context, arg AddTagToPostParams) error {
	_, err := q.db.ExecContext(ctx, queryAddTagToPost, arg.PostId, arg.TagId)
	return err
}
