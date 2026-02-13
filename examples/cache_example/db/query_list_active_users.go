package cachedb

import "context"

const queryListActiveUsers string = `SELECT * FROM users WHERE active = true ORDER BY created_at DESC;`

// ListActiveUsers retrieves all active users with 1-hour cache.
// No custom key pattern, so cache key is auto-generated.
func (q *Queries) ListActiveUsers(ctx context.Context) ([]ListActiveUsersRow, error) {
	cacheKey := "ListActiveUsers:"
	if q.cache != nil {
		if cached, ok := q.cache.Get(ctx, cacheKey); ok {
			if result, ok := cached.([]ListActiveUsersRow); ok {
				return result, nil
			}
		}
	}
	rows, err := q.db.QueryContext(ctx, queryListActiveUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListActiveUsersRow
	for rows.Next() {
		item, err := scanListActiveUsersRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if q.cache != nil {
		q.cache.Set(ctx, cacheKey, items, 3600)
	}
	return items, nil
}
