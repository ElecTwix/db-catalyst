# Migrating from sqlc to db-catalyst

This guide helps you migrate your existing sqlc projects to db-catalyst, a SQLite-focused code generator with enhanced custom types support and modern Go 1.25+ features.

## Quick Comparison

| Feature | sqlc | db-catalyst |
|---------|-----|-------------|
| **Database Support** | PostgreSQL, MySQL, SQLite | **SQLite only** (optimized) |
| **Custom Types** | Limited | **Advanced** with pointer control |
| **Go Version** | Go 1.19+ | **Go 1.25+** (latest features) |
| **Configuration** | YAML | **TOML** (simpler) |
| **Generated Code** | Interface-heavy | **Clean**, hand-written style |
| **Prepared Queries** | Basic | **Advanced** with metrics |
| **Schema Transformation** | No | **Yes** (custom types) |

## Migration Steps

### 1. Install db-catalyst

```bash
go install github.com/electwix/db-catalyst/cmd/db-catalyst@latest
```

### 2. Convert Configuration

#### sqlc configuration (`sqlc.yaml`):
```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "query/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "db"
        sql_package: "database/sql"
```

#### db-catalyst configuration (`db-catalyst.toml`):
```toml
package = "db"
out = "db"
sqlite_driver = "modernc"  # or "mattn"
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]
```

### 3. Update Custom Types

#### sqlc custom types:
```yaml
# sqlc.yaml
overrides:
  - go_type: "github.com/company/types.ID"
    db_type: "INTEGER"
    column: "users.id"
  - go_type: "github.com/company/types.Status"
    db_type: "TEXT"
    nullable: true
```

#### db-catalyst custom types:
```toml
# db-catalyst.toml
[[custom_types.mapping]]
custom_type = "ID"
sqlite_type = "INTEGER"
go_type = "github.com/company/types.ID"
pointer = false

[[custom_types.mapping]]
custom_type = "Status"
sqlite_type = "TEXT"
go_type = "github.com/company/types.Status"
pointer = true
```

**Key Differences:**
- db-catalyst custom types are **column-agnostic** (apply to all columns of the SQLite type)
- **Pointer control**: Explicit `pointer = true/false` instead of relying on nullability
- **Cleaner syntax**: Array-based configuration in TOML

### 4. Schema and Query Files

Your existing SQL files work without changes! db-catalyst uses the same:

- **Schema files**: `CREATE TABLE` statements
- **Query files**: Named queries with `-- name: QueryName :one/:many/:exec`

#### Example Schema (`schema/users.sql`):
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    status TEXT,
    created_at INTEGER NOT NULL
);
```

#### Example Queries (`queries/users.sql`):
```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email, status, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;
```

### 5. Update Generated Code Usage

#### sqlc generated usage:
```go
import "github.com/yourproject/db"

func handleUser(db *sql.DB) error {
    queries := db.New(db)
    
    user, err := queries.GetUser(context.Background(), 123)
    if err != nil {
        return err
    }
    
    users, err := queries.ListUsers(context.Background())
    if err != nil {
        return err
    }
    
    return nil
}
```

#### db-catalyst generated usage:
```go
import "github.com/yourproject/db"

func handleUser(db *sql.DB) error {
    queries := db.New(db)
    
    user, err := queries.GetUser(context.Background(), 123)
    if err != nil {
        return err
    }
    
    users, err := queries.ListUsers(context.Background())
    if err != nil {
        return err
    }
    
    return nil
}
```

**The API is identical!** 🎉

## Advanced Features

### Custom Types with Pointer Control

db-catalyst offers superior custom types with explicit pointer control:

```toml
[[custom_types.mapping]]
custom_type = "UserID"
sqlite_type = "INTEGER"
go_type = "github.com/company/types.UserID"
pointer = false  # Always value type, even if nullable

[[custom_types.mapping]]
custom_type = "OptionalStatus"
sqlite_type = "TEXT"
go_type = "github.com/company/types.Status"
pointer = true   # Always pointer type, regardless of nullability
```

#### Generated Results:
```go
type User struct {
    ID          UserID          // value type (pointer=false)
    Status      *Status         // pointer type (pointer=true)
    Email       *string         // standard nullable string
}
```

### Schema Transformation

db-catalyst can transform your schema to use custom types:

```sql
-- Original schema
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    status TEXT
);

-- Transformed schema (generated)
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    status TEXT
);
```

The transformation happens internally - your original schema stays unchanged, but the generated code uses custom types.

### Prepared Queries

Enable prepared query generation with metrics:

```toml
[prepared_queries]
enabled = true
metrics = true
thread_safe = true
```

This generates additional prepared query methods with performance tracking.

## Migration Checklist

- [ ] Install db-catalyst CLI
- [ ] Convert `sqlc.yaml` to `db-catalyst.toml`
- [ ] Update custom types configuration
- [ ] Verify schema and query files are compatible
- [ ] Test generated code compilation
- [ ] Run integration tests
- [ ] Update CI/CD pipeline
- [ ] Update documentation

## Common Migration Issues

### Issue: Custom types not applied to query results

**sqlc behavior**: Custom types apply only to specific columns
**db-catalyst solution**: Custom types apply to all columns of the SQLite type

```toml
# Instead of column-specific overrides
[[custom_types.mapping]]
custom_type = "ID"
sqlite_type = "INTEGER"  # Applies to ALL INTEGER columns
go_type = "myproject.ID"
pointer = false
```

### Issue: Generated code has different import paths

**Solution**: Update your import statements to match the new package structure:

```go
// Before
import "github.com/yourproject/db/sqlc"

// After  
import "github.com/yourproject/db"
```

### Issue: Missing sqlc-specific features

db-catalyst focuses on SQLite and doesn't support:
- Multiple database engines
- Complex column-specific overrides
- sqlc's emit features

**Solution**: Use db-catalyst's enhanced custom types and schema transformation instead.

## Performance Benefits

db-catalyst generates cleaner, more efficient code:

- **No interface{} usage**: All types are concrete
- **Optimized for SQLite**: Tailored type mappings
- **Modern Go patterns**: Uses Go 1.25+ features
- **Less generated code**: Focused on SQLite use cases

## Getting Help

- **Documentation**: [db-catalyst-spec.md](../db-catalyst-spec.md)
- **Examples**: Check the `cmd/db-catalyst/testdata/` directory
- **Issues**: [GitHub Issues](https://github.com/electwix/db-catalyst/issues)

## Complete Example

### `db-catalyst.toml`
```toml
package = "db"
out = "db"
sqlite_driver = "modernc"
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]

[custom_types]
[[custom_types.mapping]]
custom_type = "UserID"
sqlite_type = "INTEGER"
go_type = "github.com/company/types.UserID"
pointer = false

[[custom_types.mapping]]
custom_type = "Status"
sqlite_type = "TEXT"
go_type = "github.com/company/types.Status"
pointer = true

[prepared_queries]
enabled = true
metrics = true
thread_safe = true
```

### Generated Code Structure
```
db/
├── models.gen.go          # Model structs
├── querier.gen.go         # Interface definitions
├── _helpers.gen.go        # Result types and scanners
├── query_get_user.go      # Individual query implementations
├── query_list_users.go
└── schema.gen.sql         # Transformed schema (if custom types)
```

### Usage
```go
package main

import (
    "context"
    "database/sql"
    "github.com/yourproject/db"
)

func main() {
    sqlDB, _ := sql.Open("modernc.org/sqlite", ":memory:")
    queries := db.New(sqlDB)
    
    // All the same APIs you're used to!
    user, err := queries.GetUser(context.Background(), 123)
    if err != nil {
        panic(err)
    }
    
    // With proper custom types!
    var userID types.UserID = user.ID  // Strongly typed!
    var status *types.Status = user.Status  // Proper pointer!
}
```

Welcome to db-catalyst! 🚀