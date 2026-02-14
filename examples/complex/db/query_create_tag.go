package complexdb

import (
	"context"
	"database/sql"
)

type CreateTagParams struct {
	Name        string
	Description sql.NullString
}

const queryCreateTag string = `INSERT INTO tags (name, description)
VALUES (?, ?)
RETURNING *;`

func (q *Queries) CreateTag(ctx context.Context, arg CreateTagParams) (CreateTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateTag, arg.Name, arg.Description)
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
