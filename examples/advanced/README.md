# Advanced Example - Custom Types & Prepared Queries

This example demonstrates db-catalyst's unique features that sqlc doesn't have.

## Features Demonstrated

- **Custom Types with Pointer Control** - Map SQLite types to custom Go types
- **Schema Transformation** - Use domain types in SQL, get Go types in code
- **Prepared Queries** - Pre-compiled statements with metrics
- **Type Safety** - Strongly typed IDs and enums

## Custom Types

```go
// Your domain types
type UserID int64
type Status string
type Money int64 // stored as cents
```

Map them in config:
```toml
[[custom_types.mapping]]
custom_type = "user_id"
sqlite_type = "INTEGER"
go_type = "github.com/electwix/db-catalyst/examples/advanced/types.UserID"
pointer = false
```

## Running

```bash
cd examples/advanced
db-catalyst
go run main.go
```

## What Makes This Special

1. **Type Safety**: Can't accidentally pass a ProductID where UserID expected
2. **Pointer Control**: Explicit `pointer = true/false` regardless of nullability
3. **Prepared Statements**: Metrics and thread-safe initialization
4. **Schema Stays Clean**: Use custom types in SQL, they transform automatically
