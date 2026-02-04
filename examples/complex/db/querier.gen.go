package complexdb

import (
	"context"
	"database/sql"
)

type Querier interface {
	AddTagToPost(ctx context.Context, arg1 int32, arg2 int32) (sql.Result, error)
	CreateAuthor(ctx context.Context, arg1 string, arg2 string, arg3 sql.NullString) (CreateAuthorRow, error)
	CreatePost(ctx context.Context, arg1 int32, arg2 string, arg3 string, arg4 int32) (CreatePostRow, error)
	CreateTag(ctx context.Context, arg1 string, arg2 sql.NullString) (CreateTagRow, error)
	DeleteAuthor(ctx context.Context, id int32) (sql.Result, error)
	DeleteTag(ctx context.Context, id int32) (sql.Result, error)
	GetAuthor(ctx context.Context, id int32) (GetAuthorRow, error)
	GetAuthorStats(ctx context.Context, id int32) (GetAuthorStatsRow, error)
	GetAuthorWithPostCount(ctx context.Context, id int32) (GetAuthorWithPostCountRow, error)
	GetPopularTags(ctx context.Context, arg1 *int32) ([]GetPopularTagsRow, error)
	GetPost(ctx context.Context, id int32) (GetPostRow, error)
	GetPostTags(ctx context.Context, postId int32) ([]GetPostTagsRow, error)
	GetPostsByTag(ctx context.Context, name string) ([]GetPostsByTagRow, error)
	GetTag(ctx context.Context, id int32) (GetTagRow, error)
	GetTagByName(ctx context.Context, name string) (GetTagByNameRow, error)
	IncrementViewCount(ctx context.Context, id int32) (sql.Result, error)
	ListAuthors(ctx context.Context) ([]ListAuthorsRow, error)
	ListPosts(ctx context.Context) ([]ListPostsRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUnpublishedPosts(ctx context.Context) ([]ListUnpublishedPostsRow, error)
	SearchPosts(ctx context.Context, title *int32, arg2 *int32, arg3 *int32, arg4 *int32) ([]SearchPostsRow, error)
	UpdateAuthor(ctx context.Context, name string, arg2 string, arg3 sql.NullString, arg4 int32) (UpdateAuthorRow, error)
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

type QueryResult struct {
	LastInsertID int64
	RowsAffected int64
}
