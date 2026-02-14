package mysqlblog

import "context"

const queryDeletePost string = `DELETE FROM posts WHERE id = ?;`

func (q *Queries) DeletePost(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, queryDeletePost, id)
	return err
}
