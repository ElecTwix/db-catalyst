package advanceddb

import (
	"context"
	"database/sql"
)

type Querier interface {
	CreateOrder(ctx context.Context, userId any, status any, totalAmount any) (CreateOrderRow, error)
	CreateProduct(ctx context.Context, sku any, name any, price any) (CreateProductRow, error)
	CreateUser(ctx context.Context, email any) (CreateUserRow, error)
	GetOrder(ctx context.Context, id any) (GetOrderRow, error)
	GetOrderStatistics(ctx context.Context, userId any) (GetOrderStatisticsRow, error)
	GetOrdersByStatus(ctx context.Context, status any) ([]GetOrdersByStatusRow, error)
	GetProduct(ctx context.Context, id any) (GetProductRow, error)
	GetProductBySku(ctx context.Context, sku any) (GetProductBySkuRow, error)
	GetUser(ctx context.Context, id any) (GetUserRow, error)
	GetUserByEmail(ctx context.Context, email any) (GetUserByEmailRow, error)
	ListOrdersByUser(ctx context.Context, userId any) ([]ListOrdersByUserRow, error)
	ListProducts(ctx context.Context) ([]ListProductsRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	UpdateOrderStatus(ctx context.Context, status any, id any) (UpdateOrderStatusRow, error)
	UpdateProductPrice(ctx context.Context, price any, id any) (UpdateProductPriceRow, error)
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
