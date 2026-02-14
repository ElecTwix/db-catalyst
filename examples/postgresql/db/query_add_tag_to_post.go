package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

type AddTagToPostParams struct {
	PostId  *uuid.UUID
	PostId2 *uuid.UUID
}

const queryAddTagToPost string = `INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2);`

func (q *Queries) AddTagToPost(ctx context.Context, arg AddTagToPostParams) error {
	_, err := q.db.ExecContext(ctx, queryAddTagToPost, arg.PostId, arg.PostId2)
	return err
}
