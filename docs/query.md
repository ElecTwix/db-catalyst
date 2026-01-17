# Query Reference

Query files define SQL queries with metadata for Go code generation.

## Syntax

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;
```

Each query requires:
- `-- name: <MethodName>` - Go method name
- `:one`, `:many`, `:exec`, `:execresult` - Return type

## Return Types

| Suffix | Return Type | Description |
|--------|-------------|-------------|
| `:one` | `(Row, error)` | Single row or `sql.ErrNoRows` |
| `:many` | `([]Row, error)` | Zero or more rows |
| `:exec` | `(sql.Result, error)` | INSERT/UPDATE/DELETE |
| `:execresult` | `(QueryResult, error)` | With LastInsertID/RowsAffected |

## Parameters

Positional parameters `?` are inferred from context:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;  -- infers id:int64

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES (?, ?);
-- infers name:string, email:string
```

## Named Parameters

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = :id;  -- :id parameter
```

## IN Clauses with Variadic Parameters

```sql
-- name: ListUsersByIDs :many
SELECT * FROM users WHERE id IN (?1, ?2, ?3);
-- variadic count = 3

-- Dynamic IN clause
-- name: ListUsersByIDs :many
SELECT * FROM users WHERE id IN (?1...);
-- any number of arguments
```

## Comments as Documentation

```sql
-- GetUser fetches a single user by identifier.
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;
```

## Common Table Expressions (CTEs)

```sql
-- name: GetReportData :many
WITH RECURSIVE report_tree AS (
    SELECT id, parent_id, name FROM items WHERE report_id = ?
    UNION ALL
    SELECT i.id, i.parent_id, i.name FROM items i
    JOIN report_tree r ON i.parent_id = r.id
)
SELECT * FROM report_tree;
```

## RETURNING Clause

```sql
-- name: CreateUser :one
INSERT INTO users (name, email) VALUES (?, ?)
RETURNING id, created_at;
```

## Star Expansion

```sql
SELECT * FROM users;  -- expands to column list
SELECT u.* FROM users u;  -- table-qualified star
```

## Ignored Syntax

The following are parsed but don't affect code generation:
- CHECK constraints
- COLLATE clauses
- Complex DEFAULT expressions
- Index definitions in same file
