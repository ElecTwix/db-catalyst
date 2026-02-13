package complexdb

import (
	"context"
	"database/sql"
)

const queryGetAuthorWithPostCount string = `SELECT 
    a.id,
    a.name,
    a.email,
    a.bio,
    (SELECT COUNT(*) FROM posts WHERE posts.author_id = a.id) as total_posts
FROM authors a
WHERE a.id = ?;`

func (q *Queries) GetAuthorWithPostCount(ctx context.Context, id int64) (GetAuthorWithPostCountRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetAuthorWithPostCount, id)
	if err != nil {
		return GetAuthorWithPostCountRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetAuthorWithPostCountRow{}, err
		}
		return GetAuthorWithPostCountRow{}, sql.ErrNoRows
	}
	item, err := scanGetAuthorWithPostCountRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
