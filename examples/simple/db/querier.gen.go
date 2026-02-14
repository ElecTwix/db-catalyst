package simpledb

import (
	"context"
	"database/sql"
)

type Querier interface {
	CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error)
	DeleteUser(ctx context.Context, id int64) error
	GetUser(ctx context.Context, id int64) (GetUserRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	UpdateUser(ctx context.Context, arg UpdateUserParams) (UpdateUserRow, error)
}
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
type Queries struct {
	db DBTX
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}
func (q *Queries) WithTx(tx DBTX) *Queries {
	return &Queries{db: tx}
}

type QueryResult struct {
	LastInsertID int64
	RowsAffected int64
}
