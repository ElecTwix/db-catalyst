package mysqlblog

import (
	"context"
	"database/sql"
)

const queryCreateUser string = `INSERT INTO users (email, username, password_hash, status)
VALUES (?, ?, ?, ?);`

func (q *Queries) CreateUser(ctx context.Context, email string, username string, passwordHash string, status sql.NullString) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreateUser, email, username, passwordHash, status)
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
