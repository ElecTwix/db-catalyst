package mysqlblog

import (
	"context"
	"database/sql"
)

type CreateUserParams struct {
	Email        string
	Username     string
	PasswordHash string
	Status       sql.NullString
}

const queryCreateUser string = `INSERT INTO users (email, username, password_hash, status)
VALUES (?, ?, ?, ?);`

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (QueryResult, error) {
	res, err := q.db.ExecContext(ctx, queryCreateUser, arg.Email, arg.Username, arg.PasswordHash, arg.Status)
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
