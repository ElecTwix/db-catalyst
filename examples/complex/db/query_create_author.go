package complexdb

import (
	"context"
	"database/sql"
)

const queryCreateAuthor string = `INSERT INTO authors (name, email, bio)
VALUES (?, ?, ?)
RETURNING *;`

func (q *Queries) CreateAuthor(ctx context.Context, arg1 interface{}, arg2 interface{}, arg3 *interface{}) (CreateAuthorRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateAuthor, arg1, arg2, arg3)
	if err != nil {
		return CreateAuthorRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateAuthorRow{}, err
		}
		return CreateAuthorRow{}, sql.ErrNoRows
	}
	item, err := scanCreateAuthorRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
