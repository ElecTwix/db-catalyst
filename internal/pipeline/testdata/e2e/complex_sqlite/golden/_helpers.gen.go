package complex

import "database/sql"

type GetItemWithTagsRow struct {
	Id       int32
	Name     interface{}
	Metadata *int32
	Tags     *int32
}

func scanGetItemWithTagsRow(rows sql.Rows) (GetItemWithTagsRow, error) {
	var item GetItemWithTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Metadata, &item.Tags); err != nil {
		return item, err
	}
	return item, nil
}
