# sqlc Compatibility Status

This document tracks the compatibility of `db-catalyst` with `sqlc` features and patterns.

## Current Compatibility Summary

| Feature | Status | Notes |
|---------|--------|-------|
| **Core SQL** | Partail | SELECT, INSERT, UPDATE, DELETE supported. |
| **Named Parameters** | Full | `:param`, `?`, `?NNN` supported. |
| **sqlc.slice()** | Full | Variadic Go parameters (e.g., `[]int64`) generated. |
| **sqlc.arg()** | Full | Manual parameter naming. |
| **sqlc.narg()** | Partial | Named parameter recognized; nullability inference in progress. |
| **Implicit Aliases** | Full | `COUNT(*)`, `SUM(x)`, etc. now generate default names like `count`, `sum_x`. |
| **Star Expansion** | Full | `SELECT *` and `SELECT t.*` expanded into individual columns. |
| **RETURNING Clause** | Full | Basic support for `RETURNING *` and `RETURNING col` in DML. |
| **Clause Validation** | **New** | `WHERE`, `ORDER BY`, `GROUP BY` now validated for correct column/table names. |
| **CTEs** | Full | Standard and Recursive CTEs supported. |
| **Custom Types** | Advanced | Better than sqlc with explicit pointer control. |

## Detailed Feature Status

### 1. Extended Clause Validation (sqlc-parity)
`db-catalyst` now performs deep validation of your entire SQL statement, not just the `SELECT` list.
- **WHERE**: Catches typos in filter columns (e.g., `WHERE u.emial = ?`).
- **ORDER BY / GROUP BY**: Ensures sorting and grouping columns exist in the referenced tables.
- **JOIN**: Validates that join conditions use valid columns.

### 2. Implicit Aliases (sqlc-parity)
`db-catalyst` now automatically generates names for common aggregate functions when an explicit `AS` alias is omitted.
- `COUNT(*)` -> `count`
- `COUNT(col)` -> `count_col`
- `SUM(col)` -> `sum_col`
- `COALESCE(col, 0)` -> `col`

### 2. Star Expansion
`SELECT *` is now expanded using the schema catalog.
- If multiple tables are joined, all columns from all tables are expanded.
- `table.*` expands columns only for that specific table/alias.

### 3. sqlc Macros
- `WHERE id IN (sqlc.slice('ids'))`: Generates `ids []int64` and handles dynamic query expansion.
- `sqlc.arg('name')`: Forces the Go parameter name.
- `sqlc.narg('name')`: Forces a nullable parameter.

### 4. Known Gaps / Incompatibilities
- **Ambiguous Columns**: `db-catalyst` requires qualified names (`table.col`) if a column name exists in multiple tables in the query scope, whereas SQLite/sqlc might be more permissive if one is clearly intended.
- **Literal Projections**: `SELECT 1` without an alias currently fails or defaults to a warning.
- **Complex Subqueries**: Result column inference for highly nested subqueries is still being refined.
- **Advanced Driver Features**: Batching or specific `pgxv5` features are out of scope (SQLite focus).

## Migration Tips
1. Use the `sqlfix-sqlc` tool to migrate your `sqlc.yaml` overrides to `db-catalyst.toml`.
2. While we support implicit aliases, adding explicit aliases to complex expressions is still recommended for cleaner Go code.
3. Ensure your schema is compatible with the SQLite dialect.
