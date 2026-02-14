package complexdb

import "context"

type SearchPostsParams struct {
	Title   any
	Content any
	Limit   any
	Offset  any
}

const querySearchPosts string = `SELECT *
FROM posts
WHERE published = 1
  AND (title LIKE '%' || ? || '%' OR content LIKE '%' || ? || '%')
ORDER BY view_count DESC
LIMIT ? OFFSET ?;`

func (q *Queries) SearchPosts(ctx context.Context, arg SearchPostsParams) ([]SearchPostsRow, error) {
	rows, err := q.db.QueryContext(ctx, querySearchPosts, arg.Title, arg.Content, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SearchPostsRow
	for rows.Next() {
		item, err := scanSearchPostsRow(rows)
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
