package advanceddb

import (
	"context"
	"database/sql"
)

const queryGetOrder string = `SELECT 
    id,
    user_id,
    status,
    total_amount,
    created_at,
    updated_at
FROM orders
WHERE id = ?;`

func (q *Queries) GetOrder(ctx context.Context, id any) (GetOrderRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetOrder, id)
	if err != nil {
		return GetOrderRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetOrderRow{}, err
		}
		return GetOrderRow{}, sql.ErrNoRows
	}
	item, err := scanGetOrderRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
