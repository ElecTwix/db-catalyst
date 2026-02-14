package postgresqldb

import "context"

const querySearchPostsByCategory string = `SELECT * FROM posts WHERE $1 = ANY(categories) AND is_published = true;

-- Comment queries`

func (q *Queries) SearchPostsByCategory(ctx context.Context, any any) ([]SearchPostsByCategoryRow, error) {
	rows, err := q.db.QueryContext(ctx, querySearchPostsByCategory, any)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SearchPostsByCategoryRow
	for rows.Next() {
		item, err := scanSearchPostsByCategoryRow(rows)
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
