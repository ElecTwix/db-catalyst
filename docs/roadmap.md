# Roadmap

db-catalyst follows a structured release plan prioritizing simplicity and SQLite focus.

## Version History

| Version | Date | Status |
|---------|------|--------|
| v0.1.0 | Jan 2026 | Released |
| v0.2.0 | Feb 2026 | Released |
| v0.3.0 | Feb 2026 | Released |
| v0.4.0 | Feb 2026 | Released |
| v0.5.0 | Feb 2026 | Released |
| v0.6.0 | TBD | Planning |

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

### Completed

1. **PostgreSQL DDL Parser** ✅
   - Native PostgreSQL schema parsing implemented
   - Support for PostgreSQL-specific syntax:
     - `SERIAL`, `BIGSERIAL` auto-increment
     - `JSONB` with GIN indexes
     - Array types (`TEXT[]`, `INTEGER[]`)
     - `UUID` with `gen_random_uuid()`
   - Parse `CREATE TYPE` for enums
   - Parse domain constraints
   - Full CREATE TABLE with constraints
   - CREATE INDEX with USING clause
   - CREATE VIEW support
   - ALTER TABLE support
   
   **New Packages:**
   - `internal/schema/diagnostic` - Shared diagnostic types (avoids import cycles)
   - `internal/schema/parser/postgres` - Native PostgreSQL DDL parser
   
   All tests pass, integrated with PostgreSQL engine.

### Completed

2. **Engine-Aware Code Generation** ✅
   - CLI now uses engines instead of hardcoded logic
   - Added `--database` flag to override config database setting
   - Database selection validation with helpful error messages
   - Engine automatically created based on database dialect
   - Integrated with pipeline environment
   - All tests pass, backward compatible

### Completed

3. **MySQL Engine** ✅
   - Full MySQL type mapper implemented with MySQL-specific types:
     - TINYINT, SMALLINT, MEDIUMINT, INT, BIGINT
     - ENUM and SET types
     - JSON type (MySQL 5.7+)
     - Full-text indexes (FULLTEXT INDEX)
   - Native MySQL DDL parser created
   - AUTO_INCREMENT support
   - TIMESTAMP with DEFAULT CURRENT_TIMESTAMP
   - MySQL table options (ENGINE, CHARSET, COLLATE)
   - Column attributes (UNSIGNED, ZEROFILL, COMMENT)
   
   **New Packages:**
   - `internal/engine/mysql` - MySQL engine implementation
   - `internal/schema/parser/mysql` - Native MySQL DDL parser
   
   **New Semantic Types:**
   - CategoryMediumInteger, CategoryTinyText, CategoryMediumText
   - CategoryLongText, CategoryTinyBlob, CategoryMediumBlob
   - CategoryLongBlob, CategoryBinary, CategoryDateTime
   - CategoryYear, CategorySet
   
   All tests pass, integrated with CLI via `--database mysql`.

### Completed

4. **PostgreSQL Enums and Domains** ✅
   - Added `Enums` field to `model.Catalog`
   - Added `Domains` field to `model.Catalog`
   - Parse `CREATE TYPE ... AS ENUM` statements
   - Parse `CREATE DOMAIN` with constraints
   - Fixed multiple statement parsing in PostgreSQL parser
   - 45+ new tests added

5. **Integration Testing** ✅
   - 24 comprehensive test suites across all three databases
   - MySQL tests: CRUD, JSON operations, full-text search
   - PostgreSQL tests: + enums, domains, arrays, JSONB
   - SQLite tests: CRUD, JSON operations
   - End-to-end pipeline tests (SQL → Parse → Generate → Compile → Execute)
   - Docker Compose with health checks for PostgreSQL and MySQL

6. **Database-Specific Optimizations** ✅
   - **Connection Pool Configuration**: Per-database recommendations
     - SQLite: Small pools (5 max open, 2 idle)
     - PostgreSQL: Medium pools (25 max open, 5 idle)
     - MySQL: Medium pools (25 max open, 5 idle)
   - **Transaction Isolation Levels**: Database-appropriate support
     - SQLite: Serializable only
     - PostgreSQL: ReadCommitted, RepeatableRead, Serializable
     - MySQL: ReadUncommitted, ReadCommitted, RepeatableRead, Serializable
   - **Query Hints**: Engine-specific optimization hints
     - PostgreSQL: pg_hint_plan extension hints (8 hints)
     - MySQL: Index hints and optimizer hints (12 hints)
     - SQLite: No hints supported

### Implementation Plan

1. **✅ Week 1-2**: PostgreSQL DDL Parser - COMPLETED
2. **✅ Week 3-4**: Engine Integration in CLI - COMPLETED
3. **✅ Week 5-6**: MySQL Engine - COMPLETED
4. **✅ Week 7-8**: Testing, Polish & Optimizations - COMPLETED

## v0.6.0 - Language Expansion (Planning)

**Goal:** Extend code generation beyond Go to other languages.

### Planned Features

1. **Rust Code Generation**
   - Template-based generation using sqlx
   - Type mappings for PostgreSQL, MySQL, SQLite
   - Async/await support

2. **TypeScript Code Generation**
   - Template-based generation with pg driver
   - Type-safe query builders
   - Support for PostgreSQL features

3. **Language Selection**
   ```toml
   language = "rust"  # or "go" (default), "typescript"
   ```

### Non-Goals

- Multiple database drivers per language
- ORM generation
- GUI interfaces

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

- [x] All tests pass
- [x] Benchmarks run clean
- [x] Docs updated
- [x] Changelog entry
- [x] Version bump
