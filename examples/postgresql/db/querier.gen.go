package postgresqldb

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"
)

type Querier interface {
	AddTagToPost(ctx context.Context, arg AddTagToPostParams) error
	CreateComment(ctx context.Context, arg CreateCommentParams) (CreateCommentRow, error)
	CreatePost(ctx context.Context, arg CreatePostParams) (CreatePostRow, error)
	CreateTag(ctx context.Context, arg CreateTagParams) (CreateTagRow, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error)
	DeleteComment(ctx context.Context, id uuid.UUID) error
	DeletePost(ctx context.Context, id uuid.UUID) error
	DeleteTag(ctx context.Context, id uuid.UUID) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	GetCommentById(ctx context.Context, id uuid.UUID) (GetCommentByIdRow, error)
	GetPopularPosts(ctx context.Context, limit any) ([]GetPopularPostsRow, error)
	GetPostById(ctx context.Context, id uuid.UUID) (GetPostByIdRow, error)
	GetPostsForTag(ctx context.Context, tagId *uuid.UUID) ([]GetPostsForTagRow, error)
	GetTagById(ctx context.Context, id uuid.UUID) (GetTagByIdRow, error)
	GetTagByName(ctx context.Context, tagname pgtype.Text) (GetTagByNameRow, error)
	GetTagsForPost(ctx context.Context, postId *uuid.UUID) ([]GetTagsForPostRow, error)
	GetUserByEmail(ctx context.Context, useremail pgtype.Text) (GetUserByEmailRow, error)
	GetUserById(ctx context.Context, id uuid.UUID) (GetUserByIdRow, error)
	GetUserStats(ctx context.Context, id uuid.UUID) (GetUserStatsRow, error)
	IncrementPostViews(ctx context.Context, id uuid.UUID) error
	LikeComment(ctx context.Context, id uuid.UUID) error
	ListActiveUsers(ctx context.Context) ([]ListActiveUsersRow, error)
	ListCommentsByPost(ctx context.Context, postId *uuid.UUID) ([]ListCommentsByPostRow, error)
	ListCommentsByUser(ctx context.Context, userId *uuid.UUID) ([]ListCommentsByUserRow, error)
	ListPosts(ctx context.Context) ([]ListPostsRow, error)
	ListPostsByUser(ctx context.Context, userId *uuid.UUID) ([]ListPostsByUserRow, error)
	ListPublishedPosts(ctx context.Context) ([]ListPublishedPostsRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	RemoveTagFromPost(ctx context.Context, arg RemoveTagFromPostParams) error
	SearchPostsByCategory(ctx context.Context, any any) ([]SearchPostsByCategoryRow, error)
	SearchUsersByMetadata(ctx context.Context, p any) ([]SearchUsersByMetadataRow, error)
	SearchUsersByTag(ctx context.Context, any any) ([]SearchUsersByTagRow, error)
	UpdateComment(ctx context.Context, arg UpdateCommentParams) (UpdateCommentRow, error)
	UpdatePost(ctx context.Context, arg UpdatePostParams) (UpdatePostRow, error)
	UpdateTag(ctx context.Context, arg UpdateTagParams) (UpdateTagRow, error)
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
