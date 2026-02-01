package advanceddb

import (
	"context"
	"database/sql"
)

const queryUpdateProductPrice string = `UPDATE products
SET price = ?
WHERE id = ?
RETURNING *;`

func (q *Queries) UpdateProductPrice(ctx context.Context, price *int32, arg2 *int32) (UpdateProductPriceRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateProductPrice, price, arg2)
	if err != nil {
		return UpdateProductPriceRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return UpdateProductPriceRow{}, err
		}
		return UpdateProductPriceRow{}, sql.ErrNoRows
	}
	item, err := scanUpdateProductPriceRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
