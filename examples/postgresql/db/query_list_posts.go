package postgresqldb

import "context"

const queryListPosts string = `SELECT * FROM posts ORDER BY created_at DESC;`

func (q *Queries) ListPosts(ctx context.Context) ([]ListPostsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListPosts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListPostsRow
	for rows.Next() {
		item, err := scanListPostsRow(rows)
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
