package complexdb

import "context"

const queryIncrementViewCount string = `UPDATE posts
SET view_count = view_count + 1
WHERE id = ?;`

func (q *Queries) IncrementViewCount(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, queryIncrementViewCount, id)
	return err
}
