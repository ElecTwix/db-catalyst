package complexdb

import "database/sql"

type Authors struct {
	Id        int32          `json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Bio       sql.NullString `json:"bio"`
	CreatedAt int32          `json:"created_at"`
}
type Posts struct {
	Id        int32  `json:"id"`
	AuthorId  int32  `json:"author_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Published int32  `json:"published"`
	ViewCount int32  `json:"view_count"`
	CreatedAt int32  `json:"created_at"`
	UpdatedAt *int32 `json:"updated_at"`
}
type Tags struct {
	Id          int32          `json:"id"`
	Name        string         `json:"name"`
	Description sql.NullString `json:"description"`
}
