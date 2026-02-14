package mysqlblog

import (
	"context"
	"database/sql"
)

type CreateTagParams struct {
	Name        string
	Description sql.NullString
}

const queryCreateTag string = `INSERT INTO tags (name, description) VALUES (?, ?);`

func (q *Queries) CreateTag(ctx context.Context, arg CreateTagParams) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreateTag, arg.Name, arg.Description)
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
