package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryGetPostById string = `SELECT * FROM posts WHERE id = $1;`

func (q *Queries) GetPostById(ctx context.Context, id uuid.UUID) (GetPostByIdRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetPostById, id)
	if err != nil {
		return GetPostByIdRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetPostByIdRow{}, err
		}
		return GetPostByIdRow{}, sql.ErrNoRows
	}
	item, err := scanGetPostByIdRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
