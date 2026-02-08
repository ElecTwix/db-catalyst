# Roadmap

db-catalyst follows a structured release plan prioritizing simplicity and SQLite focus.

## Version History

| Version | Date | Status |
|---------|------|--------|
| v0.1.0 | Jan 2026 | Released |
| v0.2.0 | Feb 2026 | Released |
| v0.3.0 | Feb 2026 | Released |
| v0.4.0 | Feb 2026 | Released |
| v0.5.0 | TBD | In Progress |

## v0.1.0 (Released)

- Config parsing (TOML)
- Schema catalog
- Query analysis
- Go code generation
- CLI with dry-run mode
- sqlc config migration tool
- Prepared queries support

## v0.2.0 - Quality Polish

**Goal:** Improve developer experience and documentation.

### Completed
- [x] Emit empty slices option
- [x] Column ordering determinism
- [x] Documentation (schema, query, generated-code)
  - [x] Comprehensive schema reference with examples
  - [x] Complete query annotation guide
  - [x] Generated code usage patterns
  - [x] Best practices and troubleshooting

### Completed
- [x] Better parameter name inference
  - [x] Forward equality pattern (? = column)
  - [x] BETWEEN second parameter naming (columnUpper)
  - [x] HAVING clause parameter inference
- [x] Detailed diagnostics
  - [x] EndColumn and Length fields
  - [x] Multi-character underlines for spans
  - [x] Token-based diagnostic creation

## v0.3.0 - Optional Features

**Goal:** Additive features without complexity creep.

### Completed

1. **Pointer Nullables** ✅
   ```toml
   [generation]
   emit_pointers_for_null = true
   ```
   - Generates `*string` instead of `sql.NullString` for nullable columns
   - CLI flag: `--emit-pointers-for-null`

2. **JSON Tag Control** ✅
   ```toml
   [generation]
   emit_json_tags = false
   ```
   - CLI flag: `--no-json-tags`

3. **Deterministic Caching** ✅
   ```toml
   [cache]
   enabled = true
   dir = ".db-catalyst-cache"
   ```
   - File-based JSON cache with content-hash keys
   - Schema and query block-level caching
   - CLI: `--clear-cache` to clear cache
   - Target achieved: ~20ms for incremental builds (well under 200ms)

4. **Performance Benchmarks** ✅
   - Local benchmark suite (no CI - single-person project)
   - Key benchmarks: Pipeline, Tokenizer, SchemaParser, QueryParser
   - Save/compare with: `benchstat old.txt new.txt`
   - Run before major changes to detect regressions

## v0.4.0 - Developer Experience

**Goal:** Address known limitations and improve flexibility.

### Completed

1. **Parameter Type Override** ✅
   Allow explicit parameter types in SQL comments:
   ```sql
   -- @param userId: uuid.UUID
   SELECT * FROM users WHERE id = :user_id;
   ```
   - Supports any Go type (e.g., `uuid.UUID`, `custom.Email`)
   - Takes precedence over automatic type inference
   - Multiple `@param` annotations per query supported

### Completed

1. **Engine Interface Boundaries** ✅
   - Defined abstract database engine interface (`internal/engine`)
   - Separated dialect-specific logic from core
   - Created SQLite and PostgreSQL engine implementations
   - Pipeline now supports engine injection via `Environment.Engine`
   - Registry pattern for engine discovery and instantiation
   - All tests pass without modification (backward compatible)
   
   **New Packages:**
   - `internal/engine` - Core interfaces and registry
   - `internal/engine/sqlite` - SQLite engine implementation
   - `internal/engine/postgres` - PostgreSQL engine implementation  
   - `internal/engine/builtin` - Built-in engine registration
   - `internal/engine/mysql` - MySQL engine stub (future)

### Planned Features

## v0.5.0 - Multi-Database Support

**Goal:** Leverage the new engine interface to provide true multi-database code generation.

### Planned Features

1. **PostgreSQL DDL Parser** 🔄
   - Native PostgreSQL schema parsing (currently uses SQLite parser)
   - Support for PostgreSQL-specific syntax:
     - `SERIAL`, `BIGSERIAL` auto-increment
     - `JSONB` with GIN indexes
     - Array types (`TEXT[]`, `INTEGER[]`)
     - `UUID` with `gen_random_uuid()`
   - Parse `CREATE TYPE` for enums
   - Parse domain constraints

2. **Engine-Aware Code Generation** 🔄
   - Update CLI to use engines instead of hardcoded logic
   - Generate database-specific queries using engine interfaces
   - Per-engine query validation
   - Engine-specific prepared statement generation

3. **MySQL Engine** 🔄
   - Full MySQL type mapper implementation
   - MySQL DDL parser (CREATE TABLE with MySQL-specific syntax)
   - Support for MySQL features:
     - `AUTO_INCREMENT`
     - `TIMESTAMP` defaults
     - `ENUM` and `SET` types
     - Full-text indexes

4. **Database-Specific Optimizations** 🔄
   - Engine-specific query hints
   - Connection pool configuration per database
   - Transaction isolation level recommendations

### Implementation Plan

1. **Week 1-2**: PostgreSQL DDL Parser
   - Create `internal/schema/parser/postgres` package
   - Support core PostgreSQL DDL statements
   - Add tests with real PostgreSQL schemas

2. **Week 3-4**: Engine Integration in CLI
   - Migrate CLI to use engine interfaces
   - Remove hardcoded SQLite references
   - Add database selection validation

3. **Week 5-6**: MySQL Engine
   - Implement MySQL type mapper
   - Create MySQL DDL parser
   - Add MySQL examples

4. **Week 7-8**: Testing & Polish
   - Integration tests for all three databases
   - Performance benchmarks per engine
   - Documentation updates

## Non-Goals

These are explicitly out of scope:

- Multiple database drivers (stick to SQLite)
- Complex migrations (use separate tools)
- ORMs or query builders
- GUI or web interfaces
- Plugin systems

## Contributing

See [AGENTS.md](../AGENTS.md) for contribution guidelines.

## Release Checklist

- [ ] All tests pass
- [ ] Benchmarks run clean
- [ ] Docs updated
- [ ] Changelog entry
- [ ] Version bump
