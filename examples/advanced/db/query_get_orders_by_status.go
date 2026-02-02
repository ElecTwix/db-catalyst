package advanceddb

import "context"

const queryGetOrdersByStatus string = `SELECT 
    id,
    user_id,
    status,
    total_amount,
    created_at
FROM orders
WHERE status = ?
ORDER BY created_at DESC;`

func (q *Queries) GetOrdersByStatus(ctx context.Context, status interface{}) ([]GetOrdersByStatusRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetOrdersByStatus, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetOrdersByStatusRow
	for rows.Next() {
		item, err := scanGetOrdersByStatusRow(rows)
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
