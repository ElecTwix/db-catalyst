package mysqlblog

import (
	"context"
	"database/sql"
)

const queryCreatePost string = `INSERT INTO posts (user_id, title, content, status)
VALUES (?, ?, ?, ?);`

func (q *Queries) CreatePost(ctx context.Context, userId int32, title string, content sql.NullString, status sql.NullString) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreatePost, userId, title, content, status)
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
