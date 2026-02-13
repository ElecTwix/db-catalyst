package simpledb

import "database/sql"

type CreateUserRow struct {
	Id        int64
	Name      string
	Email     sql.NullString
	CreatedAt int64
}

func scanCreateUserRow(rows *sql.Rows) (CreateUserRow, error) {
	var item CreateUserRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserRow struct {
	Id        int64
	Name      string
	Email     sql.NullString
	CreatedAt int64
}

func scanGetUserRow(rows *sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListUsersRow struct {
	Id        int64
	Name      string
	Email     sql.NullString
	CreatedAt int64
}

func scanListUsersRow(rows *sql.Rows) (ListUsersRow, error) {
	var item ListUsersRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateUserRow struct {
	Id        int64
	Name      string
	Email     sql.NullString
	CreatedAt int64
}

func scanUpdateUserRow(rows *sql.Rows) (UpdateUserRow, error) {
	var item UpdateUserRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
