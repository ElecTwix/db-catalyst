package mysqlblog

import "context"

const queryIncrementPostViews string = `UPDATE posts SET view_count = view_count + 1 WHERE id = ?;`

func (q *Queries) IncrementPostViews(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, queryIncrementPostViews, id)
	return err
}
