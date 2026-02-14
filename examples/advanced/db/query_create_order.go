package advanceddb

import (
	"context"
	"database/sql"
)

type CreateOrderParams struct {
	UserId      any
	Status      any
	TotalAmount any
}

const queryCreateOrder string = `INSERT INTO orders (user_id, status, total_amount)
VALUES (?, ?, ?)
RETURNING id, user_id, status, total_amount, created_at, updated_at;`

func (q *Queries) CreateOrder(ctx context.Context, arg CreateOrderParams) (CreateOrderRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateOrder, arg.UserId, arg.Status, arg.TotalAmount)
	if err != nil {
		return CreateOrderRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateOrderRow{}, err
		}
		return CreateOrderRow{}, sql.ErrNoRows
	}
	item, err := scanCreateOrderRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
