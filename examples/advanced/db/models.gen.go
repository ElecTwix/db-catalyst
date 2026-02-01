package advanceddb

import "github.com/electwix/db-catalyst/examples/advanced/types"

type Orders struct {
	Id          interface{}  `json:"id"`
	UserId      interface{}  `json:"user_id"`
	Status      interface{}  `json:"status"`
	TotalAmount interface{}  `json:"total_amount"`
	CreatedAt   types.Money  `json:"created_at"`
	UpdatedAt   *types.Money `json:"updated_at"`
}
type Products struct {
	Id        interface{} `json:"id"`
	Sku       interface{} `json:"sku"`
	Name      types.Email `json:"name"`
	Price     interface{} `json:"price"`
	CreatedAt types.Money `json:"created_at"`
}
type Users struct {
	Id        interface{} `json:"id"`
	Email     interface{} `json:"email"`
	CreatedAt types.Money `json:"created_at"`
}
