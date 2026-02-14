package mysqlblog

import (
	"context"
	"database/sql"
)

type CreatePostParams struct {
	UserId  int32
	Title   string
	Content sql.NullString
	Status  sql.NullString
}

const queryCreatePost string = `INSERT INTO posts (user_id, title, content, status)
VALUES (?, ?, ?, ?);`

func (q *Queries) CreatePost(ctx context.Context, arg CreatePostParams) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreatePost, arg.UserId, arg.Title, arg.Content, arg.Status)
	if err != nil {
		return QueryResult{}, err
	}
	result := QueryResult{}
	if v, err := res.LastInsertId(); err == nil {
		result.LastInsertID = v
	}
	if v, err := res.RowsAffected(); err == nil {
		result.RowsAffected = v
	}
	return result, nil
}
