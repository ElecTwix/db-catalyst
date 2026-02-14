package mysqlblog

import (
	"context"
	"database/sql"
)

type UpdatePostParams struct {
	Title   string
	Content sql.NullString
	Status  sql.NullString
	Id      int32
}

const queryUpdatePost string = `UPDATE posts SET title = ?, content = ?, status = ? WHERE id = ?;`

func (q *Queries) UpdatePost(ctx context.Context, arg UpdatePostParams) error {
	_, err := q.db.ExecContext(ctx, queryUpdatePost, arg.Title, arg.Content, arg.Status, arg.Id)
	return err
}
