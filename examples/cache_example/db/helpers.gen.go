package cachedb

import "database/sql"

type GetPopularPostsRow struct {
	Id        int64
	UserId    int64
	Title     string
	Content   sql.NullString
	Likes     int64
	CreatedAt int64
}

func scanGetPopularPostsRow(rows *sql.Rows) (GetPopularPostsRow, error) {
	var item GetPopularPostsRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserByEmailRow struct {
	Id        int64
	Name      string
	Email     string
	Active    bool
	CreatedAt int64
}

func scanGetUserByEmailRow(rows *sql.Rows) (GetUserByEmailRow, error) {
	var item GetUserByEmailRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Active, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserPostsRow struct {
	Id        int64
	UserId    int64
	Title     string
	Content   sql.NullString
	Likes     int64
	CreatedAt int64
}

func scanGetUserPostsRow(rows *sql.Rows) (GetUserPostsRow, error) {
	var item GetUserPostsRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Title, &item.Content, &item.Likes, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserRow struct {
	Id        int64
	Name      string
	Email     string
	Active    bool
	CreatedAt int64
}

func scanGetUserRow(rows *sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Active, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListActiveUsersRow struct {
	Id        int64
	Name      string
	Email     string
	Active    bool
	CreatedAt int64
}

func scanListActiveUsersRow(rows *sql.Rows) (ListActiveUsersRow, error) {
	var item ListActiveUsersRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.Active, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
