package complexdb

import "context"

const queryDeleteTag string = `DELETE FROM tags WHERE id = ?;`

func (q *Queries) DeleteTag(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, queryDeleteTag, id)
	return err
}
