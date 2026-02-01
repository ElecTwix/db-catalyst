package complexdb

import "context"

const queryListAuthors string = `SELECT * FROM authors ORDER BY name;`

func (q *Queries) ListAuthors(ctx context.Context) ([]ListAuthorsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListAuthors)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListAuthorsRow
	for rows.Next() {
		item, err := scanListAuthorsRow(rows)
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
