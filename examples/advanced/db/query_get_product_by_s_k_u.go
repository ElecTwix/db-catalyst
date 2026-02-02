package advanceddb

import (
	"context"
	"database/sql"
)

const queryGetProductBySku string = `SELECT * FROM products WHERE sku = ?;`

func (q *Queries) GetProductBySku(ctx context.Context, sku interface{}) (GetProductBySkuRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetProductBySku, sku)
	if err != nil {
		return GetProductBySkuRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetProductBySkuRow{}, err
		}
		return GetProductBySkuRow{}, sql.ErrNoRows
	}
	item, err := scanGetProductBySkuRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
