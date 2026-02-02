package postgresqldb

import "context"

const queryListActiveUsers string = `SELECT * FROM users WHERE is_active = true ORDER BY created_at DESC;`

func (q *Queries) ListActiveUsers(ctx context.Context) ([]ListActiveUsersRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListActiveUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListActiveUsersRow
	for rows.Next() {
		item, err := scanListActiveUsersRow(rows)
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
