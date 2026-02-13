package cachedb

import (
	"context"
	"database/sql"
)

const queryUpdateUser string = `UPDATE users SET name = $1, email = $2 WHERE id = $3;`

// UpdateUser updates a user and invalidates related caches.
func (q *Queries) UpdateUser(ctx context.Context, name string, email string, id int64) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryUpdateUser, name, email, id)
}
