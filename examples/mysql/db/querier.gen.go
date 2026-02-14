package mysqlblog

import (
	"context"
	"database/sql"
)

type Querier interface {
	AddTagToPost(ctx context.Context, arg AddTagToPostParams) error
	CountCommentsByPost(ctx context.Context, postId int32) (int64, error)
	CreateComment(ctx context.Context, arg CreateCommentParams) (QueryResult, error)
	CreatePost(ctx context.Context, arg CreatePostParams) (QueryResult, error)
	CreateTag(ctx context.Context, arg CreateTagParams) (QueryResult, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (QueryResult, error)
	DeleteComment(ctx context.Context, id int32) error
	DeletePost(ctx context.Context, id int32) error
	DeleteTag(ctx context.Context, id int32) error
	GetComment(ctx context.Context, id int32) (GetCommentRow, error)
	GetPost(ctx context.Context, id int32) (GetPostRow, error)
	GetPostsForTag(ctx context.Context, arg GetPostsForTagParams) ([]GetPostsForTagRow, error)
	GetTag(ctx context.Context, id int32) (GetTagRow, error)
	GetTagByName(ctx context.Context, name string) (GetTagByNameRow, error)
	GetTagsForPost(ctx context.Context, postId int32) ([]GetTagsForPostRow, error)
	GetUser(ctx context.Context, id int32) (GetUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error)
	IncrementPostViews(ctx context.Context, id int32) error
	ListCommentsByPost(ctx context.Context, arg ListCommentsByPostParams) ([]ListCommentsByPostRow, error)
	ListCommentsByUser(ctx context.Context, arg ListCommentsByUserParams) ([]ListCommentsByUserRow, error)
	ListPosts(ctx context.Context, arg ListPostsParams) ([]ListPostsRow, error)
	ListPostsByUser(ctx context.Context, arg ListPostsByUserParams) ([]ListPostsByUserRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUsers(ctx context.Context, arg ListUsersParams) ([]ListUsersRow, error)
	RemoveTagFromPost(ctx context.Context, arg RemoveTagFromPostParams) error
	UpdatePost(ctx context.Context, arg UpdatePostParams) error
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
