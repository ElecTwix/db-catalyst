package mysqlblog

import "context"

type CreateCommentParams struct {
	PostId  int32
	UserId  int32
	Content string
}

const queryCreateComment string = `INSERT INTO comments (post_id, user_id, content)
VALUES (?, ?, ?);`

func (q *Queries) CreateComment(ctx context.Context, arg CreateCommentParams) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreateComment, arg.PostId, arg.UserId, arg.Content)
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
