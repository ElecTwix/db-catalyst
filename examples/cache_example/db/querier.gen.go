package cachedb

import (
	"context"
	"database/sql"
	"time"
)

type Querier interface {
	CreateUser(ctx context.Context, name string, name2 string) (sql.Result, error)
	GetPopularPosts(ctx context.Context, limit *any) ([]GetPopularPostsRow, error)
	GetUser(ctx context.Context, id int64) (GetUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error)
	GetUserPosts(ctx context.Context, userId int64) ([]GetUserPostsRow, error)
	ListActiveUsers(ctx context.Context) ([]ListActiveUsersRow, error)
	UpdateUser(ctx context.Context, name string, email string, id int64) (sql.Result, error)
}
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) sql.Row
}
type Cache interface {
	Get(ctx context.Context, key string) (any, bool)
	Set(ctx context.Context, key string, value any, ttl time.Duration)
	Delete(ctx context.Context, key string)
	Invalidate(ctx context.Context, pattern string)
}
type Queries struct {
	db    DBTX
	cache Cache
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type QueryResult struct {
	LastInsertID int64
	RowsAffected int64
}
