package postgresqldb

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"
)

type Querier interface {
	AddTagToPost(ctx context.Context, arg1 *uuid.UUID, arg2 *uuid.UUID) (sql.Result, error)
	CreateComment(ctx context.Context, arg1 *uuid.UUID, arg2 *uuid.UUID, arg3 pgtype.Text) (CreateCommentRow, error)
	CreatePost(ctx context.Context, arg1 *uuid.UUID, arg2 pgtype.Text, arg3 pgtype.Text, arg4 pgtype.Text, arg5 pgtype.Bool, arg52 *time.Time) (CreatePostRow, error)
	CreateTag(ctx context.Context, arg1 pgtype.Text, arg2 pgtype.Text) (CreateTagRow, error)
	CreateUser(ctx context.Context, arg1 pgtype.Text, arg2 pgtype.Text, arg3 *[]byte, arg4 pgtype.Text) (CreateUserRow, error)
	DeleteComment(ctx context.Context, arg1 uuid.UUID) (sql.Result, error)
	DeletePost(ctx context.Context, arg1 uuid.UUID) (sql.Result, error)
	DeleteTag(ctx context.Context, arg1 uuid.UUID) (sql.Result, error)
	DeleteUser(ctx context.Context, arg1 uuid.UUID) (sql.Result, error)
	GetCommentById(ctx context.Context, arg1 uuid.UUID) (GetCommentByIdRow, error)
	GetPopularPosts(ctx context.Context, arg1 pgtype.Int4) ([]GetPopularPostsRow, error)
	GetPostById(ctx context.Context, arg1 uuid.UUID) (GetPostByIdRow, error)
	GetPostsForTag(ctx context.Context, arg1 *uuid.UUID) ([]GetPostsForTagRow, error)
	GetTagById(ctx context.Context, arg1 uuid.UUID) (GetTagByIdRow, error)
	GetTagByName(ctx context.Context, arg1 pgtype.Text) (GetTagByNameRow, error)
	GetTagsForPost(ctx context.Context, arg1 *uuid.UUID) ([]GetTagsForPostRow, error)
	GetUserByEmail(ctx context.Context, arg1 pgtype.Text) (GetUserByEmailRow, error)
	GetUserById(ctx context.Context, arg1 uuid.UUID) (GetUserByIdRow, error)
	GetUserStats(ctx context.Context, arg1 uuid.UUID) (GetUserStatsRow, error)
	IncrementPostViews(ctx context.Context, arg1 uuid.UUID) (sql.Result, error)
	LikeComment(ctx context.Context, arg1 uuid.UUID) (sql.Result, error)
	ListActiveUsers(ctx context.Context) ([]ListActiveUsersRow, error)
	ListCommentsByPost(ctx context.Context, arg1 *uuid.UUID) ([]ListCommentsByPostRow, error)
	ListCommentsByUser(ctx context.Context, arg1 *uuid.UUID) ([]ListCommentsByUserRow, error)
	ListPosts(ctx context.Context) ([]ListPostsRow, error)
	ListPostsByUser(ctx context.Context, arg1 *uuid.UUID) ([]ListPostsByUserRow, error)
	ListPublishedPosts(ctx context.Context) ([]ListPublishedPostsRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	RemoveTagFromPost(ctx context.Context, arg1 *uuid.UUID, arg2 *uuid.UUID) (sql.Result, error)
	SearchPostsByCategory(ctx context.Context, arg1 pgtype.Int4) ([]SearchPostsByCategoryRow, error)
	SearchUsersByMetadata(ctx context.Context, arg1 pgtype.Int4) ([]SearchUsersByMetadataRow, error)
	SearchUsersByTag(ctx context.Context, arg1 pgtype.Int4) ([]SearchUsersByTagRow, error)
	UpdateComment(ctx context.Context, arg2 pgtype.Text, arg1 uuid.UUID) (UpdateCommentRow, error)
	UpdatePost(ctx context.Context, arg2 pgtype.Text, arg3 pgtype.Text, arg4 pgtype.Text, arg5 pgtype.Bool, arg52 pgtype.Int4, arg1 uuid.UUID) (UpdatePostRow, error)
	UpdateTag(ctx context.Context, arg2 pgtype.Text, arg3 pgtype.Text, arg1 uuid.UUID) (UpdateTagRow, error)
	UpdateUser(ctx context.Context, arg2 pgtype.Text, arg3 pgtype.Text, arg4 *[]byte, arg5 pgtype.Text, arg1 uuid.UUID) (UpdateUserRow, error)
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
