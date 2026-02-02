package advanceddb

import (
	"context"
	"database/sql"
)

type Querier interface {
	CreateOrder(ctx context.Context, arg1 interface{}, arg2 interface{}, arg3 interface{}) (CreateOrderRow, error)
	CreateProduct(ctx context.Context, arg1 interface{}, arg2 string, arg3 interface{}) (CreateProductRow, error)
	CreateUser(ctx context.Context, arg1 interface{}) (CreateUserRow, error)
	GetOrder(ctx context.Context, id interface{}) (GetOrderRow, error)
	GetOrderStatistics(ctx context.Context, userId interface{}) (GetOrderStatisticsRow, error)
	GetOrdersByStatus(ctx context.Context, status interface{}) ([]GetOrdersByStatusRow, error)
	GetProduct(ctx context.Context, id interface{}) (GetProductRow, error)
	GetProductBySku(ctx context.Context, sku interface{}) (GetProductBySkuRow, error)
	GetUser(ctx context.Context, id interface{}) (GetUserRow, error)
	GetUserByEmail(ctx context.Context, email interface{}) (GetUserByEmailRow, error)
	ListOrdersByUser(ctx context.Context, userId interface{}) ([]ListOrdersByUserRow, error)
	ListProducts(ctx context.Context) ([]ListProductsRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	UpdateOrderStatus(ctx context.Context, status interface{}, arg2 interface{}) (UpdateOrderStatusRow, error)
	UpdateProductPrice(ctx context.Context, price interface{}, arg2 interface{}) (UpdateProductPriceRow, error)
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
