package advanceddb

import (
	"context"
	"database/sql"
)

const queryGetOrderStatistics string = `SELECT 
    COUNT(*) as total_orders,
    COALESCE(SUM(total_amount), 0) as total_revenue
FROM orders
WHERE user_id = ?;`

func (q *Queries) GetOrderStatistics(ctx context.Context, userId any) (GetOrderStatisticsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryGetOrderStatistics, userId)
	if err != nil {
		return GetOrderStatisticsRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetOrderStatisticsRow{}, err
		}
		return GetOrderStatisticsRow{}, sql.ErrNoRows
	}
	item, err := scanGetOrderStatisticsRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
