package complex

import (
	"context"
	"database/sql"
)

type Querier interface {
	DeleteTag(ctx context.Context, tag interface{}) (sql.Result, error)
	GetItemWithTags(ctx context.Context, id *int32) (GetItemWithTagsRow, error)
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
