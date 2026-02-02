package advanceddb

type Orders struct {
	Id          OrderID `json:"id"`
	UserId      UserID  `json:"user_id"`
	Status      Status  `json:"status"`
	TotalAmount Money   `json:"total_amount"`
	CreatedAt   int32   `json:"created_at"`
	UpdatedAt   *int32  `json:"updated_at"`
}
type Products struct {
	Id        ProductID `json:"id"`
	Sku       SKU       `json:"sku"`
	Name      string    `json:"name"`
	Price     Money     `json:"price"`
	CreatedAt int32     `json:"created_at"`
}
type Users struct {
	Id        UserID `json:"id"`
	Email     Email  `json:"email"`
	CreatedAt int32  `json:"created_at"`
}
