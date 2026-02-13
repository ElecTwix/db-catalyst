package cachedb

import (
	"context"
	"database/sql"
)

const queryCreateUser string = `INSERT INTO users (name, email, active) VALUES ($1, $2, true);`

// CreateUser inserts a new user (no caching, but invalidates users list).
// This is a write operation so we don't cache it, but we invalidate
// the cached list of users.
func (q *Queries) CreateUser(ctx context.Context, name string, name2 string) (sql.Result, error) {
	return q.db.ExecContext(ctx, queryCreateUser, name, name2)
}
