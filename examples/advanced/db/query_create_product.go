package advanceddb

import (
	"context"
	"database/sql"
)

type CreateProductParams struct {
	Sku   any
	Name  any
	Price any
}

const queryCreateProduct string = `INSERT INTO products (sku, name, price)
VALUES (?, ?, ?)
RETURNING *;`

func (q *Queries) CreateProduct(ctx context.Context, arg CreateProductParams) (CreateProductRow, error) {
	rows, err := q.db.QueryContext(ctx, queryCreateProduct, arg.Sku, arg.Name, arg.Price)
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
