package postgresqldb

import "context"

const queryGetPopularPosts string = `SELECT * FROM posts 
WHERE is_published = true 
ORDER BY view_count DESC, published_at DESC
LIMIT $1;`

func (q *Queries) GetPopularPosts(ctx context.Context, limit *any) ([]GetPopularPostsRow, error) {
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
	return items, nil
}
