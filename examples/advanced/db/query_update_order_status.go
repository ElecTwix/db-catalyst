package advanceddb

import (
	"context"
	"database/sql"
)

const queryUpdateOrderStatus string = `UPDATE orders
SET status = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING id, user_id, status, total_amount, created_at, updated_at;`

func (q *Queries) UpdateOrderStatus(ctx context.Context, status any, arg2 any) (UpdateOrderStatusRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateOrderStatus, status, arg2)
	if err != nil {
		return UpdateOrderStatusRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdateOrderStatusRow{}, err
		}
		return UpdateOrderStatusRow{}, sql.ErrNoRows
	}
	item, err := scanUpdateOrderStatusRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
