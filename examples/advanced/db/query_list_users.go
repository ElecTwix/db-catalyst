package advanceddb

import "context"

const queryListUsers string = `SELECT * FROM users ORDER BY created_at DESC;`

func (q *Queries) ListUsers(ctx context.Context) ([]ListUsersRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListUsersRow
	for rows.Next() {
		item, err := scanListUsersRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
