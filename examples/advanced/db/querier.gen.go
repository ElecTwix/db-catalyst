package advanceddb

import (
	"context"
	"database/sql"

	"github.com/electwix/db-catalyst/examples/advanced/types"
)

type Querier interface {
	CreateOrder(ctx context.Context, arg1 int32, arg2 int32, arg3 int32) (CreateOrderRow, error)
	CreateProduct(ctx context.Context, arg1 int32, arg2 types.SKU, arg3 int32) (CreateProductRow, error)
	CreateUser(ctx context.Context, arg1 int32) (CreateUserRow, error)
	GetOrder(ctx context.Context, id *int32) (GetOrderRow, error)
	GetOrderStatistics(ctx context.Context, userId *int32) (GetOrderStatisticsRow, error)
	GetOrdersByStatus(ctx context.Context, status *int32) ([]GetOrdersByStatusRow, error)
	GetProduct(ctx context.Context, id *int32) (GetProductRow, error)
	GetProductBySku(ctx context.Context, sku *int32) (GetProductBySkuRow, error)
	GetUser(ctx context.Context, id *int32) (GetUserRow, error)
	GetUserByEmail(ctx context.Context, email *int32) (GetUserByEmailRow, error)
	ListOrdersByUser(ctx context.Context, userId *int32) ([]ListOrdersByUserRow, error)
	ListProducts(ctx context.Context) ([]ListProductsRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	UpdateOrderStatus(ctx context.Context, status *int32, arg2 *int32) (UpdateOrderStatusRow, error)
	UpdateProductPrice(ctx context.Context, price *int32, arg2 *int32) (UpdateProductPriceRow, error)
}
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) sql.Row
}
type Queries struct {
	db DBTX
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type QueryResult struct {
	LastInsertID int64
	RowsAffected int64
}
