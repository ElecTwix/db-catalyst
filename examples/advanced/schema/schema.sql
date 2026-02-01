-- Schema using custom type names
-- These get transformed to standard SQLite types during processing

CREATE TABLE users (
    id user_id PRIMARY KEY AUTOINCREMENT,
    email email NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE products (
    id product_id PRIMARY KEY AUTOINCREMENT,
    sku sku NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price money NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE orders (
    id order_id PRIMARY KEY AUTOINCREMENT,
    user_id user_id NOT NULL,
    status status NOT NULL DEFAULT 'pending',
    total_amount money NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id order_id NOT NULL,
    product_id product_id NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price money NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);
