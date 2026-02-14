package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgtype"
)

type CreateTagParams struct {
	Tagname  pgtype.Text
	Tagname2 pgtype.Text
}

const queryCreateTag string = `INSERT INTO tags (tagname, tagdescription) VALUES ($1, $2) RETURNING *;`

func (q *Queries) CreateTag(ctx context.Context, arg CreateTagParams) (CreateTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateTag, arg.Tagname, arg.Tagname2)
	if err != nil {
		return CreateTagRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateTagRow{}, err
		}
		return CreateTagRow{}, sql.ErrNoRows
	}
	item, err := scanCreateTagRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
