//nolint:gocritic // Example code uses log.Fatal after defer for simplicity.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	advanceddb "github.com/electwix/db-catalyst/examples/advanced/db"
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
	defer func() { _ = sqlDB.Close() }()

	// Initialize schema
	if _, err := sqlDB.ExecContext(ctx, schema); err != nil {
		_ = sqlDB.Close()
		log.Print(err)
		return
	}

	queries := advanceddb.New(sqlDB)

	fmt.Println("=== Advanced Features Demo ===")

	// Create users with strongly-typed IDs
	fmt.Println("1. Creating users with custom types...")
	user1, err := queries.CreateUser(ctx, "alice@example.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created user: ID=%v, Email=%v\n", user1.Id, user1.Email)

	user2, err := queries.CreateUser(ctx, "bob@example.com")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created user: ID=%v, Email=%v\n", user2.Id, user2.Email)

	// Create products with Money type
	fmt.Println("\n2. Creating products with Money type...")
	//nolint:mnd // Example product prices
	product1, err := queries.CreateProduct(ctx, types.SKU("LAPTOP-001"), "Gaming Laptop", types.FromDollars(1299.99))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created product: ID=%v, SKU=%v, Price=$%.2f\n",
		product1.Id, product1.Sku, types.Money(product1.Price.(int64)).Dollars())

	//nolint:mnd // Example product prices
	product2, err := queries.CreateProduct(ctx, types.SKU("MOUSE-001"), "Wireless Mouse", types.FromDollars(49.99))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created product: ID=%v, SKU=%v, Price=$%.2f\n",
		product2.Id, product2.Sku, types.Money(product2.Price.(int64)).Dollars())

	// Create order with custom types
	fmt.Println("\n3. Creating orders with Status enum...")
	order1, err := queries.CreateOrder(ctx, user1.Id, types.StatusPending, types.FromDollars(0))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created order: ID=%v, User=%v, Status=%v\n",
		order1.Id, order1.UserId, order1.Status)

	// Query with custom types
	fmt.Println("\n4. Querying with custom types...")
	fetchedOrder, err := queries.GetOrder(ctx, order1.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Fetched order: ID=%v, User=%v, Status=%v, Total=$%.2f\n",
		fetchedOrder.Id, fetchedOrder.UserId, fetchedOrder.Status,
		types.Money(fetchedOrder.TotalAmount.(int64)).Dollars())

	// Get user orders
	fmt.Println("\n5. Getting user orders...")
	userOrders, err := queries.ListOrdersByUser(ctx, user1.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   User orders: %d\n", len(userOrders))

	// Update order status
	fmt.Println("\n6. Updating order status...")
	updatedOrder, err := queries.UpdateOrderStatus(ctx, types.StatusProcessing, order1.Id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Updated order status: %v\n", updatedOrder.Status)

	// List orders by status
	fmt.Println("\n7. Listing orders by status...")
	pendingOrders, err := queries.GetOrdersByStatus(ctx, types.StatusPending)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Pending orders: %d\n", len(pendingOrders))

	// Get product by SKU
	fmt.Println("\n8. Getting product by SKU...")
	product, err := queries.GetProductBySku(ctx, types.SKU("LAPTOP-001"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found product: %v, Price=$%.2f\n",
		product.Name, types.Money(product.Price.(int64)).Dollars())

	// List all products
	fmt.Println("\n9. Listing all products...")
	products, err := queries.ListProducts(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range products {
		fmt.Printf("   - %v: $%.2f\n", p.Name, types.Money(p.Price.(int64)).Dollars())
	}

	// Demonstrate type safety
	fmt.Println("\n=== Type Safety Demo ===")
	fmt.Printf("UserID type: %T (value: %v)\n", user1.Id, user1.Id)
	fmt.Printf("OrderID type: %T (value: %v)\n", order1.Id, order1.Id)
	fmt.Printf("ProductID type: %T (value: %v)\n", product1.Id, product1.Id)
	fmt.Printf("Status type: %T (value: %v)\n", order1.Status, order1.Status)

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
