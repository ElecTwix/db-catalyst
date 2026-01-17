# Schema Reference

db-catalyst generates Go types from SQLite schema definitions.

## Supported Syntax

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Column Types

| SQLite Type | Go Type |
|-------------|---------|
| INTEGER | int64 |
| REAL | float64 |
| TEXT | string |
| BLOB | []byte |
| NUMERIC | float64 |

## Nullable Columns

Columns without `NOT NULL` are generated as pointers:

```sql
email TEXT          -- nullable → *string
age INTEGER         -- nullable → *int64
```

## Default Values

Default values are supported but ignored during code generation. SQLite handles defaults at runtime.

## CHECK Constraints

CHECK constraints are parsed but ignored. They don't affect generated code.

## PRIMARY KEY

- `INTEGER PRIMARY KEY` becomes the table's primary key type
- `PRIMARY KEY` on other columns doesn't change type generation

## STRICT Tables

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
) STRICT, WITHOUT ROWID;
```

STRICT tables are fully supported. Type checking happens at insert time in SQLite.

## Generating Models

Models are generated from tables referenced in queries:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;
```

Generates `models.gen.go` with the `User` struct.
