package mysqlblog

import "database/sql"

type Comments struct {
	Id        int32          `json:"id"`
	PostId    int32          `json:"post_id"`
	UserId    int32          `json:"user_id"`
	Content   string         `json:"content"`
	CreatedAt sql.NullString `json:"created_at"`
}
type Posts struct {
	Id        int32          `json:"id"`
	UserId    int32          `json:"user_id"`
	Title     string         `json:"title"`
	Content   sql.NullString `json:"content"`
	Status    sql.NullString `json:"status"`
	ViewCount *int32         `json:"view_count"`
	CreatedAt sql.NullString `json:"created_at"`
}
type Tags struct {
	Id          int32          `json:"id"`
	Name        string         `json:"name"`
	Description sql.NullString `json:"description"`
	CreatedAt   sql.NullString `json:"created_at"`
}
type Users struct {
	Id           int32          `json:"id"`
	Email        string         `json:"email"`
	Username     string         `json:"username"`
	PasswordHash string         `json:"password_hash"`
	Status       sql.NullString `json:"status"`
	CreatedAt    sql.NullString `json:"created_at"`
}
