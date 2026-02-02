package complex

import "database/sql"

type GetItemWithTagsRow struct {
	Id       int32
	Name     string
	Metadata *any
	Tags     *any
}

func scanGetItemWithTagsRow(rows sql.Rows) (GetItemWithTagsRow, error) {
	var item GetItemWithTagsRow
	if err := rows.Scan(&item.Id, &item.Name, &item.Metadata, &item.Tags); err != nil {
		return item, err
	}
	return item, nil
}
