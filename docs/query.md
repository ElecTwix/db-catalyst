# Query Reference

Complete guide to writing SQL queries with db-catalyst annotations. Learn how to define queries, use parameters, handle return types, and write complex queries.

## Table of Contents

- [Overview](#overview)
- [Query Annotations](#query-annotations)
- [Return Types](#return-types)
- [Parameters](#parameters)
- [SELECT Queries](#select-queries)
- [INSERT Queries](#insert-queries)
- [UPDATE Queries](#update-queries)
- [DELETE Queries](#delete-queries)
- [JOINs and Relationships](#joins-and-relationships)
- [Common Table Expressions (CTEs)](#common-table-expressions-ctes)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Examples by Use Case](#examples-by-use-case)
- [Troubleshooting](#troubleshooting)

## Overview

Query files contain SQL statements with special annotations that tell db-catalyst how to generate Go code. Each query needs:

1. **A name**: The Go method name
2. **A return type**: How many rows and what to return
3. **The SQL**: The actual query

### Basic Query Structure

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;
```

This generates:

```go
func (q *Queries) GetUser(ctx context.Context, id int64) (User, error)
```

### File Organization

```
queries/
├── users.sql       -- User-related queries
├── posts.sql       -- Post-related queries
└── analytics.sql   -- Reporting queries
```

Reference in `db-catalyst.toml`:

```toml
queries = ["queries/*.sql"]
```

## Query Annotations

### Syntax

```sql
-- name: <MethodName> :<ReturnType>
<SQL statement>;
```

### Method Naming

Use descriptive, action-oriented names:

```sql
-- Good
-- name: GetUserByID :one
-- name: ListActiveUsers :many
-- name: CreateUser :one
-- name: UpdateUserEmail :exec
-- name: DeleteExpiredSessions :exec

-- Avoid
-- name: User :one
-- name: Select :many
-- name: DoIt :exec
```

### Documentation Comments

Add documentation above the annotation:

```sql
-- GetUserByID retrieves a single user by their unique identifier.
-- Returns sql.ErrNoRows if the user doesn't exist.
-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;
```

Generated code includes the comment:

```go
// GetUserByID retrieves a single user by their unique identifier.
// Returns sql.ErrNoRows if the user doesn't exist.
func (q *Queries) GetUserByID(ctx context.Context, id int64) (User, error)
```

## Return Types

### :one - Single Row

Returns a single struct or `sql.ErrNoRows`:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserEmail :one
SELECT email FROM users WHERE id = ?;
```

Generated signatures:

```go
func (q *Queries) GetUser(ctx context.Context, id int64) (User, error)
func (q *Queries) GetUserEmail(ctx context.Context, id int64) (string, error)
```

### :many - Multiple Rows

Returns a slice of structs:

```sql
-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: ListActiveUsers :many
SELECT * FROM users WHERE is_active = 1 ORDER BY name;
```

Generated signatures:

```go
func (q *Queries) ListUsers(ctx context.Context) ([]User, error)
func (q *Queries) ListActiveUsers(ctx context.Context) ([]User, error)
```

### :exec - Execute Only

For INSERT, UPDATE, DELETE without RETURNING:

```sql
-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: UpdateUserStatus :exec
UPDATE users SET is_active = ? WHERE id = ?;
```

Generated signatures:

```go
func (q *Queries) DeleteUser(ctx context.Context, id int64) error
func (q *Queries) UpdateUserStatus(ctx context.Context, isActive int64, id int64) error
```

### :execresult - Execution with Result

Returns `sql.Result` for LastInsertId/RowsAffected:

```sql
-- name: CreateUser :execresult
INSERT INTO users (email, name) VALUES (?, ?);
```

Generated signature:

```go
func (q *Queries) CreateUser(ctx context.Context, email string, name string) (sql.Result, error)
```

Usage:

```go
result, err := queries.CreateUser(ctx, "alice@example.com", "Alice")
if err != nil {
    return err
}
lastID, _ := result.LastInsertId()
```

### Return Type Summary

| Suffix | Return Type | Use For |
|--------|-------------|---------|
| `:one` | `(T, error)` | Single row lookups |
| `:many` | `([]T, error)` | Lists, searches |
| `:exec` | `error` | Updates, deletes without RETURNING |
| `:execresult` | `(sql.Result, error)` | Inserts needing LastInsertId |

## Parameters

### Positional Parameters (?)

SQLite-style positional parameters:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: CreateUser :one
INSERT INTO users (email, name) VALUES (?, ?);
```

Parameter types are inferred from column types.

### Numbered Parameters (?1, ?2, etc.)

Reuse parameters by position:

```sql
-- name: UpdateUser :one
UPDATE users 
SET name = ?2, email = ?3 
WHERE id = ?1
RETURNING *;
-- ?1 = id, ?2 = name, ?3 = email
```

### PostgreSQL Parameters ($1, $2)

For PostgreSQL databases:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, name) VALUES ($1, $2) RETURNING *;
```

### Named Parameters

Use descriptive parameter names (parsed but converted to positional):

```sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = :email;

-- name: CreateUser :one
INSERT INTO users (email, name) VALUES (:email, :name);
```

### Parameter Type Inference

db-catalyst infers types from the context:

```sql
-- Infers id:int64 from users.id column
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- Infers email:string from users.email column
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;
```

### Multiple Parameters

```sql
-- name: SearchUsers :many
SELECT * FROM users 
WHERE is_active = ? 
  AND created_at > ?
ORDER BY name
LIMIT ? OFFSET ?;
-- Parameters: is_active (int64), created_at (string), limit (int64), offset (int64)
```

## SELECT Queries

### Basic SELECT

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
```

### Column Selection

Select specific columns for custom result types:

```sql
-- name: GetUserSummary :one
SELECT id, email, name FROM users WHERE id = ?;

-- name: GetUserStats :one
SELECT 
    u.id,
    u.email,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON p.author_id = u.id
WHERE u.id = ?
GROUP BY u.id, u.email;
```

Generated struct for custom columns:

```go
type GetUserStatsRow struct {
    ID        int64
    Email     string
    PostCount int64
}
```

### Star Expansion

`SELECT *` expands to actual column names:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;
-- Generates: SELECT id, email, name, created_at FROM users WHERE id = ?
```

Table-qualified stars:

```sql
-- name: GetUserWithPosts :many
SELECT u.*, p.id as post_id, p.title
FROM users u
LEFT JOIN posts p ON p.author_id = u.id
WHERE u.id = ?;
```

### WHERE Clauses

```sql
-- Equality
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- Multiple conditions
-- name: SearchUsers :many
SELECT * FROM users 
WHERE is_active = ? AND created_at > ?;

-- IN clause (fixed count)
-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id IN (?1, ?2, ?3);

-- LIKE pattern matching
-- name: SearchUsersByName :many
SELECT * FROM users WHERE name LIKE '%' || ? || '%';
```

### ORDER BY

```sql
-- name: ListUsersByName :many
SELECT * FROM users ORDER BY name ASC;

-- name: ListRecentUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- Multiple columns
-- name: ListUsers :many
SELECT * FROM users ORDER BY is_active DESC, name ASC;
```

### LIMIT and OFFSET

```sql
-- name: ListUsersPaged :many
SELECT * FROM users ORDER BY id LIMIT ? OFFSET ?;
```

Usage:

```go
pageSize := 20
page := 1
users, err := queries.ListUsersPaged(ctx, pageSize, (page-1)*pageSize)
```

### Aggregation

```sql
-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = 1;

-- name: GetUserStats :one
SELECT 
    COUNT(*) as total_users,
    SUM(CASE WHEN is_active = 1 THEN 1 ELSE 0 END) as active_users
FROM users;
```

## INSERT Queries

### Basic INSERT

```sql
-- name: CreateUser :exec
INSERT INTO users (email, name) VALUES (?, ?);
```

### INSERT with RETURNING

```sql
-- name: CreateUser :one
INSERT INTO users (email, name) VALUES (?, ?) RETURNING *;

-- name: CreateUserWithID :one
INSERT INTO users (email, name) VALUES (?, ?) RETURNING id;
```

### INSERT OR IGNORE

```sql
-- name: CreateUserIfNotExists :exec
INSERT OR IGNORE INTO users (email, name) VALUES (?, ?);
```

### ON CONFLICT

```sql
-- name: UpsertUser :one
INSERT INTO users (email, name) VALUES (?, ?)
ON CONFLICT(email) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: CreateTagIfNotExists :exec
INSERT INTO tags (name) VALUES (?)
ON CONFLICT DO NOTHING;
```

### Batch Insert

```sql
-- Insert multiple rows (fixed count)
-- name: CreateTwoUsers :exec
INSERT INTO users (email, name) VALUES 
    (?1, ?2),
    (?3, ?4);
```

For dynamic batch inserts, use a loop in Go or `:many` with a temporary table approach.

## UPDATE Queries

### Basic UPDATE

```sql
-- name: UpdateUser :exec
UPDATE users SET name = ?, email = ? WHERE id = ?;
```

### UPDATE with RETURNING

```sql
-- name: UpdateUser :one
UPDATE users SET name = ?, email = ? WHERE id = ? RETURNING *;
```

### Conditional UPDATE

```sql
-- name: UpdateUserStatus :exec
UPDATE users SET is_active = ? WHERE id = ?;

-- name: IncrementViewCount :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = ?;
```

### Partial Updates

Update only provided fields using COALESCE:

```sql
-- name: PatchUser :one
UPDATE users 
SET 
    name = COALESCE(?2, name),
    email = COALESCE(?3, email)
WHERE id = ?1
RETURNING *;
```

## DELETE Queries

### Basic DELETE

```sql
-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;
```

### DELETE with RETURNING (PostgreSQL)

```sql
-- name: DeleteUser :one
DELETE FROM users WHERE id = $1 RETURNING *;
```

### Bulk DELETE

```sql
-- name: DeleteInactiveUsers :exec
DELETE FROM users WHERE is_active = 0 AND last_login_at < ?;
```

## JOINs and Relationships

### INNER JOIN

```sql
-- name: GetPostWithAuthor :one
SELECT 
    p.id,
    p.title,
    p.content,
    u.id as author_id,
    u.name as author_name
FROM posts p
JOIN users u ON p.author_id = u.id
WHERE p.id = ?;
```

### LEFT JOIN

```sql
-- name: GetUserWithPostCount :one
SELECT 
    u.id,
    u.name,
    COUNT(p.id) as post_count
FROM users u
LEFT JOIN posts p ON p.author_id = u.id
WHERE u.id = ?
GROUP BY u.id, u.name;
```

### Multiple JOINs

```sql
-- name: GetCommentWithDetails :one
SELECT 
    c.id,
    c.content,
    c.created_at,
    u.name as author_name,
    p.title as post_title
FROM comments c
JOIN users u ON c.author_id = u.id
JOIN posts p ON c.post_id = p.id
WHERE c.id = ?;
```

### Self JOIN

```sql
-- name: GetSubordinates :many
SELECT e.* FROM employees e
JOIN employees m ON e.manager_id = m.id
WHERE m.id = ?;
```

## Common Table Expressions (CTEs)

### Basic CTE

```sql
-- name: GetActiveUsersWithPostCount :many
WITH user_posts AS (
    SELECT author_id, COUNT(*) as post_count
    FROM posts
    GROUP BY author_id
)
SELECT 
    u.id,
    u.name,
    COALESCE(up.post_count, 0) as post_count
FROM users u
LEFT JOIN user_posts up ON up.author_id = u.id
WHERE u.is_active = 1;
```

### Recursive CTE

```sql
-- name: GetCategoryTree :many
WITH RECURSIVE category_tree AS (
    -- Base case: start with the given category
    SELECT id, name, parent_id, 0 as depth
    FROM categories
    WHERE id = ?
    
    UNION ALL
    
    -- Recursive case: get children
    SELECT c.id, c.name, c.parent_id, ct.depth + 1
    FROM categories c
    JOIN category_tree ct ON c.parent_id = ct.id
)
SELECT * FROM category_tree ORDER BY depth, name;
```

### Multiple CTEs

```sql
-- name: GetDashboardStats :one
WITH 
user_stats AS (
    SELECT COUNT(*) as total_users FROM users
),
post_stats AS (
    SELECT COUNT(*) as total_posts FROM posts
),
comment_stats AS (
    SELECT COUNT(*) as total_comments FROM comments
)
SELECT 
    u.total_users,
    p.total_posts,
    c.total_comments
FROM user_stats u, post_stats p, comment_stats c;
```

## Advanced Features

### Subqueries

```sql
-- Scalar subquery
-- name: GetUserWithPostCount :one
SELECT 
    u.*,
    (SELECT COUNT(*) FROM posts WHERE author_id = u.id) as post_count
FROM users u
WHERE u.id = ?;

-- IN subquery
-- name: GetUsersWithPosts :many
SELECT * FROM users 
WHERE id IN (SELECT DISTINCT author_id FROM posts);

-- EXISTS subquery
-- name: GetUsersWithoutPosts :many
SELECT * FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM posts WHERE author_id = u.id
);
```

### Window Functions

```sql
-- name: GetUserRankings :many
SELECT 
    id,
    name,
    score,
    RANK() OVER (ORDER BY score DESC) as rank,
    ROW_NUMBER() OVER (ORDER BY score DESC) as row_num
FROM users
ORDER BY score DESC
LIMIT 100;
```

### CASE Expressions

```sql
-- name: GetUserStatusSummary :many
SELECT 
    id,
    name,
    CASE 
        WHEN last_login_at > datetime('now', '-7 days') THEN 'active'
        WHEN last_login_at > datetime('now', '-30 days') THEN 'recent'
        ELSE 'inactive'
    END as activity_status
FROM users;
```

### UNION and UNION ALL

```sql
-- name: SearchContent :many
SELECT 'post' as type, id, title as content, created_at
FROM posts
WHERE title LIKE '%' || ? || '%'

UNION ALL

SELECT 'comment' as type, id, content, created_at
FROM comments
WHERE content LIKE '%' || ? || '%'

ORDER BY created_at DESC
LIMIT 50;
```

### PostgreSQL-Specific Features

```sql
-- Array operations
-- name: SearchUsersByTag :many
SELECT * FROM users WHERE $1 = ANY(tags);

-- JSONB operations
-- name: SearchUsersByMetadata :many
SELECT * FROM users WHERE metadata @> $1;

-- Text search
-- name: SearchPosts :many
SELECT * FROM posts 
WHERE to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1);
```

## Best Practices

### 1. Use Descriptive Names

```sql
-- Good
-- name: GetUserByEmail :one
-- name: ListActiveUsersByDate :many
-- name: SoftDeleteUser :exec

-- Avoid
-- name: User :one
-- name: Get :many
-- name: Delete :exec
```

### 2. Document Your Queries

```sql
-- GetUserByEmail retrieves a user by their email address.
-- Returns sql.ErrNoRows if no user exists with the given email.
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;
```

### 3. Use RETURNING for Created Data

```sql
-- Good - returns the created record
-- name: CreateUser :one
INSERT INTO users (email, name) VALUES (?, ?) RETURNING *;

-- Avoid - requires separate query to get the record
-- name: CreateUser :execresult
INSERT INTO users (email, name) VALUES (?, ?);
```

### 4. Be Explicit with Column Selection

```sql
-- Good - clear what you're selecting
-- name: GetUserSummary :one
SELECT id, email, name FROM users WHERE id = ?;

-- Avoid - * can cause issues if schema changes
-- name: GetUserSummary :one
SELECT * FROM users WHERE id = ?;
```

### 5. Handle NULLs Explicitly

```sql
-- name: GetUserWithOptionalBio :one
SELECT 
    id,
    email,
    COALESCE(bio, '') as bio
FROM users
WHERE id = ?;
```

### 6. Use Transactions for Multi-Step Operations

```go
// In your application code
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

q := queries.WithTx(tx)
user, err := q.CreateUser(ctx, "alice@example.com", "Alice")
if err != nil {
    return err
}

_, err = q.CreateUserProfile(ctx, user.ID)
if err != nil {
    return err
}

return tx.Commit()
```

## Examples by Use Case

### User Management

```sql
-- name: CreateUser :one
INSERT INTO users (email, name, password_hash) 
VALUES (?, ?, ?) 
RETURNING id, email, name, created_at;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: UpdateUser :one
UPDATE users 
SET name = ?, email = ?
WHERE id = ?
RETURNING *;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: SearchUsers :many
SELECT * FROM users 
WHERE name LIKE '%' || ? || '%' OR email LIKE '%' || ? || '%'
ORDER BY name
LIMIT ?;
```

### Blog/Content Management

```sql
-- name: CreatePost :one
INSERT INTO posts (author_id, title, slug, content, status)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPostBySlug :one
SELECT * FROM posts WHERE slug = ?;

-- name: GetPublishedPosts :many
SELECT * FROM posts 
WHERE status = 'published' 
ORDER BY published_at DESC
LIMIT ? OFFSET ?;

-- name: GetPostsByAuthor :many
SELECT * FROM posts WHERE author_id = ? ORDER BY created_at DESC;

-- name: UpdatePost :one
UPDATE posts 
SET title = ?, content = ?, status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: PublishPost :one
UPDATE posts 
SET status = 'published', published_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = ?;

-- name: GetPostWithAuthor :one
SELECT 
    p.*,
    u.id as author_id,
    u.name as author_name,
    u.email as author_email
FROM posts p
JOIN users u ON p.author_id = u.id
WHERE p.id = ?;
```

### E-Commerce

```sql
-- name: CreateOrder :one
INSERT INTO orders (customer_id, status, total_cents)
VALUES (?, 'pending', ?)
RETURNING *;

-- name: AddOrderItem :one
INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetOrderWithItems :one
SELECT 
    o.*,
    oi.id as item_id,
    oi.product_id,
    oi.quantity,
    oi.unit_price_cents
FROM orders o
LEFT JOIN order_items oi ON oi.order_id = o.id
WHERE o.id = ?;

-- name: GetCustomerOrders :many
SELECT * FROM orders 
WHERE customer_id = ? 
ORDER BY created_at DESC;

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = ? WHERE id = ?;

-- name: GetProductInventory :one
SELECT id, sku, name, stock_quantity FROM products WHERE id = ?;

-- name: UpdateInventory :exec
UPDATE products SET stock_quantity = stock_quantity - ? WHERE id = ?;
```

### Analytics and Reporting

```sql
-- name: GetDailySignups :many
SELECT 
    date(created_at) as date,
    COUNT(*) as count
FROM users
WHERE created_at >= ?
GROUP BY date(created_at)
ORDER BY date;

-- name: GetTopPosts :many
SELECT 
    p.id,
    p.title,
    COUNT(c.id) as comment_count,
    p.view_count
FROM posts p
LEFT JOIN comments c ON c.post_id = p.id
WHERE p.status = 'published'
GROUP BY p.id, p.title, p.view_count
ORDER BY p.view_count DESC
LIMIT ?;

-- name: GetUserActivityStats :one
SELECT 
    u.id,
    u.name,
    COUNT(DISTINCT p.id) as posts_created,
    COUNT(DISTINCT c.id) as comments_made,
    MAX(p.created_at) as last_post_at
FROM users u
LEFT JOIN posts p ON p.author_id = u.id
LEFT JOIN comments c ON c.author_id = u.id
WHERE u.id = ?
GROUP BY u.id, u.name;
```

## Troubleshooting

### Common Issues

**Issue**: `query references unknown table`
- **Solution**: Ensure the table is defined in your schema files and the schema is included in `db-catalyst.toml`.

**Issue**: `column not found` in generated code
- **Solution**: Check that column names in queries match the schema exactly (case-sensitive in some contexts).

**Issue**: Parameter types are wrong
- **Solution**: db-catalyst infers types from column definitions. Ensure your schema has proper types.

**Issue**: `RETURNING *` not working
- **Solution**: Use `:one` return type with RETURNING. `:exec` ignores RETURNING clauses.

**Issue**: JOIN queries returning wrong types
- **Solution**: Use explicit column aliases for ambiguous columns:
  ```sql
  SELECT u.id as user_id, p.id as post_id ...
  ```

**Issue**: Query is too complex to parse
- **Solution**: Simplify the query or break it into multiple queries. Some advanced SQL features may not be supported.

---

For more information, see:
- [Schema Reference](schema.md) - Defining database schemas
- [Generated Code Reference](generated-code.md) - Understanding generated code
- [Feature Flags](feature-flags.md) - Configuration options
