package cachedb

import (
	"context"
	"fmt"
)

const queryGetUserPosts string = `SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC;`

// GetUserPosts retrieves posts for a specific user.
func (q *Queries) GetUserPosts(ctx context.Context, userId int64) ([]GetUserPostsRow, error) {
	cacheKey := "GetUserPosts:" + fmt.Sprintf("userId=%v:", userId)
	if q.cache != nil {
		if cached, ok := q.cache.Get(ctx, cacheKey); ok {
			if result, ok := cached.([]GetUserPostsRow); ok {
				return result, nil
			}
		}
	}
	rows, err := q.db.QueryContext(ctx, queryGetUserPosts, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetUserPostsRow
	for rows.Next() {
		item, err := scanGetUserPostsRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if q.cache != nil {
		q.cache.Set(ctx, cacheKey, items, 600)
	}
	return items, nil
}
