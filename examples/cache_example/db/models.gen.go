package cachedb

import "database/sql"

type Posts struct {
	Id        int32
	UserId    int32
	Title     string
	Content   sql.NullString
	Likes     int32
	CreatedAt int32
}
type Users struct {
	Id        int32
	Name      string
	Email     string
	Active    bool
	CreatedAt int32
}
