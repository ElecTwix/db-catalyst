package complexdb

import "database/sql"

type CreateAuthorRow struct {
	Id        int64
	Name      string
	Email     string
	Bio       sql.NullString
	CreatedAt int64
}

func scanCreateAuthorRow(rows sql.Rows) (CreateAuthorRow, error) {
	var item CreateAuthorRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreatePostRow struct {
	Id        int64
	AuthorId  int64
	Title     string
	Content   string
	Published int64
	ViewCount int64
	CreatedAt int64
	UpdatedAt sql.NullInt64
}

func scanCreatePostRow(rows sql.Rows) (CreatePostRow, error) {
	var item CreatePostRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreateTagRow struct {
	Id          int64
	Name        string
	Description sql.NullString
}

func scanCreateTagRow(rows sql.Rows) (CreateTagRow, error) {
	var item CreateTagRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type GetAuthorRow struct {
	Id        int64
	Name      string
	Email     string
	Bio       sql.NullString
	CreatedAt int64
}

func scanGetAuthorRow(rows sql.Rows) (GetAuthorRow, error) {
	var item GetAuthorRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetAuthorStatsRow struct {
	Id         int64
	Name       string
	TotalPosts *any
	TotalViews *any
}

func scanGetAuthorStatsRow(rows sql.Rows) (GetAuthorStatsRow, error) {
	var item GetAuthorStatsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.TotalPosts, &item.TotalViews); err != nil {
		return item, err
	}
	return item, nil
}

type GetAuthorWithPostCountRow struct {
	Id         int64
	Name       string
	Email      string
	Bio        sql.NullString
	TotalPosts *any
}

func scanGetAuthorWithPostCountRow(rows sql.Rows) (GetAuthorWithPostCountRow, error) {
	var item GetAuthorWithPostCountRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.TotalPosts); err != nil {
		return item, err
	}
	return item, nil
}

type GetPopularTagsRow struct {
	Id          int64
	Name        string
	Description sql.NullString
	PostCount   *any
}

func scanGetPopularTagsRow(rows sql.Rows) (GetPopularTagsRow, error) {
	var item GetPopularTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.PostCount); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostRow struct {
	Id        int64
	AuthorId  int64
	Title     string
	Content   string
	Published int64
	ViewCount int64
	CreatedAt int64
	UpdatedAt sql.NullInt64
}

func scanGetPostRow(rows sql.Rows) (GetPostRow, error) {
	var item GetPostRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostTagsRow struct {
	Id          int64
	Name        string
	Description sql.NullString
}

func scanGetPostTagsRow(rows sql.Rows) (GetPostTagsRow, error) {
	var item GetPostTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostsByTagRow struct {
	Id        int64
	AuthorId  int64
	Title     string
	Content   string
	Published int64
	ViewCount int64
	CreatedAt int64
	UpdatedAt sql.NullInt64
}

func scanGetPostsByTagRow(rows sql.Rows) (GetPostsByTagRow, error) {
	var item GetPostsByTagRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagByNameRow struct {
	Id          int64
	Name        string
	Description sql.NullString
}

func scanGetTagByNameRow(rows sql.Rows) (GetTagByNameRow, error) {
	var item GetTagByNameRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagRow struct {
	Id          int64
	Name        string
	Description sql.NullString
}

func scanGetTagRow(rows sql.Rows) (GetTagRow, error) {
	var item GetTagRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type ListAuthorsRow struct {
	Id        int64
	Name      string
	Email     string
	Bio       sql.NullString
	CreatedAt int64
}

func scanListAuthorsRow(rows sql.Rows) (ListAuthorsRow, error) {
	var item ListAuthorsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsRow struct {
	Id        int64
	AuthorId  int64
	Title     string
	Content   string
	Published int64
	ViewCount int64
	CreatedAt int64
	UpdatedAt sql.NullInt64
}

func scanListPostsRow(rows sql.Rows) (ListPostsRow, error) {
	var item ListPostsRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListTagsRow struct {
	Id          int64
	Name        string
	Description sql.NullString
}

func scanListTagsRow(rows sql.Rows) (ListTagsRow, error) {
	var item ListTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type ListUnpublishedPostsRow struct {
	Id        int64
	AuthorId  int64
	Title     string
	Content   string
	Published int64
	ViewCount int64
	CreatedAt int64
	UpdatedAt sql.NullInt64
}

func scanListUnpublishedPostsRow(rows sql.Rows) (ListUnpublishedPostsRow, error) {
	var item ListUnpublishedPostsRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type SearchPostsRow struct {
	Id        int64
	AuthorId  int64
	Title     string
	Content   string
	Published int64
	ViewCount int64
	CreatedAt int64
	UpdatedAt sql.NullInt64
}

func scanSearchPostsRow(rows sql.Rows) (SearchPostsRow, error) {
	var item SearchPostsRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateAuthorRow struct {
	Id        int64
	Name      string
	Email     string
	Bio       sql.NullString
	CreatedAt int64
}

func scanUpdateAuthorRow(rows sql.Rows) (UpdateAuthorRow, error) {
	var item UpdateAuthorRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
