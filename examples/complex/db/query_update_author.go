package complexdb

import (
	"context"
	"database/sql"
)

const queryUpdateAuthor string = `UPDATE authors
SET name = ?, email = ?, bio = ?
WHERE id = ?
RETURNING *;`

func (q *Queries) UpdateAuthor(ctx context.Context, name string, email string, bio sql.NullString, id int64) (UpdateAuthorRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateAuthor, name, email, bio, id)
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
