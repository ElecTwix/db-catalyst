package cachedb

import "context"

type UpdateUserParams struct {
	Name  string
	Email string
	Id    int64
}

const queryUpdateUser string = `UPDATE users SET name = $1, email = $2 WHERE id = $3;`

// UpdateUser updates a user and invalidates related caches.
func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) error {
	_, err := q.db.ExecContext(ctx, queryUpdateUser, arg.Name, arg.Email, arg.Id)
	return err
}
