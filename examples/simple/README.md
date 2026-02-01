# Simple Example - Basic CRUD Operations

This example demonstrates the simplest use case: basic CRUD operations on a users table.

## Structure

```
simple/
├── schema/
│   └── users.sql       # Table definition
├── queries/
│   └── users.sql       # CRUD queries
├── db-catalyst.toml    # Configuration
└── main.go            # Usage example
```

## Running

```bash
cd examples/simple

# Generate code
db-catalyst

# Run the example
go run main.go
```

## Generated Code

The tool generates:
- `models.gen.go` - User struct
- `querier.gen.go` - Interface definition
- `query_*.go` - Individual query implementations
- `_helpers.gen.go` - Scanner helpers

## Features Demonstrated

- Basic CREATE TABLE
- Simple SELECT/INSERT/UPDATE/DELETE
- Named queries with `:one`, `:many`, `:exec`
- Nullable fields (email)
