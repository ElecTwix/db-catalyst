package mysqlblog

import (
	"context"
	"database/sql"
)

type Querier interface {
	AddTagToPost(ctx context.Context, postId int32, tagId int32) (sql.Result, error)
	CountCommentsByPost(ctx context.Context, postId int32) (CountCommentsByPostRow, error)
	CreateComment(ctx context.Context, postId int32, userId int32, content string) (QueryResult, error)
	CreatePost(ctx context.Context, userId int32, title string, content sql.NullString, status sql.NullString) (QueryResult, error)
	CreateTag(ctx context.Context, name string, description sql.NullString) (QueryResult, error)
	CreateUser(ctx context.Context, email string, username string, passwordHash string, status sql.NullString) (QueryResult, error)
	DeleteComment(ctx context.Context, id int32) (sql.Result, error)
	DeletePost(ctx context.Context, id int32) (sql.Result, error)
	DeleteTag(ctx context.Context, id int32) (sql.Result, error)
	GetComment(ctx context.Context, id int32) (GetCommentRow, error)
	GetPost(ctx context.Context, id int32) (GetPostRow, error)
	GetPostsForTag(ctx context.Context, tagId int32, limit *any) ([]GetPostsForTagRow, error)
	GetTag(ctx context.Context, id int32) (GetTagRow, error)
	GetTagByName(ctx context.Context, name string) (GetTagByNameRow, error)
	GetTagsForPost(ctx context.Context, postId int32) ([]GetTagsForPostRow, error)
	GetUser(ctx context.Context, id int32) (GetUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error)
	IncrementPostViews(ctx context.Context, id int32) (sql.Result, error)
	ListCommentsByPost(ctx context.Context, postId int32, limit *any) ([]ListCommentsByPostRow, error)
	ListCommentsByUser(ctx context.Context, userId int32, limit *any) ([]ListCommentsByUserRow, error)
	ListPosts(ctx context.Context, status sql.NullString, limit *any) ([]ListPostsRow, error)
	ListPostsByUser(ctx context.Context, userId int32, status sql.NullString, limit *any) ([]ListPostsByUserRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUsers(ctx context.Context, status sql.NullString, limit *any) ([]ListUsersRow, error)
	RemoveTagFromPost(ctx context.Context, postId int32, tagId int32) (sql.Result, error)
	UpdatePost(ctx context.Context, title string, content sql.NullString, status sql.NullString, id int32) (sql.Result, error)
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
