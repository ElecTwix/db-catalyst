package advanceddb

import (
	"context"
	"database/sql"
)

type UpdateProductPriceParams struct {
	Price any
	Id    any
}

const queryUpdateProductPrice string = `UPDATE products
SET price = ?
WHERE id = ?
RETURNING *;`

func (q *Queries) UpdateProductPrice(ctx context.Context, arg UpdateProductPriceParams) (UpdateProductPriceRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUpdateProductPrice, arg.Price, arg.Id)
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
