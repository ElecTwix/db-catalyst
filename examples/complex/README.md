# Complex Example - Advanced SQL Features

This example demonstrates advanced SQL features: joins, CTEs, aggregations, and sqlc macros.

## Features Demonstrated

- **Many-to-many relationships** (posts <-> tags)
- **CTEs** (Common Table Expressions)
- **JOINs** (INNER, LEFT)
- **Aggregations** (COUNT, GROUP BY)
- **sqlc macros** (slice, arg)
- **Complex queries** with subqueries

## Schema

- `authors` - Blog authors
- `posts` - Blog posts with foreign key to authors
- `tags` - Available tags
- `post_tags` - Many-to-many join table

## Running

```bash
cd examples/complex
db-catalyst
go run main.go
```
