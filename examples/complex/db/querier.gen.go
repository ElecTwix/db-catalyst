package complexdb

import (
	"context"
	"database/sql"
)

type Querier interface {
	AddTagToPost(ctx context.Context, postId int64, tagId int64) (sql.Result, error)
	CreateAuthor(ctx context.Context, name string, email string, bio sql.NullString) (CreateAuthorRow, error)
	CreatePost(ctx context.Context, authorId int64, title string, content string, published int64) (CreatePostRow, error)
	CreateTag(ctx context.Context, name string, description sql.NullString) (CreateTagRow, error)
	DeleteAuthor(ctx context.Context, id int64) (sql.Result, error)
	DeleteTag(ctx context.Context, id int64) (sql.Result, error)
	GetAuthor(ctx context.Context, id int64) (GetAuthorRow, error)
	GetAuthorStats(ctx context.Context, id int64) (GetAuthorStatsRow, error)
	GetAuthorWithPostCount(ctx context.Context, id int64) (GetAuthorWithPostCountRow, error)
	GetPopularTags(ctx context.Context, limit *any) ([]GetPopularTagsRow, error)
	GetPost(ctx context.Context, id int64) (GetPostRow, error)
	GetPostTags(ctx context.Context, postId int64) ([]GetPostTagsRow, error)
	GetPostsByTag(ctx context.Context, name string) ([]GetPostsByTagRow, error)
	GetTag(ctx context.Context, id int64) (GetTagRow, error)
	GetTagByName(ctx context.Context, name string) (GetTagByNameRow, error)
	IncrementViewCount(ctx context.Context, id int64) (sql.Result, error)
	ListAuthors(ctx context.Context) ([]ListAuthorsRow, error)
	ListPosts(ctx context.Context) ([]ListPostsRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUnpublishedPosts(ctx context.Context) ([]ListUnpublishedPostsRow, error)
	SearchPosts(ctx context.Context, title *any, content *any, limit *any, offset *any) ([]SearchPostsRow, error)
	UpdateAuthor(ctx context.Context, name string, email string, bio sql.NullString, id int64) (UpdateAuthorRow, error)
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
