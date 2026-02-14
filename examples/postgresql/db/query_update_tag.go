package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UpdateTagParams struct {
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	Id             uuid.UUID
}

const queryUpdateTag string = `UPDATE tags SET tagname = $2, tagdescription = $3 WHERE id = $1 RETURNING *;`

func (q *Queries) UpdateTag(ctx context.Context, arg UpdateTagParams) (UpdateTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateTag, arg.Tagname, arg.Tagdescription, arg.Id)
	if err != nil {
		return UpdateTagRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdateTagRow{}, err
		}
		return UpdateTagRow{}, sql.ErrNoRows
	}
	item, err := scanUpdateTagRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
