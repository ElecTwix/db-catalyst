package mysqlblog

import (
	"context"
	"database/sql"
)

const queryGetComment string = `SELECT * FROM comments WHERE id = ?;`

func (q *Queries) GetComment(ctx context.Context, id int32) (GetCommentRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetComment, id)
	if err != nil {
		return GetCommentRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetCommentRow{}, err
		}
		return GetCommentRow{}, sql.ErrNoRows
	}
	item, err := scanGetCommentRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
