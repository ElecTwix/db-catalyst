package simpledb

import "context"

const queryDeleteUser string = `DELETE FROM users WHERE id = ?;`

func (q *Queries) DeleteUser(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, queryDeleteUser, id)
	return err
}
