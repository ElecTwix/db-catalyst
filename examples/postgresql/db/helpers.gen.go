package postgresqldb

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type CreateCommentRow struct {
	Id          uuid.UUID
	PostId      *uuid.UUID
	UserId      *uuid.UUID
	Commentbody pgtype.Text
	Likes       pgtype.Int4
	CreatedAt   *time.Time
}

func scanCreateCommentRow(rows *sql.Rows) (CreateCommentRow, error) {
	var item CreateCommentRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Commentbody, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreatePostRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanCreatePostRow(rows *sql.Rows) (CreatePostRow, error) {
	var item CreatePostRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreateTagRow struct {
	Id             uuid.UUID
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	CreatedAt      *time.Time
}

func scanCreateTagRow(rows *sql.Rows) (CreateTagRow, error) {
	var item CreateTagRow
	if err := rows.Scan(&item.Id, &item.Tagname, &item.Tagdescription, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreateUserRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanCreateUserRow(rows *sql.Rows) (CreateUserRow, error) {
	var item CreateUserRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetCommentByIdRow struct {
	Id          uuid.UUID
	PostId      *uuid.UUID
	UserId      *uuid.UUID
	Commentbody pgtype.Text
	Likes       pgtype.Int4
	CreatedAt   *time.Time
}

func scanGetCommentByIdRow(rows *sql.Rows) (GetCommentByIdRow, error) {
	var item GetCommentByIdRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Commentbody, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPopularPostsRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanGetPopularPostsRow(rows *sql.Rows) (GetPopularPostsRow, error) {
	var item GetPopularPostsRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostByIdRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanGetPostByIdRow(rows *sql.Rows) (GetPostByIdRow, error) {
	var item GetPostByIdRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostsForTagRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanGetPostsForTagRow(rows *sql.Rows) (GetPostsForTagRow, error) {
	var item GetPostsForTagRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagByIdRow struct {
	Id             uuid.UUID
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	CreatedAt      *time.Time
}

func scanGetTagByIdRow(rows *sql.Rows) (GetTagByIdRow, error) {
	var item GetTagByIdRow
	if err := rows.Scan(&item.Id, &item.Tagname, &item.Tagdescription, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagByNameRow struct {
	Id             uuid.UUID
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	CreatedAt      *time.Time
}

func scanGetTagByNameRow(rows *sql.Rows) (GetTagByNameRow, error) {
	var item GetTagByNameRow
	if err := rows.Scan(&item.Id, &item.Tagname, &item.Tagdescription, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagsForPostRow struct {
	Id             uuid.UUID
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	CreatedAt      *time.Time
}

func scanGetTagsForPostRow(rows *sql.Rows) (GetTagsForPostRow, error) {
	var item GetTagsForPostRow
	if err := rows.Scan(&item.Id, &item.Tagname, &item.Tagdescription, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserByEmailRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanGetUserByEmailRow(rows *sql.Rows) (GetUserByEmailRow, error) {
	var item GetUserByEmailRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserByIdRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanGetUserByIdRow(rows *sql.Rows) (GetUserByIdRow, error) {
	var item GetUserByIdRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserStatsRow struct {
	Id           uuid.UUID
	Username     pgtype.Text
	PostCount    int64
	CommentCount int64
	TotalViews   any
}

func scanGetUserStatsRow(rows *sql.Rows) (GetUserStatsRow, error) {
	var item GetUserStatsRow
	if err := rows.Scan(&item.Id, &item.Username, &item.PostCount, &item.CommentCount, &item.TotalViews); err != nil {
		return item, err
	}
	return item, nil
}

type ListActiveUsersRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanListActiveUsersRow(rows *sql.Rows) (ListActiveUsersRow, error) {
	var item ListActiveUsersRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListCommentsByPostRow struct {
	Id          uuid.UUID
	PostId      *uuid.UUID
	UserId      *uuid.UUID
	Commentbody pgtype.Text
	Likes       pgtype.Int4
	CreatedAt   *time.Time
}

func scanListCommentsByPostRow(rows *sql.Rows) (ListCommentsByPostRow, error) {
	var item ListCommentsByPostRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Commentbody, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListCommentsByUserRow struct {
	Id          uuid.UUID
	PostId      *uuid.UUID
	UserId      *uuid.UUID
	Commentbody pgtype.Text
	Likes       pgtype.Int4
	CreatedAt   *time.Time
}

func scanListCommentsByUserRow(rows *sql.Rows) (ListCommentsByUserRow, error) {
	var item ListCommentsByUserRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Commentbody, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsByUserRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanListPostsByUserRow(rows *sql.Rows) (ListPostsByUserRow, error) {
	var item ListPostsByUserRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanListPostsRow(rows *sql.Rows) (ListPostsRow, error) {
	var item ListPostsRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPublishedPostsRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanListPublishedPostsRow(rows *sql.Rows) (ListPublishedPostsRow, error) {
	var item ListPublishedPostsRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListTagsRow struct {
	Id             uuid.UUID
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	CreatedAt      *time.Time
}

func scanListTagsRow(rows *sql.Rows) (ListTagsRow, error) {
	var item ListTagsRow
	if err := rows.Scan(&item.Id, &item.Tagname, &item.Tagdescription, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListUsersRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanListUsersRow(rows *sql.Rows) (ListUsersRow, error) {
	var item ListUsersRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type SearchPostsByCategoryRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanSearchPostsByCategoryRow(rows *sql.Rows) (SearchPostsByCategoryRow, error) {
	var item SearchPostsByCategoryRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type SearchUsersByMetadataRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanSearchUsersByMetadataRow(rows *sql.Rows) (SearchUsersByMetadataRow, error) {
	var item SearchUsersByMetadataRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type SearchUsersByTagRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanSearchUsersByTagRow(rows *sql.Rows) (SearchUsersByTagRow, error) {
	var item SearchUsersByTagRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateCommentRow struct {
	Id          uuid.UUID
	PostId      *uuid.UUID
	UserId      *uuid.UUID
	Commentbody pgtype.Text
	Likes       pgtype.Int4
	CreatedAt   *time.Time
}

func scanUpdateCommentRow(rows *sql.Rows) (UpdateCommentRow, error) {
	var item UpdateCommentRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Commentbody, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdatePostRow struct {
	Id          uuid.UUID
	UserId      *uuid.UUID
	Title       pgtype.Text
	Postbody    pgtype.Text
	Categories  pgtype.Text
	ViewCount   pgtype.Int4
	Rating      *decimal.Decimal
	IsPublished pgtype.Bool
	PublishedAt *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func scanUpdatePostRow(rows *sql.Rows) (UpdatePostRow, error) {
	var item UpdatePostRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Postbody, &item.Categories, &item.ViewCount, &item.Rating, &item.IsPublished, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateTagRow struct {
	Id             uuid.UUID
	Tagname        pgtype.Text
	Tagdescription pgtype.Text
	CreatedAt      *time.Time
}

func scanUpdateTagRow(rows *sql.Rows) (UpdateTagRow, error) {
	var item UpdateTagRow
	if err := rows.Scan(&item.Id, &item.Tagname, &item.Tagdescription, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateUserRow struct {
	Id        uuid.UUID
	Username  pgtype.Text
	Useremail pgtype.Text
	Metadata  *[]byte
	Tags      pgtype.Text
	IsActive  pgtype.Bool
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func scanUpdateUserRow(rows *sql.Rows) (UpdateUserRow, error) {
	var item UpdateUserRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Useremail, &item.Metadata, &item.Tags, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}
