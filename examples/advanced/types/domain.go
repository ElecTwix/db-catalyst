package types

// UserID is a strongly-typed user identifier
type UserID int64

// ProductID is a strongly-typed product identifier
type ProductID int64

// OrderID is a strongly-typed order identifier
type OrderID int64

// Status represents an order status
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusShipped    Status = "shipped"
	StatusDelivered  Status = "delivered"
	StatusCancelled  Status = "cancelled"
)

// Money represents monetary values in cents (INTEGER in SQLite)
type Money int64

// Dollars converts Money to dollar amount
func (m Money) Dollars() float64 {
	return float64(m) / 100
}

// FromDollars creates Money from dollar amount
func FromDollars(d float64) Money {
	return Money(d * 100)
}

// Email is a validated email type
type Email string

// SKU is a product stock keeping unit
type SKU string
