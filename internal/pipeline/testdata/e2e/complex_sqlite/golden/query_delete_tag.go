package complex

import "context"

const queryDeleteTag string = `DELETE FROM tags WHERE tag = :tag;`

func (q *Queries) DeleteTag(ctx context.Context, tag string) error {
	_, err := q.db.ExecContext(ctx, queryDeleteTag, tag)
	return err
}
