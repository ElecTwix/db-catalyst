package mysqlblog

import "database/sql"

type GetCommentRow struct {
	Id        int32
	PostId    int32
	UserId    int32
	Content   string
	CreatedAt sql.NullString
}

func scanGetCommentRow(rows *sql.Rows) (GetCommentRow, error) {
	var item GetCommentRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Content, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostRow struct {
	Id        int32
	UserId    int32
	Title     string
	Content   sql.NullString
	Status    sql.NullString
	ViewCount *int32
	CreatedAt sql.NullString
}

func scanGetPostRow(rows *sql.Rows) (GetPostRow, error) {
	var item GetPostRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.Status, &item.ViewCount, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetPostsForTagRow struct {
	Id        int32
	UserId    int32
	Title     string
	Content   sql.NullString
	Status    sql.NullString
	ViewCount *int32
	CreatedAt sql.NullString
}

func scanGetPostsForTagRow(rows *sql.Rows) (GetPostsForTagRow, error) {
	var item GetPostsForTagRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.Status, &item.ViewCount, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagByNameRow struct {
	Id          int32
	Name        string
	Description sql.NullString
	CreatedAt   sql.NullString
}

func scanGetTagByNameRow(rows *sql.Rows) (GetTagByNameRow, error) {
	var item GetTagByNameRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagRow struct {
	Id          int32
	Name        string
	Description sql.NullString
	CreatedAt   sql.NullString
}

func scanGetTagRow(rows *sql.Rows) (GetTagRow, error) {
	var item GetTagRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetTagsForPostRow struct {
	Id          int32
	Name        string
	Description sql.NullString
	CreatedAt   sql.NullString
}

func scanGetTagsForPostRow(rows *sql.Rows) (GetTagsForPostRow, error) {
	var item GetTagsForPostRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserByEmailRow struct {
	Id           int32
	Email        string
	Username     string
	PasswordHash string
	Status       sql.NullString
	CreatedAt    sql.NullString
}

func scanGetUserByEmailRow(rows *sql.Rows) (GetUserByEmailRow, error) {
	var item GetUserByEmailRow
	if err := rows.Scan(&item.Id, &item.Email, &item.Username, &item.PasswordHash, &item.Status, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserRow struct {
	Id           int32
	Email        string
	Username     string
	PasswordHash string
	Status       sql.NullString
	CreatedAt    sql.NullString
}

func scanGetUserRow(rows *sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Email, &item.Username, &item.PasswordHash, &item.Status, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListCommentsByPostRow struct {
	Id        int32
	PostId    int32
	UserId    int32
	Content   string
	CreatedAt sql.NullString
}

func scanListCommentsByPostRow(rows *sql.Rows) (ListCommentsByPostRow, error) {
	var item ListCommentsByPostRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Content, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListCommentsByUserRow struct {
	Id        int32
	PostId    int32
	UserId    int32
	Content   string
	CreatedAt sql.NullString
}

func scanListCommentsByUserRow(rows *sql.Rows) (ListCommentsByUserRow, error) {
	var item ListCommentsByUserRow
	if err := rows.Scan(&item.Id, &item.PostId, &item.UserId, &item.Content, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsByUserRow struct {
	Id        int32
	UserId    int32
	Title     string
	Content   sql.NullString
	Status    sql.NullString
	ViewCount *int32
	CreatedAt sql.NullString
}

func scanListPostsByUserRow(rows *sql.Rows) (ListPostsByUserRow, error) {
	var item ListPostsByUserRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.Status, &item.ViewCount, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsRow struct {
	Id        int32
	UserId    int32
	Title     string
	Content   sql.NullString
	Status    sql.NullString
	ViewCount *int32
	CreatedAt sql.NullString
}

func scanListPostsRow(rows *sql.Rows) (ListPostsRow, error) {
	var item ListPostsRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.Status, &item.ViewCount, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListTagsRow struct {
	Id          int32
	Name        string
	Description sql.NullString
	CreatedAt   sql.NullString
}

func scanListTagsRow(rows *sql.Rows) (ListTagsRow, error) {
	var item ListTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListUsersRow struct {
	Id           int32
	Email        string
	Username     string
	PasswordHash string
	Status       sql.NullString
	CreatedAt    sql.NullString
}

func scanListUsersRow(rows *sql.Rows) (ListUsersRow, error) {
	var item ListUsersRow
	if err := rows.Scan(&item.Id, &item.Email, &item.Username, &item.PasswordHash, &item.Status, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
