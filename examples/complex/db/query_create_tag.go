package complexdb

import (
	"context"
	"database/sql"
)

const queryCreateTag string = `INSERT INTO tags (name, description)
VALUES (?, ?)
RETURNING *;`

func (q *Queries) CreateTag(ctx context.Context, arg1 string, arg2 sql.NullString) (CreateTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateTag, arg1, arg2)
	if err != nil {
		return CreateTagRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateTagRow{}, err
		}
		return CreateTagRow{}, sql.ErrNoRows
	}
	item, err := scanCreateTagRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
