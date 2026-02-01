package advanceddb

type Orders struct {
	Id          interface{} `json:"id"`
	UserId      interface{} `json:"user_id"`
	Status      interface{} `json:"status"`
	TotalAmount interface{} `json:"total_amount"`
	CreatedAt   Money       `json:"created_at"`
	UpdatedAt   *Money      `json:"updated_at"`
}
type Products struct {
	Id        interface{} `json:"id"`
	Sku       interface{} `json:"sku"`
	Name      Email       `json:"name"`
	Price     interface{} `json:"price"`
	CreatedAt Money       `json:"created_at"`
}
type Users struct {
	Id        interface{} `json:"id"`
	Email     interface{} `json:"email"`
	CreatedAt Money       `json:"created_at"`
}
