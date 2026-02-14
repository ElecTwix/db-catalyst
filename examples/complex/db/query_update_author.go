package complexdb

import (
	"context"
	"database/sql"
)

type UpdateAuthorParams struct {
	Name  string
	Email string
	Bio   sql.NullString
	Id    int64
}

const queryUpdateAuthor string = `UPDATE authors
SET name = ?, email = ?, bio = ?
WHERE id = ?
RETURNING *;`

func (q *Queries) UpdateAuthor(ctx context.Context, arg UpdateAuthorParams) (UpdateAuthorRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateAuthor, arg.Name, arg.Email, arg.Bio, arg.Id)
	if err != nil {
		return UpdateAuthorRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdateAuthorRow{}, err
		}
		return UpdateAuthorRow{}, sql.ErrNoRows
	}
	item, err := scanUpdateAuthorRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
