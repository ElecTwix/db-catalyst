package complexdb

import (
	"context"
	"database/sql"
)

type CreateAuthorParams struct {
	Name  string
	Email string
	Bio   sql.NullString
}

const queryCreateAuthor string = `INSERT INTO authors (name, email, bio)
VALUES (?, ?, ?)
RETURNING *;`

func (q *Queries) CreateAuthor(ctx context.Context, arg CreateAuthorParams) (CreateAuthorRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateAuthor, arg.Name, arg.Email, arg.Bio)
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
