package simpledb

import "database/sql"

type Users struct {
	Id        int32
	Name      string
	Email     sql.NullString
	CreatedAt int32
}
