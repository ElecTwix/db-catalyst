package basic

import (
	"context"
	"database/sql"
)

type Querier interface {
	CreateUser(ctx context.Context, username string, email string) (CreateUserRow, error)
	GetUser(ctx context.Context, id int32) (GetUserRow, error)
	ListPostsByAuthor(ctx context.Context, authorId int32) ([]ListPostsByAuthorRow, error)
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
