package complexdb

import (
	"context"
	"database/sql"
)

const queryGetAuthorStats string = `SELECT 
    a.id,
    a.name,
    (SELECT COUNT(*) FROM posts p WHERE p.author_id = a.id) as total_posts,
    (SELECT COALESCE(SUM(p.view_count), 0) FROM posts p WHERE p.author_id = a.id) as total_views
FROM authors a
WHERE a.id = ?;`

func (q *Queries) GetAuthorStats(ctx context.Context, id int32) (GetAuthorStatsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetAuthorStats, id)
	if err != nil {
		return GetAuthorStatsRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetAuthorStatsRow{}, err
		}
		return GetAuthorStatsRow{}, sql.ErrNoRows
	}
	item, err := scanGetAuthorStatsRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
