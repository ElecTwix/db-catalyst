package advanceddb

import (
	"context"
	"database/sql"

	"github.com/electwix/db-catalyst/examples/advanced/types"
)

const queryCreateProduct string = `INSERT INTO products (sku, name, price)
VALUES (?, ?, ?)
RETURNING *;`

func (q *Queries) CreateProduct(ctx context.Context, arg1 int32, arg2 types.SKU, arg3 int32) (CreateProductRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateProduct, arg1, arg2, arg3)
	if err != nil {
		return CreateProductRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateProductRow{}, err
		}
		return CreateProductRow{}, sql.ErrNoRows
	}
	item, err := scanCreateProductRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
