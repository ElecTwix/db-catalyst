package cachedb

import "context"

type CreateUserParams struct {
	Name  string
	Name2 string
}

const queryCreateUser string = `INSERT INTO users (name, email, active) VALUES ($1, $2, true);`

// CreateUser inserts a new user (no caching, but invalidates users list).
// This is a write operation so we don't cache it, but we invalidate
// the cached list of users.
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) error {
	_, err := q.db.ExecContext(ctx, queryCreateUser, arg.Name, arg.Name2)
	return err
}
