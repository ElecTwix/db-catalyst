package complexdb

import "context"

type AddTagToPostParams struct {
	PostId int64
	TagId  int64
}

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;`

func (q *Queries) AddTagToPost(ctx context.Context, arg AddTagToPostParams) error {
	_, err := q.db.ExecContext(ctx, queryAddTagToPost, arg.PostId, arg.TagId)
	return err
}
