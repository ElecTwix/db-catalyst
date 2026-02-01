# Examples

This directory contains examples demonstrating db-catalyst features.

## Examples

### [simple](simple/) - Basic CRUD
- Basic CREATE TABLE
- Simple SELECT/INSERT/UPDATE/DELETE
- Named queries with `:one`, `:many`, `:exec`
- Nullable fields

### [complex](complex/) - Advanced SQL
- Many-to-many relationships
- CTEs (Common Table Expressions)
- JOINs and aggregations
- Complex queries with subqueries

### [advanced](advanced/) - Custom Types & Prepared Queries
- **Custom types with pointer control** (unique to db-catalyst)
- **Schema transformation** (unique to db-catalyst)
- **Prepared queries with metrics** (unique to db-catalyst)
- Type-safe domain modeling

## Running Examples

Each example can be run independently:

```bash
cd examples/simple
go generate ./...
go run main.go
```

Or use the db-catalyst CLI directly:

```bash
cd examples/simple
db-catalyst
go run main.go
```

## What Makes These Examples Special

Unlike sqlc examples, these demonstrate db-catalyst's unique features:

1. **Grammar-driven parsing** - Clean, extensible architecture
2. **Custom type system** - Map SQLite types to domain types
3. **Schema transformation** - Use domain types in SQL
4. **Modern Go** - Context-first, clean structs, no interfaces
