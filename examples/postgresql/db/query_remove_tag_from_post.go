package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

type RemoveTagFromPostParams struct {
	PostId *uuid.UUID
	TagId  *uuid.UUID
}

const queryRemoveTagFromPost string = `DELETE FROM post_tags WHERE post_id = $1 AND tag_id = $2;`

func (q *Queries) RemoveTagFromPost(ctx context.Context, arg RemoveTagFromPostParams) error {
	_, err := q.db.ExecContext(ctx, queryRemoveTagFromPost, arg.PostId, arg.TagId)
	return err
}
