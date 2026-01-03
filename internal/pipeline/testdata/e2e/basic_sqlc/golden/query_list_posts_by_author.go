package basic

import "context"

const queryListPostsByAuthor string = `SELECT * FROM posts WHERE author_id = :author_id ORDER BY id DESC;`

func (q *Queries) ListPostsByAuthor(ctx context.Context, authorId int32) ([]ListPostsByAuthorRow, error) {
	rows, err := q.db.QueryContext(ctx, queryListPostsByAuthor, authorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ListPostsByAuthorRow, 0)
	for rows.Next() {
		item, err := scanListPostsByAuthorRow(rows)
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
