package advanceddb

import "context"

const queryListOrdersByUser string = `SELECT 
    id,
    status,
    total_amount,
    created_at
FROM orders
WHERE user_id = ?
ORDER BY created_at DESC;`

func (q *Queries) ListOrdersByUser(ctx context.Context, userId any) ([]ListOrdersByUserRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListOrdersByUser, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListOrdersByUserRow
	for rows.Next() {
		item, err := scanListOrdersByUserRow(rows)
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
