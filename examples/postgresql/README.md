# PostgreSQL Example

This example demonstrates db-catalyst's PostgreSQL support with various PostgreSQL-specific types and features.

## Features Demonstrated

- **UUID Primary Keys**: Using `gen_random_uuid()` for automatic UUID generation
- **JSONB Columns**: Storing flexible metadata with JSONB type
- **Array Types**: PostgreSQL arrays for tags and categories
- **Temporal Types**: TIMESTAMPTZ for timezone-aware timestamps
- **Decimal Types**: DECIMAL for precise numeric values
- **Foreign Keys**: Proper referential integrity with CASCADE deletes
- **GIN Indexes**: For efficient array and JSONB queries

## Schema Overview

### Tables

1. **users**: User accounts with UUID PK, JSONB metadata, and text arrays
2. **posts**: Blog posts with categories array, ratings, and publishing status
3. **comments**: Comments on posts with like counts
4. **tags**: Tag definitions for categorization
5. **post_tags**: Many-to-many junction table

## Queries

The example includes 30+ queries demonstrating:

- Basic CRUD operations
- Array containment queries (`ANY`, `@>`)
- JSONB containment queries
- Joins and aggregations
- Complex statistics queries
- Pagination with LIMIT

## Usage

```bash
# Generate code
cd examples/postgresql
db-catalyst generate

# The generated code will use pgx types for PostgreSQL compatibility
```

## Generated Types

The code generator will produce Go types using:

- `github.com/jackc/pgx/v5/pgtype` for PostgreSQL-specific types
- `UUID` for UUID columns
- `[]string` for text arrays
- `map[string]interface{}` for JSONB (or custom struct if defined)
- `pgtype.Timestamptz` for TIMESTAMPTZ
- `pgtype.Numeric` for DECIMAL
