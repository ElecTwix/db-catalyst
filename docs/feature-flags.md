# Feature Flags

This document captures configuration toggles that change the db-catalyst code generator or runtime behaviour. All flags live inside `db-catalyst.toml` unless otherwise noted.

## Generation Options

Control the style and format of generated code.

### JSON Tags

```toml
[generation]
emit_json_tags = false
```

- `emit_json_tags` *(bool, default `true`)*: when `true`, generated structs include `json:"column_name"` tags. Disable when you don't need JSON serialization or prefer to handle it separately.

CLI override: `--no-json-tags`

### Pointer Nullables

```toml
[generation]
emit_pointers_for_null = true
```

- `emit_pointers_for_null` *(bool, default `false`)*: when `true`, nullable columns use pointer types (`*string`, `*int64`) instead of `sql.NullString`, `sql.NullInt64`, etc. This produces more idiomatic Go code but requires checking for nil instead of using the `.Valid` field.

CLI override: `--emit-pointers-for-null`

### Empty Slices

```toml
[generation]
emit_empty_slices = true
```

- `emit_empty_slices` *(bool, default `false`)*: when `true`, queries that return slices will return an empty slice `[]Type{}` instead of `nil` when no rows are found.

## Cache

Enable deterministic caching for faster incremental builds. The cache stores parsed ASTs and query analysis results.

```toml
[cache]
enabled = true
dir = ".db-catalyst-cache"
```

- `enabled` *(bool, default `false`)*: when `true`, enables file-based caching of parsed schemas and queries.
- `dir` *(string, default `.db-catalyst-cache`)*: directory where cache files are stored. Relative paths are resolved from the project root.

Cache invalidation is automatic based on:
- File content hashes (modifications invalidate cache)
- Schema/query file changes
- db-catalyst version

CLI commands:
- `--clear-cache`: Clear the cache and exit

Performance: Caching achieves ~20ms builds for small-to-medium projects (target was <200ms).

## Parameter Type Override

Override automatic type inference with explicit type annotations in SQL comments.

```sql
-- @param userId: uuid.UUID
-- @param email: custom.Email
-- name: GetUser :one
SELECT * FROM users WHERE id = :user_id AND email = :email;
```

- Use `-- @param paramName: GoType` syntax in query documentation comments
- The `paramName` should match the camelCase version of your SQL parameter
- Supports any valid Go type including custom types from imports
- Multiple `@param` annotations per query are supported
- Explicit types take precedence over automatic inference

Examples:
```sql
-- Use UUID type from github.com/google/uuid
-- @param id: uuid.UUID
SELECT * FROM users WHERE id = :id;

-- Custom type from your project
-- @param status: myapp.UserStatus
SELECT * FROM users WHERE status = :status;

-- Slice of custom types
-- @param ids: []uuid.UUID
SELECT * FROM users WHERE id IN (sqlc.slice('ids'));
```

## Prepared Queries

The generator can emit a prepared-statement aware wrapper that reuses compiled SQL statements and optionally instruments each invocation.

```toml
[prepared_queries]
enabled = true
metrics = true
thread_safe = true
```

- `enabled` *(bool, default `false`)*: when `true`, db-catalyst emits a `prepared.go` companion and the `Prepare` helper. Legacy output remains untouched when this is disabled.
- `metrics` *(bool, default `false`)*: wraps each prepared method with a duration/error callback. Provide a `PreparedMetricsRecorder` implementation when using this toggle; otherwise the hooks remain dormant.
- `thread_safe` *(bool, default `false`)*: guards statement preparation and closure with per-query mutexes so concurrent goroutines can lazily initialize statements safely. When `false`, statements are prepared eagerly in `Prepare` and cached without additional locking.

> **Lifecycle tip:** `Prepare` returns a `PreparedQueries` wrapper that exposes `Close()`. Always call `Close()` when you are done to release held statements; when `thread_safe` is enabled the method is idempotent under the hood.

## Adding New Flags

Keep feature flags additive: default to the legacy behaviour, gate optional output behind explicit opt-in, and document new keys both here and in `db-catalyst-spec.md`. Update goldens and specs whenever a flag changes generated output.
