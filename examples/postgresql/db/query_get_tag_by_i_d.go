package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const queryGetTagById string = `SELECT * FROM tags WHERE id = $1;`

func (q *Queries) GetTagById(ctx context.Context, id uuid.UUID) (GetTagByIdRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetTagById, id)
	if err != nil {
		return GetTagByIdRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetTagByIdRow{}, err
		}
		return GetTagByIdRow{}, sql.ErrNoRows
	}
	item, err := scanGetTagByIdRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
