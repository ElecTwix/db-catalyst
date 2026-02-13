package postgresqldb

import "context"

const querySearchUsersByTag string = `SELECT * FROM users WHERE $1 = ANY(tags);`

func (q *Queries) SearchUsersByTag(ctx context.Context, any *any) ([]SearchUsersByTagRow, error) {
	rows, err := q.db.QueryContext(ctx, querySearchUsersByTag, any)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SearchUsersByTagRow
	for rows.Next() {
		item, err := scanSearchUsersByTagRow(rows)
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
