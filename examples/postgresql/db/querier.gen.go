package postgresqldb

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"
)

type Querier interface {
	AddTagToPost(ctx context.Context, postId *uuid.UUID, postId2 *uuid.UUID) (sql.Result, error)
	CreateComment(ctx context.Context, postId *uuid.UUID, postId2 *uuid.UUID, postId3 pgtype.Text) (CreateCommentRow, error)
	CreatePost(ctx context.Context, userId *uuid.UUID, userId2 pgtype.Text, userId3 pgtype.Text, userId4 pgtype.Text, userId5 pgtype.Bool, userId6 *time.Time) (CreatePostRow, error)
	CreateTag(ctx context.Context, tagname pgtype.Text, tagname2 pgtype.Text) (CreateTagRow, error)
	CreateUser(ctx context.Context, username pgtype.Text, username2 pgtype.Text, username3 *[]byte, username4 pgtype.Text) (CreateUserRow, error)
	DeleteComment(ctx context.Context, id uuid.UUID) (sql.Result, error)
	DeletePost(ctx context.Context, id uuid.UUID) (sql.Result, error)
	DeleteTag(ctx context.Context, id uuid.UUID) (sql.Result, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (sql.Result, error)
	GetCommentById(ctx context.Context, id uuid.UUID) (GetCommentByIdRow, error)
	GetPopularPosts(ctx context.Context, limit *any) ([]GetPopularPostsRow, error)
	GetPostById(ctx context.Context, id uuid.UUID) (GetPostByIdRow, error)
	GetPostsForTag(ctx context.Context, tagId *uuid.UUID) ([]GetPostsForTagRow, error)
	GetTagById(ctx context.Context, id uuid.UUID) (GetTagByIdRow, error)
	GetTagByName(ctx context.Context, tagname pgtype.Text) (GetTagByNameRow, error)
	GetTagsForPost(ctx context.Context, postId *uuid.UUID) ([]GetTagsForPostRow, error)
	GetUserByEmail(ctx context.Context, useremail pgtype.Text) (GetUserByEmailRow, error)
	GetUserById(ctx context.Context, id uuid.UUID) (GetUserByIdRow, error)
	GetUserStats(ctx context.Context, id uuid.UUID) (GetUserStatsRow, error)
	IncrementPostViews(ctx context.Context, id uuid.UUID) (sql.Result, error)
	LikeComment(ctx context.Context, id uuid.UUID) (sql.Result, error)
	ListActiveUsers(ctx context.Context) ([]ListActiveUsersRow, error)
	ListCommentsByPost(ctx context.Context, postId *uuid.UUID) ([]ListCommentsByPostRow, error)
	ListCommentsByUser(ctx context.Context, userId *uuid.UUID) ([]ListCommentsByUserRow, error)
	ListPosts(ctx context.Context) ([]ListPostsRow, error)
	ListPostsByUser(ctx context.Context, userId *uuid.UUID) ([]ListPostsByUserRow, error)
	ListPublishedPosts(ctx context.Context) ([]ListPublishedPostsRow, error)
	ListTags(ctx context.Context) ([]ListTagsRow, error)
	ListUsers(ctx context.Context) ([]ListUsersRow, error)
	RemoveTagFromPost(ctx context.Context, postId *uuid.UUID, tagId *uuid.UUID) (sql.Result, error)
	SearchPostsByCategory(ctx context.Context, any *any) ([]SearchPostsByCategoryRow, error)
	SearchUsersByMetadata(ctx context.Context, p *any) ([]SearchUsersByMetadataRow, error)
	SearchUsersByTag(ctx context.Context, any *any) ([]SearchUsersByTagRow, error)
	UpdateComment(ctx context.Context, commentbody pgtype.Text, id uuid.UUID) (UpdateCommentRow, error)
	UpdatePost(ctx context.Context, title pgtype.Text, postbody pgtype.Text, categories pgtype.Text, isPublished pgtype.Bool, publishedAt *any, id uuid.UUID) (UpdatePostRow, error)
	UpdateTag(ctx context.Context, tagname pgtype.Text, tagdescription pgtype.Text, id uuid.UUID) (UpdateTagRow, error)
	UpdateUser(ctx context.Context, username pgtype.Text, useremail pgtype.Text, metadata *[]byte, tags pgtype.Text, id uuid.UUID) (UpdateUserRow, error)
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
