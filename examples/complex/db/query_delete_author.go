package complexdb

import "context"

const queryDeleteAuthor string = `DELETE FROM authors WHERE id = ?;`

func (q *Queries) DeleteAuthor(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, queryDeleteAuthor, id)
	return err
}
