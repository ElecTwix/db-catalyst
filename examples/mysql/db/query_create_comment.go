package mysqlblog

import "context"

const queryCreateComment string = `INSERT INTO comments (post_id, user_id, content)
VALUES (?, ?, ?);`

func (q *Queries) CreateComment(ctx context.Context, postId int32, userId int32, content string) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreateComment, postId, userId, content)
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
