-- Schema using custom type names
-- These get transformed to standard SQLite types during processing

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    TEXT TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    TEXT TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    price INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    INTEGER INTEGER NOT NULL,
    TEXT TEXT NOT NULL DEFAULT 'pending',
    total_amount INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER,
    FOREIGN KEY (INTEGER) REFERENCES users(id)
);

CREATE TABLE order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    INTEGER INTEGER NOT NULL,
    INTEGER INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL,
    FOREIGN KEY (INTEGER) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (INTEGER) REFERENCES products(id)
);
