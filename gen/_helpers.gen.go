package test

import "database/sql"

type GetUserRow struct {
	Id          interface{}
	Name        interface{}
	TriggerType *interface{}
}

func scanGetUserRow(rows sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Name, &item.TriggerType); err != nil {
		return item, err
	}
	return item, nil
}
