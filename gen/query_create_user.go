package test

import (
	"context"
	"database/sql"
)

const queryCreateUser string = `INSERT INTO users (id, name, trigger_type) VALUES (?, ?, ?);`

func (q *Queries) CreateUser(ctx context.Context, arg1 interface{}, arg2 interface{}, arg3 *interface{}) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryCreateUser, arg1, arg2, arg3)
}
