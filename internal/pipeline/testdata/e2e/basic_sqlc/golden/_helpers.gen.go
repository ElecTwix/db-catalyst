package basic

import "database/sql"

type CreateUserRow struct {
	Id int32
}

func scanCreateUserRow(rows sql.Rows) (CreateUserRow, error) {
	var item CreateUserRow
	if err := rows.Scan(&item.Id); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserRow struct {
	Id        int32
	Username  interface{}
	Email     interface{}
	CreatedAt *int32
}

func scanGetUserRow(rows sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsByAuthorRow struct {
	Id       int32
	AuthorId int32
	Title    interface{}
	Body     interface{}
	Status   interface{}
}

func scanListPostsByAuthorRow(rows sql.Rows) (ListPostsByAuthorRow, error) {
	var item ListPostsByAuthorRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Body, &item.Status); err != nil {
		return item, err
	}
	return item, nil
}
