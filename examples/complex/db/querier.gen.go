package complexdb

import (
	"context"
	"database/sql"
)

type Querier interface {
	AddTagToPost(ctx context.Context, arg AddTagToPostParams) error
	CreateAuthor(ctx context.Context, arg CreateAuthorParams) (CreateAuthorRow, error)
	CreatePost(ctx context.Context, arg CreatePostParams) (CreatePostRow, error)
	CreateTag(ctx context.Context, arg CreateTagParams) (CreateTagRow, error)
	DeleteAuthor(ctx context.Context, id int64) error
	DeleteTag(ctx context.Context, id int64) error
	GetAuthor(ctx context.Context, id int64) (GetAuthorRow, error)
	GetAuthorStats(ctx context.Context, id int64) (GetAuthorStatsRow, error)
	GetAuthorWithPostCount(ctx context.Context, id int64) (GetAuthorWithPostCountRow, error)
	GetPopularTags(ctx context.Context, limit any) ([]GetPopularTagsRow, error)
	GetPost(ctx context.Context, id int64) (GetPostRow, error)
	GetPostTags(ctx context.Context, postId int64) ([]GetPostTagsRow, error)
	GetPostsByTag(ctx context.Context, name string) ([]GetPostsByTagRow, error)
	GetTag(ctx context.Context, id int64) (GetTagRow, error)
	GetTagByName(ctx context.Context, name string) (GetTagByNameRow, error)
	IncrementViewCount(ctx context.Context, id int64) error
	ListAuthors(ctx context.Context) ([]ListAuthorsRow, error)
	ListPosts(ctx context.Context) ([]ListPostsRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUnpublishedPosts(ctx context.Context) ([]ListUnpublishedPostsRow, error)
	SearchPosts(ctx context.Context, arg SearchPostsParams) ([]SearchPostsRow, error)
	UpdateAuthor(ctx context.Context, arg UpdateAuthorParams) (UpdateAuthorRow, error)
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
