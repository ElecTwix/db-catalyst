package cachedb

import (
	"context"
	"database/sql"
	"fmt"
)

const queryGetUserByEmail string = `SELECT * FROM users WHERE email = $1;`

// GetUserByEmail retrieves a user by email with caching.
func (q *Queries) GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error) {
	cacheKey := "GetUserByEmail:" + fmt.Sprintf("email=%v:", email)
	if q.cache != nil {
		if cached, ok := q.cache.Get(ctx, cacheKey); ok {
			if result, ok := cached.(GetUserByEmailRow); ok {
				return result, nil
			}
		}
	}
	rows, err := q.db.QueryContext(ctx, queryGetUserByEmail, email)
	if err != nil {
		return GetUserByEmailRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetUserByEmailRow{}, err
		}
		return GetUserByEmailRow{}, sql.ErrNoRows
	}
	item, err := scanGetUserByEmailRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	if q.cache != nil {
		q.cache.Set(ctx, cacheKey, item, 300)
	}
	return item, nil
}
