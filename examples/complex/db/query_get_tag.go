package complexdb

import (
	"context"
	"database/sql"
)

const queryGetTag string = `SELECT * FROM tags WHERE id = ?;`

func (q *Queries) GetTag(ctx context.Context, id int64) (GetTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetTag, id)
	if err != nil {
		return GetTagRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetTagRow{}, err
		}
		return GetTagRow{}, sql.ErrNoRows
	}
	item, err := scanGetTagRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
