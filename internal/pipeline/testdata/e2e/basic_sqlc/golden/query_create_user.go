package basic

import "context"

type CreateUserParams struct {
	Username string
	Email    string
}

const queryCreateUser string = `INSERT INTO users (username, email) VALUES (?, ?) RETURNING id;`

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (int32, error) {
	row := q.db.QueryRowContext(ctx, queryCreateUser, arg.Username, arg.Email)
	if err := row.Err(); err != nil {
		return 0, err
	}
	var item int32
	err := row.Scan(&item)
	if err != nil {
		return 0, err
	}
	return item, nil
}
