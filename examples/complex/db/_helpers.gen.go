package complexdb

import "database/sql"

type CreateAuthorRow struct {
	Id        int32
	Name      interface{}
	Email     interface{}
	Bio       *interface{}
	CreatedAt int32
}

func scanCreateAuthorRow(rows sql.Rows) (CreateAuthorRow, error) {
	var item CreateAuthorRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreatePostRow struct {
	Id        int32
	AuthorId  int32
	Title     interface{}
	Content   interface{}
	Published int32
	ViewCount int32
	CreatedAt int32
	UpdatedAt *int32
}

func scanCreatePostRow(rows sql.Rows) (CreatePostRow, error) {
	var item CreatePostRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreateTagRow struct {
	Id          int32
	Name        interface{}
	Description *interface{}
}

func scanCreateTagRow(rows sql.Rows) (CreateTagRow, error) {
	var item CreateTagRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type GetAuthorRow struct {
	Id        int32
	Name      interface{}
	Email     interface{}
	Bio       *interface{}
	CreatedAt int32
}

func scanGetAuthorRow(rows sql.Rows) (GetAuthorRow, error) {
	var item GetAuthorRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetAuthorStatsRow struct {
	Id         int32
	Name       interface{}
	TotalPosts *int32
	TotalViews *int32
}

func scanGetAuthorStatsRow(rows sql.Rows) (GetAuthorStatsRow, error) {
	var item GetAuthorStatsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.TotalPosts, &item.TotalViews); err != nil {
		return item, err
	}
	return item, nil
}

type GetAuthorWithPostCountRow struct {
	Id         int32
	Name       interface{}
	Email      interface{}
	Bio        *interface{}
	TotalPosts *int32
}

func scanGetAuthorWithPostCountRow(rows sql.Rows) (GetAuthorWithPostCountRow, error) {
	var item GetAuthorWithPostCountRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.TotalPosts); err != nil {
		return item, err
	}
	return item, nil
}

type GetPopularTagsRow struct {
	Id          int32
	Name        interface{}
	Description *interface{}
	PostCount   *int32
}

func scanGetPopularTagsRow(rows sql.Rows) (GetPopularTagsRow, error) {
	var item GetPopularTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.PostCount); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostRow struct {
	Id        int32
	AuthorId  int32
	Title     interface{}
	Content   interface{}
	Published int32
	ViewCount int32
	CreatedAt int32
	UpdatedAt *int32
}

func scanGetPostRow(rows sql.Rows) (GetPostRow, error) {
	var item GetPostRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostTagsRow struct {
	Id          int32
	Name        interface{}
	Description *interface{}
}

func scanGetPostTagsRow(rows sql.Rows) (GetPostTagsRow, error) {
	var item GetPostTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostsByTagRow struct {
	Id        int32
	AuthorId  int32
	Title     interface{}
	Content   interface{}
	Published int32
	ViewCount int32
	CreatedAt int32
	UpdatedAt *int32
}

func scanGetPostsByTagRow(rows sql.Rows) (GetPostsByTagRow, error) {
	var item GetPostsByTagRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagByNameRow struct {
	Id          int32
	Name        interface{}
	Description *interface{}
}

func scanGetTagByNameRow(rows sql.Rows) (GetTagByNameRow, error) {
	var item GetTagByNameRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagRow struct {
	Id          int32
	Name        interface{}
	Description *interface{}
}

func scanGetTagRow(rows sql.Rows) (GetTagRow, error) {
	var item GetTagRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type ListAuthorsRow struct {
	Id        int32
	Name      interface{}
	Email     interface{}
	Bio       *interface{}
	CreatedAt int32
}

func scanListAuthorsRow(rows sql.Rows) (ListAuthorsRow, error) {
	var item ListAuthorsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsRow struct {
	Id        int32
	AuthorId  int32
	Title     interface{}
	Content   interface{}
	Published int32
	ViewCount int32
	CreatedAt int32
	UpdatedAt *int32
}

func scanListPostsRow(rows sql.Rows) (ListPostsRow, error) {
	var item ListPostsRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListTagsRow struct {
	Id          int32
	Name        interface{}
	Description *interface{}
}

func scanListTagsRow(rows sql.Rows) (ListTagsRow, error) {
	var item ListTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description); err != nil {
		return item, err
	}
	return item, nil
}

type ListUnpublishedPostsRow struct {
	Id        int32
	AuthorId  int32
	Title     interface{}
	Content   interface{}
	Published int32
	ViewCount int32
	CreatedAt int32
	UpdatedAt *int32
}

func scanListUnpublishedPostsRow(rows sql.Rows) (ListUnpublishedPostsRow, error) {
	var item ListUnpublishedPostsRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type SearchPostsRow struct {
	Id        int32
	AuthorId  int32
	Title     interface{}
	Content   interface{}
	Published int32
	ViewCount int32
	CreatedAt int32
	UpdatedAt *int32
}

func scanSearchPostsRow(rows sql.Rows) (SearchPostsRow, error) {
	var item SearchPostsRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Content, &item.Published, &item.ViewCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateAuthorRow struct {
	Id        int32
	Name      interface{}
	Email     interface{}
	Bio       *interface{}
	CreatedAt int32
}

func scanUpdateAuthorRow(rows sql.Rows) (UpdateAuthorRow, error) {
	var item UpdateAuthorRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Bio, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
