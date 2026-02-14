package basic

import (
	"database/sql"
	"time"
)

type GetUserRow struct {
	Id        int32
	Username  string
	Email     string
	CreatedAt *time.Time
}

func scanGetUserRow(rows *sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Username, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListPostsByAuthorRow struct {
	Id       int32
	AuthorId int32
	Title    string
	Body     string
	Status   string
}

func scanListPostsByAuthorRow(rows *sql.Rows) (ListPostsByAuthorRow, error) {
	var item ListPostsByAuthorRow
	if err := rows.Scan(&item.Id, &item.AuthorId, &item.Title, &item.Body, &item.Status); err != nil {
		return item, err
	}
	return item, nil
}
