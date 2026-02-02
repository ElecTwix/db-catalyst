package postgresqldb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const querySearchUsersByMetadata string = `SELECT * FROM users WHERE metadata @> $1;

-- Post queries`

func (q *Queries) SearchUsersByMetadata(ctx context.Context, arg1 pgtype.Int4) ([]SearchUsersByMetadataRow, error) {
	rows, err := q.db.QueryContext(ctx, querySearchUsersByMetadata, arg1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SearchUsersByMetadataRow
	for rows.Next() {
		item, err := scanSearchUsersByMetadataRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
