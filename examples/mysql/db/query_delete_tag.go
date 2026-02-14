package mysqlblog

import "context"

const queryDeleteTag string = `DELETE FROM tags WHERE id = ?;`

func (q *Queries) DeleteTag(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, queryDeleteTag, id)
	return err
}
