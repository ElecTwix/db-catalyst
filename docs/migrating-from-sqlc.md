# Migrating from sqlc to db-catalyst

This guide helps you migrate your existing sqlc projects to db-catalyst, focusing on the key differences and providing step-by-step instructions.

## Overview

db-catalyst is a SQLite-specific code generator that provides similar functionality to sqlc but with a simpler, more focused approach. The main differences are:

- **SQLite-only**: Optimized specifically for SQLite, with no multi-database complexity
- **Custom types support**: Built-in support for custom type mappings with schema transformation
- **Simplified configuration**: Cleaner TOML configuration with fewer options
- **Generated file naming**: Uses `.gen.` prefixes for all generated files

## Quick Migration Checklist

- [ ] Convert `sqlc.json` to `db-catalyst.toml`
- [ ] Update custom types configuration (if applicable)
- [ ] Adjust file naming expectations (`.gen.` prefix)
- [ ] Update build scripts and imports
- [ ] Test generated code

## Configuration Migration

### sqlc.json to db-catalyst.toml

#### Before (sqlc.json)
```json
{
  "version": "2",
  "sql": [
    {
      "engine": "sqlite",
      "queries": "queries/",
      "schema": "schema/"
    }
  ],
  "overrides": {
    "go": {
      "package": "db",
      "out": "db/",
      "sql_package": "database/sql",
      "emit_json_tags": true,
      "emit_prepared_queries": false,
      "emit_interface": true,
      "emit_exact_table_names": false
    }
  },
  "plugins": []
}
```

#### After (db-catalyst.toml)
```toml
package = "db"
out = "db"
sqlite_driver = "modernc"  # or "mattn"
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]

[prepared_queries]
enabled = false
metrics = false
thread_safe = false
```

### Key Configuration Differences

| sqlc | db-catalyst | Notes |
|------|-------------|-------|
| `version` | Not needed | db-catalyst has a single stable version |
| `sql[].engine` | `sqlite_driver` | Choose "modernc" or "mattn" |
| `sql[].queries` | `queries` | Supports glob patterns |
| `sql[].schema` | `schemas` | Supports glob patterns, plural name |
| `overrides.go.package` | `package` | Direct field mapping |
| `overrides.go.out` | `out` | Direct field mapping |
| `overrides.go.emit_json_tags` | Not needed | JSON tags are always emitted |
| `overrides.go.emit_prepared_queries` | `prepared_queries.enabled` | More explicit configuration |
| `overrides.go.sql_package` | Not needed | Always uses `database/sql` |
| `plugins` | Not needed | db-catalyst has built-in functionality |

## Custom Types Migration

### sqlc Custom Types

#### Before (sqlc)
```json
{
  "overrides": {
    "go": {
      "overrides": [
        {
          "db_type": "TEXT",
          "go_type": "github.com/company/types.ID"
        },
        {
          "column": "users.status",
          "go_type": {
            "type": "Status",
            "import": "github.com/company/types",
            "package": "types",
            "pointer": true
          }
        }
      ]
    }
  }
}
```

#### After (db-catalyst)
```toml
[custom_types]
[[custom_types.mapping]]
custom_type = "ID"
sqlite_type = "TEXT"
go_type = "github.com/company/types.ID"
go_import = "github.com/company/types"
go_package = "types"
pointer = false

[[custom_types.mapping]]
custom_type = "Status"
sqlite_type = "TEXT"
go_type = "Status"
go_import = "github.com/company/types"
go_package = "types"
pointer = true
```

### Custom Types Advantages in db-catalyst

1. **Schema Transformation**: db-catalyst automatically generates a `schema.gen.sql` file with SQLite-compatible types
2. **Cleaner Syntax**: Use custom types directly in your schema:
   ```sql
   CREATE TABLE users (
       id ID PRIMARY KEY,
       status Status NOT NULL
   );
   ```
3. **Type Safety**: Generated Go code uses your custom types while maintaining SQLite compatibility

## File Naming Changes

db-catalyst uses `.gen.` prefixes for all generated files to clearly distinguish generated code:

| sqlc | db-catalyst |
|------|-------------|
| `db.go` | `db.gen.go` |
| `models.go` | `models.gen.go` |
| `querier.go` | `querier.gen.go` |
| `_helpers.go` | `_helpers.gen.go` |
| `query_name.go` | `query_name.gen.go` |

### Update Your Imports

#### Before
```go
import (
    "yourproject/db"
)
```

#### After
```go
import (
    "yourproject/db"
)
```

The import path stays the same, but the generated files within the package have new names.

## Query Syntax Compatibility

db-catalyst supports the same query syntax as sqlc for SQLite:

### Supported Query Commands
- `:one` - Execute and return one row
- `:many` - Execute and return multiple rows  
- `:exec` - Execute without returning rows
- `:execresult` - Execute and return result info

### Example Queries

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (?, ?);

-- name: UpdateUser :execresult
UPDATE users SET name = ? WHERE id = ?;
```

All existing sqlc queries should work without modification in db-catalyst.

## Step-by-Step Migration

### 1. Backup Your Project
```bash
cp -r your-project your-project-backup
```

### 2. Install db-catalyst
```bash
go install github.com/electwix/db-catalyst/cmd/db-catalyst@latest
```

### 3. Convert Configuration
Create `db-catalyst.toml`:

```toml
package = "db"  # Match your sqlc package name
out = "db"     # Match your sqlc output directory
sqlite_driver = "modernc"  # or "mattn"
schemas = ["schema/*.sql"]   # Adjust paths as needed
queries = ["queries/*.sql"]  # Adjust paths as needed
```

### 4. Handle Custom Types (if applicable)

If you use custom types in sqlc:

1. **Create custom types configuration** in `db-catalyst.toml`
2. **Update your schema files** to use custom type names directly
3. **Remove old sqlc overrides** from your configuration

Example migration:

#### Original Schema
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL
);
```

#### Updated Schema with Custom Types
```sql
CREATE TABLE users (
    id ID PRIMARY KEY,
    status Status NOT NULL
);
```

#### Custom Types Configuration
```toml
[custom_types]
[[custom_types.mapping]]
custom_type = "ID"
sqlite_type = "TEXT"
go_type = "github.com/yourproject/types.ID"
go_import = "github.com/yourproject/types"
go_package = "types"

[[custom_types.mapping]]
custom_type = "Status"
sqlite_type = "TEXT"
go_type = "Status"
go_import = "github.com/yourproject/types"
go_package = "types"
pointer = true
```

### 5. Update Build Scripts

#### Makefile Example
```makefile
# Before
generate:
	sqlc generate

# After  
generate:
	db-catalyst generate
```

#### Go Generate Example
```go
//go:generate sqlc generate
```

Change to:
```go
//go:generate db-catalyst generate
```

### 6. Update Generated File References

If you have any direct references to generated files (unlikely but possible):

```bash
# Find any references to old file names
grep -r "db\.go" .
grep -r "models\.go" .
grep -r "querier\.go" .
```

Update imports if you were importing specific generated files (not recommended):

#### Before
```go
import _ "yourproject/db/models"
```

#### After
```go
import _ "yourproject/db/models.gen"
```

### 7. Test the Migration

```bash
# Clean old generated files
rm -rf db/*

# Generate with db-catalyst
db-catalyst generate

# Check that files were generated
ls -la db/

# Run your tests
go test ./...
```

### 8. Commit Changes

```bash
git add .
git commit -m "Migrate from sqlc to db-catalyst"
```

## Advanced Migration Topics

### Prepared Queries

If you used sqlc's prepared queries:

#### sqlc Configuration
```json
{
  "overrides": {
    "go": {
      "emit_prepared_queries": true
    }
  }
}
```

#### db-catalyst Configuration
```toml
[prepared_queries]
enabled = true
metrics = false    # Optional: enable query metrics
thread_safe = true # Optional: make prepared queries thread-safe
```

### Multiple Schema Files

Both sqlc and db-catalyst support multiple schema files, but db-catalyst uses glob patterns:

#### sqlc
```json
{
  "sql": [
    {
      "schema": ["schema1.sql", "schema2.sql"]
    }
  ]
}
```

#### db-catalyst
```toml
schemas = ["schema/*.sql"]
```

### Complex Custom Type Migrations

For complex custom type setups, consider this migration strategy:

1. **Start with default types** - Migrate without custom types first
2. **Add custom types gradually** - Introduce custom types one at a time
3. **Validate each step** - Ensure tests pass after each change

Example gradual migration:

#### Step 1: Basic Migration
```toml
# No custom types yet
package = "db"
out = "db"
sqlite_driver = "modernc"
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]
```

#### Step 2: Add First Custom Type
```sql
-- Keep original schema for now
CREATE TABLE users (
    id TEXT PRIMARY KEY,  -- Will migrate to ID later
    name TEXT NOT NULL
);
```

```toml
[custom_types]
[[custom_types.mapping]]
custom_type = "ID"
sqlite_type = "TEXT"
go_type = "github.com/yourproject/types.ID"
```

#### Step 3: Update Schema
```sql
-- Now use custom type
CREATE TABLE users (
    id ID PRIMARY KEY,
    name TEXT NOT NULL
);
```

## Troubleshooting

### Common Issues

#### 1. "custom types used but not defined" Error

**Problem**: You're using a custom type in your schema but haven't defined it in the configuration.

**Solution**: Add the custom type mapping to `db-catalyst.toml`:
```toml
[custom_types]
[[custom_types.mapping]]
custom_type = "YourCustomType"
sqlite_type = "TEXT"
go_type = "path/to/YourCustomType"
```

#### 2. Generated Code Uses Wrong Types

**Problem**: Query parameters or results use default Go types instead of custom types.

**Solution**: Ensure your custom type mappings include the correct `sqlite_type` that matches what's in your transformed schema.

#### 3. Import Path Issues

**Problem**: Generated code has incorrect import paths for custom types.

**Solution**: Use the full import path in `go_import`:
```toml
go_import = "github.com/yourproject/types"
```

#### 4. Build Failures After Migration

**Problem**: Go build fails after migrating to db-catalyst.

**Solution**: Check for:
- Old file name references (`.go` vs `.gen.go`)
- Missing custom type imports
- Interface changes (db-catalyst may generate slightly different interfaces)

### Getting Help

If you encounter issues during migration:

1. **Check the diagnostics**: db-catalyst provides detailed error messages
2. **Use dry-run mode**: `db-catalyst generate --dry-run` to check without writing files
3. **Compare output**: Use `diff` to compare old and new generated code
4. **Gradual migration**: Migrate complex projects incrementally

## Benefits of Migration

After migrating to db-catalyst, you'll enjoy:

1. **Better Performance**: SQLite-specific optimizations
2. **Cleaner Configuration**: Simpler TOML format
3. **Enhanced Type Safety**: Better custom type support with schema transformation
4. **Active Development**: Focused on SQLite improvements
5. **Clearer Generated Code**: `.gen.` prefixes distinguish generated files
6. **Better Tooling**: Integrated schema transformation and validation

## Example: Complete Migration

Let's see a complete example migration from sqlc to db-catalyst.

### Before: sqlc Project

```
project/
├── sqlc.json
├── schema/
│   └── users.sql
├── queries/
│   └── users.sql
└── go.mod
```

**sqlc.json:**
```json
{
  "version": "2",
  "sql": [
    {
      "engine": "sqlite",
      "queries": "queries/",
      "schema": "schema/"
    }
  ],
  "overrides": {
    "go": {
      "package": "db",
      "out": "db/",
      "sql_package": "database/sql",
      "emit_json_tags": true,
      "emit_prepared_queries": false,
      "overrides": [
        {
          "db_type": "TEXT",
          "go_type": "github.com/project/types.ID"
        }
      ]
    }
  }
}
```

**schema/users.sql:**
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);
```

### After: db-catalyst Project

```
project/
├── db-catalyst.toml
├── schema/
│   └── users.sql
├── queries/
│   └── users.sql
└── go.mod
```

**db-catalyst.toml:**
```toml
package = "db"
out = "db"
sqlite_driver = "modernc"
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]

[custom_types]
[[custom_types.mapping]]
custom_type = "ID"
sqlite_type = "TEXT"
go_type = "github.com/project/types.ID"
go_import = "github.com/project/types"
go_package = "types"
```

**schema/users.sql:**
```sql
CREATE TABLE users (
    id ID PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);
```

**Generated schema.gen.sql (new):**
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);
```

### Generated Files Comparison

#### sqlc Generated Files
```
db/
├── db.go
├── models.go
├── querier.go
├── _helpers.go
└── query_users.sql.go
```

#### db-catalyst Generated Files
```
db/
├── db.gen.go
├── models.gen.go
├── querier.gen.go
├── _helpers.gen.go
├── query_users.gen.go
└── schema.gen.sql  # New: transformed schema
```

The migration is complete! Your project now uses db-catalyst with enhanced custom type support and cleaner generated code.