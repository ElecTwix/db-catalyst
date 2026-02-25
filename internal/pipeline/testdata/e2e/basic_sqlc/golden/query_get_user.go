package basic

import "context"

const queryGetUser string = `SELECT * FROM users WHERE id = :id;`

func (q *Queries) GetUser(ctx context.Context, id int32) (GetUserRow, error) {
	row := q.db.QueryRowContext(ctx, queryGetUser, id)
	if err := row.Err(); err != nil {
		return GetUserRow{}, err
	}
	var item GetUserRow
	err := row.Scan(&item.Id, &item.Username, &item.Email, &item.CreatedAt)
	if err != nil {
		return GetUserRow{}, err
	}
	return item, nil
}
