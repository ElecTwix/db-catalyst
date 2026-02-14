package cachedb

import (
	"context"
	"fmt"
)

const queryGetPopularPosts string = `SELECT * FROM posts WHERE likes > 100 ORDER BY created_at DESC LIMIT $1;`

// GetPopularPosts retrieves popular posts with longer cache TTL.
func (q *Queries) GetPopularPosts(ctx context.Context, limit any) ([]GetPopularPostsRow, error) {
	cacheKey := "GetPopularPosts:" + fmt.Sprintf("limit=%v:", limit)
	if q.cache != nil {
		if cached, ok := q.cache.Get(ctx, cacheKey); ok {
			if result, ok := cached.([]GetPopularPostsRow); ok {
				return result, nil
			}
		}
	}
	rows, err := q.db.QueryContext(ctx, queryGetPopularPosts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPopularPostsRow
	for rows.Next() {
		item, err := scanGetPopularPostsRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if q.cache != nil {
		q.cache.Set(ctx, cacheKey, items, 1800)
	}
	return items, nil
}
