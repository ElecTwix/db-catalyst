package advanceddb

import (
	"context"
	"database/sql"
)

const queryCreateProduct string = `INSERT INTO products (sku, name, price)
VALUES (?, ?, ?)
RETURNING *;`

func (q *Queries) CreateProduct(ctx context.Context, sku any, name any, price any) (CreateProductRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateProduct, sku, name, price)
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
