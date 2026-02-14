# SQLC to db-catalyst Migration Guide

This guide helps migrate from SQLC to db-catalyst, focusing on the key configuration differences.

## Quick Comparison

| Feature | SQLC | db-catalyst |
|---------|------|-------------|
| Config file | `sqlc.yaml` | `db-catalyst.toml` |
| Engine | `engine: sqlite` | `database = "sqlite"` (default) |
| Package | `gen.go.package` | `package = "db"` |
| Output | `gen.go.out` | `out = "dbgen"` |
| Queries | `queries:` array | `queries = []` array |
| Schema | `schema:` array | `schemas = []` array |
| Prepared queries | `emit_prepared_queries` | `[prepared_queries]` section |
| JSON tags | `emit_json_tags` | `emit_json_tags = true` in `[generation]` |
| Empty slices | `emit_empty_slices` | `emit_empty_slices = true` in `[generation]` |
| Pointers for null | `emit_pointers_for_null` | `emit_pointers_for_null = true` in `[generation]` |

## Column Overrides Migration

### SQLC Format (yaml)

```yaml
version: '2'
sql:
  - engine: 'sqlite'
    queries: 'queries/'
    schema: 'schema/'
    gen:
      go:
        package: 'dbgen'
        out: 'dbgen'
        emit_prepared_queries: true
        emit_empty_slices: true
        overrides:
          - column: 'user_.id'
            go_type:
              import: 'epin-mono/libs/db/pkg/idwrap'
              package: 'idwrap'
              type: 'IDWrap'
          - column: 'user_.role_id'
            go_type:
              import: 'epin-mono/libs/db/pkg/idwrap'
              package: 'idwrap'
              type: 'IDWrap'
          - column: 'seller.name'
            go_type: 'string'
```

### db-catalyst Format (toml)

```toml
package = "dbgen"
out = "dbgen"
schemas = ["schema/"]
queries = ["queries/"]

[prepared_queries]
enabled = true

[generation]
emit_empty_slices = true

[[overrides]]
column = "user_.id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "user_.role_id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "seller.name"
go_type = "string"
```

## Key Differences

### 1. Configuration Structure

**SQLC** uses a nested YAML structure with database-specific settings under `sql:` array.

**db-catalyst** uses a flat TOML structure at the root level with sections for related options.

### 2. Column Override Syntax

Both support the same features:
- Simple type: `go_type = "string"`
- Complex type with import: `go_type = { import = "...", package = "...", type = "..." }`
- Pointer support: `go_type = { type = "IDWrap", pointer = true }`

### 3. Command Line

**SQLC:**
```bash
sqlc generate
sqlc vet
```

**db-catalyst:**
```bash
db-catalyst -c db-catalyst.toml
db-catalyst -c db-catalyst.toml -dry-run  # Preview changes
```

## Migration Steps

1. **Create db-catalyst.toml** alongside your `sqlc.yaml`
2. **Migrate basic settings** (package, out, schemas, queries)
3. **Migrate generation options** to `[generation]` section
4. **Migrate column overrides** using `[[overrides]]` array
5. **Test generation** with `db-catalyst -c db-catalyst.toml`
6. **Update imports** in your Go code (package name may change)
7. **Remove sqlc.yaml** once migration is complete

## Epin-Mono Example

Complete migration for a complex project like epin-mono:

```toml
# db-catalyst.toml
package = "db"
out = "dbgen"
schemas = ["schema/"]
queries = ["queries/"]

[prepared_queries]
enabled = true
emit_empty_slices = true

[generation]
emit_json_tags = true
emit_pointers_for_null = true

# ULID ID columns - mapped to custom IDWrap type
[[overrides]]
column = "user_.id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "user_.role_id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "user_.user_kind_id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "seller.id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "seller.user_id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

[[overrides]]
column = "seller.seller_kind_id"
go_type = { import = "epin-mono/libs/db/pkg/idwrap", package = "idwrap", type = "IDWrap" }

# Role permission bits - use native Go types
[[overrides]]
column = "role.perm_bits_first"
go_type = "uint64"

[[overrides]]
column = "role.perm_bits_second"
go_type = "uint64"

# User kind color - use uint32 for RGB
[[overrides]]
column = "user_kind.color_rgb"
go_type = "uint32"
```

## Troubleshooting

### Unknown table/alias "excluded"

**Problem:** UPSERT queries with `ON CONFLICT ... DO UPDATE SET col = excluded.col` fail.

**Status:** ✅ Fixed in db-catalyst. The `excluded` pseudo-table is now recognized.

### Schema INSERT statements

**Problem:** Schema files with `INSERT INTO` statements for data seeding.

**Status:** ✅ Fixed. INSERT statements in schema files now produce warnings instead of errors.

### Cross-file foreign keys

**Problem:** Foreign key references to tables defined in other schema files fail validation.

**Workaround:** Use `-- db-catalyst:ignore` comment above the constraint to skip validation.

### Custom types in schema

**Problem:** Schema uses custom types like `idwrap` instead of `BLOB`.

**Solution:** Use column overrides to map the BLOB columns to your custom Go types.

## Feature Support Matrix

| Feature | SQLC | db-catalyst | Notes |
|---------|------|-------------|-------|
| Basic CRUD | ✅ | ✅ | Full support |
| Transactions | ✅ | ✅ | Via WithTx() |
| Prepared queries | ✅ | ✅ | With metrics, thread-safe options |
| JSON tags | ✅ | ✅ | Optional |
| Empty slices | ✅ | ✅ | Optional |
| Pointers for null | ✅ | ✅ | Optional |
| Column overrides | ✅ | ✅ | Full support with imports |
| Custom types in schema | ✅ | ⚠️ | Use column overrides |
| sqlc.arg() | ✅ | ✅ | Supported |
| sqlc.narg() | ✅ | ✅ | Supported |
| sqlc.slice() | ✅ | ✅ | Supported |
| ON CONFLICT | ✅ | ✅ | Now supported |
| RETURNING | ✅ | ✅ | Full support |
| CTEs (WITH) | ✅ | ✅ | Full support |
| Cursor pagination | ✅ | ⚠️ | Partial - use sqlc.narg() pattern |
| Stored procedures | ❌ | ❌ | Not supported in either |

## Getting Help

- Run `db-catalyst --help` for CLI options
- Check `AGENTS.md` in this repo for development guidelines
- File issues at: https://github.com/ElecTwix/db-catalyst/issues
