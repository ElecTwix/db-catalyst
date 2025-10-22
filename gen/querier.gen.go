package test

import (
	"context"
	"database/sql"
)

type Querier interface {
	CreateUser(ctx context.Context, arg1 interface{}, arg2 interface{}, arg3 *interface{}) (sql.Result, error)
	GetUser(ctx context.Context, arg1 interface{}) (GetUserRow, error)
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
