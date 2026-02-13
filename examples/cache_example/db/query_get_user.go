package cachedb

import (
	"context"
	"database/sql"
	"fmt"
)

const queryGetUser string = `SELECT * FROM users WHERE id = $1;`

// GetUser retrieves a user by ID with 5-minute cache.
// The cache key includes the user ID.
func (q *Queries) GetUser(ctx context.Context, id int64) (GetUserRow, error) {
	cacheKey := "GetUser:" + fmt.Sprintf("id=%v:", id)
	if q.cache != nil {
		if cached, ok := q.cache.Get(ctx, cacheKey); ok {
			if result, ok := cached.(GetUserRow); ok {
				return result, nil
			}
		}
	}
	rows, err := q.db.QueryContext(ctx, queryGetUser, id)
	if err != nil {
		return GetUserRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetUserRow{}, err
		}
		return GetUserRow{}, sql.ErrNoRows
	}
	item, err := scanGetUserRow(rows)
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
