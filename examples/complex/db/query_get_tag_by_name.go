package complexdb

import (
	"context"
	"database/sql"
)

const queryGetTagByName string = `SELECT * FROM tags WHERE name = ?;`

func (q *Queries) GetTagByName(ctx context.Context, name string) (GetTagByNameRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetTagByName, name)
	if err != nil {
		return GetTagByNameRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetTagByNameRow{}, err
		}
		return GetTagByNameRow{}, sql.ErrNoRows
	}
	item, err := scanGetTagByNameRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
