package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgtype"
)

const queryGetTagByName string = `SELECT * FROM tags WHERE tagname = $1;`

func (q *Queries) GetTagByName(ctx context.Context, arg1 pgtype.Text) (GetTagByNameRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetTagByName, arg1)
	if err != nil {
		return GetTagByNameRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetTagByNameRow{}, err
		}
		return GetTagByNameRow{}, sql.ErrNoRows
	}
	item, err := scanGetTagByNameRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
