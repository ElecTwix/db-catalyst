# Schema Reference

Complete guide to writing SQL schemas for db-catalyst. This document covers supported SQL features, type mappings, constraints, and best practices.

## Table of Contents

- [Overview](#overview)
- [Supported SQL Features](#supported-sql-features)
- [Data Type Mappings](#data-type-mappings)
- [Column Constraints](#column-constraints)
- [Indexes](#indexes)
- [Foreign Keys](#foreign-keys)
- [Views](#views)
- [Triggers](#triggers)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [SQL Schema Generation](#sql-schema-generation)

## Overview

db-catalyst parses SQL schema files and generates type-safe Go code. The schema defines your database structure—tables, columns, constraints, and relationships.

### Basic Table Definition

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    bio TEXT,                    -- nullable column
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);
```

### File Organization

Organize schemas by feature or keep them in a single file:

```
schema/
├── 001_users.sql      -- migrations-style numbering
├── 002_posts.sql
└── 003_comments.sql

-- or

schema/
└── schema.sql         -- single comprehensive file
```

Reference schemas in `db-catalyst.toml`:

```toml
schemas = ["schema/*.sql"]
```

## Supported SQL Features

### CREATE TABLE

Full support for standard CREATE TABLE syntax:

```sql
CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    price REAL NOT NULL CHECK (price >= 0),
    quantity INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    metadata BLOB,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### STRICT Tables (SQLite 3.37+)

STRICT tables enforce type checking at the database level:

```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    occurred_at DATETIME NOT NULL
) STRICT, WITHOUT ROWID;
```

Benefits:
- Type safety enforced by SQLite
- Clearer intent in schema
- Better compatibility across database dialects

### IF NOT EXISTS

Safe for idempotent migrations:

```sql
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action TEXT NOT NULL,
    performed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### TEMPORARY Tables

Parsed but not used for code generation:

```sql
CREATE TEMPORARY TABLE temp_import (
    raw_data TEXT
);
```

## Data Type Mappings

### SQLite to Go Types

| SQLite Type | Go Type | Notes |
|-------------|---------|-------|
| `INTEGER` | `int64` | Primary keys, counters, flags |
| `REAL` | `float64` | Decimal numbers |
| `TEXT` | `string` | Strings, dates stored as text |
| `BLOB` | `[]byte` | Binary data |
| `NUMERIC` | `float64` | Numeric affinity |
| `BOOLEAN` | `int64` | SQLite stores as 0/1 |
| `DATETIME` | `string` | Store ISO 8601 format |

### Type Aliases

These SQLite type aliases map to the same Go types:

```sql
-- All map to int64
INTEGER, INT, TINYINT, SMALLINT, MEDIUMINT, BIGINT, UNSIGNED BIG INT
INT2, INT8

-- All map to float64  
REAL, DOUBLE, DOUBLE PRECISION, FLOAT

-- All map to string
TEXT, CHARACTER, VARCHAR, VARYING CHARACTER, NCHAR, NATIVE CHARACTER
NVARCHAR, CLOB

-- All map to []byte
BLOB, no datatype specified
```

### Nullable Types

Columns without `NOT NULL` generate pointer types:

```sql
CREATE TABLE articles (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,       -- string (required)
    subtitle TEXT,             -- *string (nullable)
    body TEXT NOT NULL,        -- string (required)
    summary TEXT,              -- *string (nullable)
    view_count INTEGER NOT NULL DEFAULT 0,  -- int64
    deleted_at DATETIME        -- *string (nullable)
);
```

Generated Go struct:

```go
type Article struct {
    ID         int64
    Title      string
    Subtitle   *string
    Body       string
    Summary    *string
    ViewCount  int64
    DeletedAt  *string
}
```

### Custom Type Mappings

Override default mappings in `db-catalyst.toml`:

```toml
[[custom_types.mapping]]
custom_type = "user_id"
sqlite_type = "INTEGER"
go_type = "github.com/example/types.UserID"
```

## Column Constraints

### PRIMARY KEY

Single column primary key:

```sql
CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,  -- auto-incrementing
    name TEXT NOT NULL
);
```

Composite primary key:

```sql
CREATE TABLE order_items (
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    PRIMARY KEY (order_id, product_id)
);
```

### UNIQUE Constraints

Column-level:

```sql
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE
);
```

Table-level (composite):

```sql
CREATE TABLE memberships (
    user_id INTEGER NOT NULL,
    organization_id INTEGER NOT NULL,
    role TEXT NOT NULL,
    UNIQUE (user_id, organization_id)  -- one membership per user/org
);
```

Named constraints:

```sql
CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    sku TEXT NOT NULL,
    CONSTRAINT uq_products_sku UNIQUE (sku)
);
```

### NOT NULL

Required columns:

```sql
CREATE TABLE comments (
    id INTEGER PRIMARY KEY,
    post_id INTEGER NOT NULL,      -- must have a post
    author_id INTEGER NOT NULL,    -- must have an author
    body TEXT NOT NULL,            -- must have content
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### DEFAULT Values

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    priority INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

Supported default expressions:
- Literal values: `DEFAULT 'active'`, `DEFAULT 0`
- `CURRENT_TIMESTAMP`, `CURRENT_DATE`, `CURRENT_TIME`
- `unixepoch()` for epoch timestamps
- `gen_random_uuid()` (PostgreSQL)

### CHECK Constraints

Parsed but not enforced in generated code (SQLite enforces at runtime):

```sql
CREATE TABLE reviews (
    id INTEGER PRIMARY KEY,
    product_id INTEGER NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title TEXT CHECK (length(title) <= 100),
    body TEXT
);
```

### COLLATE

Case-insensitive text comparison:

```sql
CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE  -- 'Tag' == 'tag'
);
```

## Indexes

### Basic Indexes

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_posts_author ON posts(author_id);
```

### Composite Indexes

```sql
CREATE INDEX idx_posts_author_status ON posts(author_id, status);
```

### Unique Indexes

```sql
CREATE UNIQUE INDEX idx_accounts_lower_email ON accounts(lower(email));
```

### Partial Indexes (SQLite 3.8.0+)

```sql
CREATE INDEX idx_active_users ON users(created_at) WHERE is_active = 1;
```

### IF NOT EXISTS

```sql
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
```

### Index Best Practices

1. **Index foreign key columns**: Always index columns used in JOINs
2. **Index filter columns**: Index columns used in WHERE clauses
3. **Consider composite indexes**: For multi-column filters
4. **Don't over-index**: Each index adds write overhead

```sql
-- Good: indexes for common queries
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    author_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    published_at DATETIME,
    FOREIGN KEY (author_id) REFERENCES authors(id)
);

CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_published ON posts(published_at) WHERE published_at IS NOT NULL;
```

## Foreign Keys

### Basic Foreign Keys

```sql
CREATE TABLE comments (
    id INTEGER PRIMARY KEY,
    post_id INTEGER NOT NULL,
    body TEXT NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

### With ON DELETE/UPDATE Actions

```sql
CREATE TABLE order_items (
    id INTEGER PRIMARY KEY,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
);
```

Available actions:
- `CASCADE`: Delete/update related rows
- `RESTRICT`: Prevent delete/update if references exist
- `SET NULL`: Set foreign key to NULL
- `SET DEFAULT`: Set to default value
- `NO ACTION`: Similar to RESTRICT (deferred checking)

### Composite Foreign Keys

```sql
CREATE TABLE schedule (
    employee_id INTEGER NOT NULL,
    department_id INTEGER NOT NULL,
    shift_date DATE NOT NULL,
    PRIMARY KEY (employee_id, department_id, shift_date),
    FOREIGN KEY (employee_id, department_id) 
        REFERENCES employees(id, department_id)
);
```

### Self-Referencing

```sql
CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id INTEGER,
    FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE
);
```

## Views

Views are parsed but don't generate models unless referenced in queries:

```sql
CREATE VIEW active_users AS
SELECT id, email, name, created_at
FROM users
WHERE is_active = 1;
```

Query the view:

```sql
-- name: ListActiveUsers :many
SELECT * FROM active_users ORDER BY created_at DESC;
```

Complex view with JOINs:

```sql
CREATE VIEW post_summary AS
SELECT 
    p.id,
    p.title,
    a.name as author_name,
    COUNT(c.id) as comment_count
FROM posts p
JOIN authors a ON p.author_id = a.id
LEFT JOIN comments c ON c.post_id = p.id
GROUP BY p.id, p.title, a.name;
```

## Triggers

Triggers are parsed but don't affect code generation:

```sql
CREATE TRIGGER update_tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;
```

## Best Practices

### 1. Use Descriptive Names

```sql
-- Good
CREATE TABLE user_profiles (
    id INTEGER PRIMARY KEY,
    display_name TEXT NOT NULL,
    avatar_url TEXT
);

-- Avoid
CREATE TABLE up (
    id INTEGER PRIMARY KEY,
    dn TEXT NOT NULL,
    au TEXT
);
```

### 2. Always Define Primary Keys

```sql
-- Good
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,  -- UUID or random string
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL
);

-- Avoid (unless specific use case)
CREATE TABLE logs (
    message TEXT,
    created_at DATETIME
);
```

### 3. Use NOT NULL Liberally

```sql
-- Good - clear what's required
CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total_cents INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shipped_at DATETIME  -- nullable, not all orders ship
);
```

### 4. Index Foreign Keys

```sql
CREATE TABLE comments (
    id INTEGER PRIMARY KEY,
    post_id INTEGER NOT NULL,
    author_id INTEGER NOT NULL,
    body TEXT NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);

CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_author ON comments(author_id);
```

### 5. Use Consistent Naming Conventions

```sql
-- Tables: plural, snake_case
CREATE TABLE user_sessions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    -- Columns: snake_case
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);

-- Indexes: idx_<table>_<column>
CREATE INDEX idx_user_sessions_user ON user_sessions(user_id);
```

### 6. Document Your Schema

```sql
-- User accounts table
-- Stores authentication and profile information
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- Unique email for login
    email TEXT NOT NULL UNIQUE,
    -- Display name shown in UI
    display_name TEXT NOT NULL,
    -- Optional bio for profile page
    bio TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Examples

### Blog Application

```sql
-- Authors/Users
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    bio TEXT,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- Blog posts
CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    excerpt TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    published_at INTEGER,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER,
    FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE
);

-- Categories
CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    description TEXT
);

-- Many-to-many: posts <-> categories
CREATE TABLE post_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, category_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- Comments
CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    content TEXT NOT NULL,
    is_approved INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_published ON posts(published_at) WHERE published_at IS NOT NULL;
CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_approved ON comments(is_approved);
```

### E-Commerce Application

```sql
-- Products
CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Customers
CREATE TABLE customers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    phone TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Orders
CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    total_cents INTEGER NOT NULL DEFAULT 0,
    shipping_address TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- Order items
CREATE TABLE order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Indexes
CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_items_order ON order_items(order_id);
```

## SQL Schema Generation

db-catalyst can generate normalized SQL schema for deployment to different databases.

### Configuration

```toml
[generation]
sql_dialect = "sqlite"   # sqlite, mysql, or postgres
```

Or via CLI:

```bash
db-catalyst --sql-dialect postgres
```

### Supported Dialects

| Dialect | Identifier | Output |
|---------|------------|--------|
| SQLite | `sqlite` | `CREATE TABLE IF NOT EXISTS ...` |
| MySQL | `mysql` | `DROP TABLE IF EXISTS ...` with `ENGINE=InnoDB` |
| PostgreSQL | `postgres` | `DROP TABLE IF EXISTS ...` with `GENERATED ALWAYS AS IDENTITY` |

### Type Mapping by Dialect

#### SQLite Types

| Input Type | Output Type |
|------------|-------------|
| INTEGER, INT, BIGINT | INTEGER |
| REAL, FLOAT, DOUBLE | REAL |
| TEXT, VARCHAR, CHAR | TEXT |
| BLOB | BLOB |
| BOOLEAN | INTEGER |

#### MySQL Types

| Input Type | Output Type |
|------------|-------------|
| INTEGER, INT, BIGINT | BIGINT |
| REAL, FLOAT, DOUBLE | DOUBLE |
| TEXT, VARCHAR, CHAR | TEXT (preserves length) |
| BLOB | BLOB |
| BOOLEAN | TINYINT(1) |
| DATETIME, TIMESTAMP | DATETIME |
| JSON | JSON |

#### PostgreSQL Types

| Input Type | Output Type |
|------------|-------------|
| INTEGER, INT | INTEGER |
| BIGINT | BIGINT |
| REAL, FLOAT4 | REAL |
| DOUBLE, FLOAT8 | DOUBLE PRECISION |
| DECIMAL, NUMERIC | NUMERIC |
| TEXT, VARCHAR | TEXT (preserves length) |
| BLOB | BYTEA |
| BOOLEAN | BOOLEAN |
| DATETIME, TIMESTAMP | TIMESTAMP |
| DATE | DATE |
| TIME | TIME |
| JSON | JSON |
| JSONB | JSONB |
| UUID | UUID |
| SERIAL | SERIAL |
| BIGSERIAL | BIGSERIAL |

### CLI Options

```bash
--sql-dialect string   Generate SQL schema output (sqlite, mysql, postgres)
--sql-output           Enable SQL schema generation
--if-not-exists        Use IF NOT EXISTS in SQL output (default true)
```

---

## Troubleshooting

### Common Issues

**Issue**: `table not found` errors in generated code
- **Solution**: Ensure table is referenced in at least one query. db-catalyst only generates models for tables used in queries.

**Issue**: Column types not matching expectations
- **Solution**: Check type affinity rules. SQLite is flexible with types; db-catalyst maps based on declared type.

**Issue**: Foreign key constraints not enforced
- **Solution**: Enable foreign keys in SQLite: `PRAGMA foreign_keys = ON;`

**Issue**: Views not generating models
- **Solution**: Views only generate models when referenced in queries with `-- name:` annotations.

---

For more information, see:
- [Query Reference](query.md) - Writing SQL queries
- [Generated Code Reference](generated-code.md) - Understanding generated code
- [Feature Flags](feature-flags.md) - Configuration options
