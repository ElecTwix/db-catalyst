package postgresqldb

import (
	"context"

	"github.com/google/uuid"
)

const queryGetPostsForTag string = `SELECT p.* FROM posts p
JOIN post_tags pt ON p.id = pt.post_id
WHERE pt.tag_id = $1 AND p.is_published = true;

-- Complex queries`

func (q *Queries) GetPostsForTag(ctx context.Context, tagId *uuid.UUID) ([]GetPostsForTagRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetPostsForTag, tagId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPostsForTagRow
	for rows.Next() {
		item, err := scanGetPostsForTagRow(rows)
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
