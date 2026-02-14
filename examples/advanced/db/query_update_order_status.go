package advanceddb

import (
	"context"
	"database/sql"
)

type UpdateOrderStatusParams struct {
	Status any
	Id     any
}

const queryUpdateOrderStatus string = `UPDATE orders
SET status = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING id, user_id, status, total_amount, created_at, updated_at;`

func (q *Queries) UpdateOrderStatus(ctx context.Context, arg UpdateOrderStatusParams) (UpdateOrderStatusRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateOrderStatus, arg.Status, arg.Id)
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
