package complexdb

import (
	"context"
	"database/sql"
)

const queryGetAuthor string = `SELECT * FROM authors WHERE id = ?;`

func (q *Queries) GetAuthor(ctx context.Context, id int32) (GetAuthorRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetAuthor, id)
	if err != nil {
		return GetAuthorRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetAuthorRow{}, err
		}
		return GetAuthorRow{}, sql.ErrNoRows
	}
	item, err := scanGetAuthorRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
