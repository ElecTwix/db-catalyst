package simpledb

import (
	"context"
	"database/sql"
)

const queryDeleteUser string = `DELETE FROM users WHERE id = ?;`

func (q *Queries) DeleteUser(ctx context.Context, id int32) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryDeleteUser, id)
}
