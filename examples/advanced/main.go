package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/electwix/db-catalyst/examples/advanced/db"
	"github.com/electwix/db-catalyst/examples/advanced/types"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	// Open database
	sqlDB, err := sql.Open("modernc.org/sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Initialize schema
	if _, err := sqlDB.Exec(schema); err != nil {
		log.Fatal(err)
	}

	queries := db.New(sqlDB)

	fmt.Println("=== Advanced Features Demo ===\n")

	// Create users with strongly-typed IDs
	fmt.Println("1. Creating users with custom types...")
	user1, err := queries.CreateUser(ctx, "alice@example.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created user: ID=%v, Email=%s\n", user1.ID, user1.Email)

	user2, err := queries.CreateUser(ctx, "bob@example.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created user: ID=%v, Email=%s\n", user2.ID, user2.Email)

	// Create products with Money type
	fmt.Println("\n2. Creating products with Money type...")
	product1, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Sku:   "LAPTOP-001",
		Name:  "Gaming Laptop",
		Price: types.FromDollars(1299.99),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created product: ID=%v, SKU=%s, Price=$%.2f\n",
		product1.ID, product1.Sku, product1.Price.Dollars())

	product2, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Sku:   "MOUSE-001",
		Name:  "Wireless Mouse",
		Price: types.FromDollars(49.99),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created product: ID=%v, SKU=%s, Price=$%.2f\n",
		product2.ID, product2.Sku, product2.Price.Dollars())

	product3, err := queries.CreateProduct(ctx, db.CreateProductParams{
		Sku:   "KEYBOARD-001",
		Name:  "Mechanical Keyboard",
		Price: types.FromDollars(149.99),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created product: ID=%v, SKU=%s, Price=$%.2f\n",
		product3.ID, product3.Sku, product3.Price.Dollars())

	// Create order with custom types
	fmt.Println("\n3. Creating orders with Status enum...")
	order1, err := queries.CreateOrder(ctx, db.CreateOrderParams{
		UserID:      user1.ID,
		Status:      types.StatusPending,
		TotalAmount: 0,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created order: ID=%v, User=%v, Status=%s\n",
		order1.ID, order1.UserID, order1.Status)

	// Add items to order
	fmt.Println("\n4. Adding items to order...")
	_, err = queries.AddOrderItem(ctx, db.AddOrderItemParams{
		OrderID:   order1.ID,
		ProductID: product1.ID,
		Quantity:  1,
		UnitPrice: product1.Price,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Added: %s x 1 @ $%.2f\n", product1.Name, product1.Price.Dollars())

	_, err = queries.AddOrderItem(ctx, db.AddOrderItemParams{
		OrderID:   order1.ID,
		ProductID: product2.ID,
		Quantity:  2,
		UnitPrice: product2.Price,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Added: %s x 2 @ $%.2f each\n", product2.Name, product2.Price.Dollars())

	// Update order total
	if err := queries.UpdateOrderTotal(ctx, db.UpdateOrderTotalParams{
		OrderID: order1.ID,
		ID:      order1.ID,
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   Order total updated")

	// Update order status
	fmt.Println("\n5. Updating order status...")
	updatedOrder, err := queries.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:     order1.ID,
		Status: types.StatusProcessing,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Order status: %s -> %s\n", order1.Status, updatedOrder.Status)

	// Query with custom types
	fmt.Println("\n6. Querying with custom types...")
	fetchedOrder, err := queries.GetOrder(ctx, order1.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Fetched order: ID=%v, User=%v, Status=%s, Total=$%.2f\n",
		fetchedOrder.ID, fetchedOrder.UserID, fetchedOrder.Status, fetchedOrder.TotalAmount.Dollars())

	// Get order items
	fmt.Println("\n7. Getting order items...")
	items, err := queries.GetOrderItems(ctx, order1.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   Order items:")
	for _, item := range items {
		fmt.Printf("     - %s (SKU: %s): %d x $%.2f = $%.2f\n",
			item.ProductName, item.ProductSku, item.Quantity,
			item.UnitPrice.Dollars(),
			float64(item.Quantity)*item.UnitPrice.Dollars())
	}

	// Get user orders
	fmt.Println("\n8. Getting user order statistics...")
	stats, err := queries.GetOrderStatistics(ctx, user1.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   User %s statistics:\n", user1.Email)
	fmt.Printf("     Total orders: %d\n", stats.TotalOrders)
	fmt.Printf("     Total revenue: $%.2f\n", stats.TotalRevenue.Dollars())
	fmt.Printf("     Average order: $%.2f\n", stats.AverageOrderValue)
	fmt.Printf("     Pending: %d, Delivered: %d\n", stats.PendingCount, stats.DeliveredCount)

	// Create more orders for user2
	fmt.Println("\n9. Creating more orders...")
	order2, err := queries.CreateOrder(ctx, db.CreateOrderParams{
		UserID:      user2.ID,
		Status:      types.StatusDelivered,
		TotalAmount: types.FromDollars(199.99),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created delivered order: ID=%v\n", order2.ID)

	// List orders by status
	fmt.Println("\n10. Listing orders by status...")
	pendingOrders, err := queries.GetOrdersByStatus(ctx, types.StatusPending)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Pending orders: %d\n", len(pendingOrders))

	deliveredOrders, err := queries.GetOrdersByStatus(ctx, types.StatusDelivered)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Delivered orders: %d\n", len(deliveredOrders))

	// Demonstrate type safety
	fmt.Println("\n=== Type Safety Demo ===")
	fmt.Printf("UserID type: %T (value: %v)\n", user1.ID, user1.ID)
	fmt.Printf("OrderID type: %T (value: %v)\n", order1.ID, order1.ID)
	fmt.Printf("ProductID type: %T (value: %v)\n", product1.ID, product1.ID)
	fmt.Printf("Status type: %T (value: %s)\n", order1.Status, order1.Status)
	fmt.Printf("Money type: %T (value: $%.2f)\n", product1.Price, product1.Price.Dollars())

	fmt.Println("\n✅ All advanced features working!")
}

const schema = `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total_amount INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);
`
