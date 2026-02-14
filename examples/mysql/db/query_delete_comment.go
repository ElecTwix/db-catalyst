package mysqlblog

import "context"

const queryDeleteComment string = `DELETE FROM comments WHERE id = ?;`

func (q *Queries) DeleteComment(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, queryDeleteComment, id)
	return err
}
