package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryGetCommentById string = `SELECT * FROM comments WHERE id = $1;`

func (q *Queries) GetCommentById(ctx context.Context, arg1 uuid.UUID) (GetCommentByIdRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetCommentById, arg1)
	if err != nil {
		return GetCommentByIdRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetCommentByIdRow{}, err
		}
		return GetCommentByIdRow{}, sql.ErrNoRows
	}
	item, err := scanGetCommentByIdRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
