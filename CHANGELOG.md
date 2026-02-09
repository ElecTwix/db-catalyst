# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-02-09

### Added
- **Multi-Database Support**: Full support for PostgreSQL and MySQL alongside SQLite
- **PostgreSQL DDL Parser**: Native parser supporting SERIAL, JSONB, arrays, UUID, CREATE TYPE (enums), CREATE DOMAIN
- **MySQL Engine**: Complete implementation with type mapper, DDL parser, AUTO_INCREMENT, and MySQL-specific types
- **PostgreSQL Enums and Domains**: Parse and store enum types and domain constraints in catalog
- **Integration Test Suite**: 24 comprehensive test suites with Docker Compose for all three databases
- **Database-Specific Optimizations**:
  - Connection pool configuration recommendations per database
  - Transaction isolation level support (SQLite: Serializable, PostgreSQL: 3 levels, MySQL: 4 levels)
  - Query hints (PostgreSQL: pg_hint_plan, MySQL: index hints)
- **Engine Interface**: Abstraction layer for database-specific behavior
- CLI `--database` flag for database selection override
- 45+ new tests for engine implementations

### Changed
- CLI now uses engine abstraction instead of hardcoded SQLite logic
- All engines implement standardized `Engine` interface
- Improved test coverage for engine package

## [0.4.0] - 2026-02-05

### Added
- Parameter type override via SQL comments (`-- @param name: type`)
- Engine interface boundaries for multi-database abstraction

### Changed
- Separated dialect-specific logic from core using engine interface

## [0.3.0] - 2026-02-02

### Added
- Pointer nullables (`emit_pointers_for_null` config option)
- JSON tag control (`emit_json_tags` config option)
- Deterministic caching for incremental builds (~20ms target achieved)
- Performance benchmark suite

## [0.2.0] - 2026-02-01

### Added
- Emit empty slices option
- Better parameter name inference (forward equality, BETWEEN, HAVING)
- Detailed diagnostics with multi-character underlines
- Comprehensive documentation (schema, query, generated-code references)

### Changed
- Column ordering determinism

## [0.1.0] - 2026-01-22

### Added
- Initial release
- Config parsing (TOML)
- Schema catalog with SQLite DDL support
- Query analysis with CTEs, aggregates, JOINs
- Go code generation
- CLI with dry-run mode
- sqlc config migration tool
- Prepared queries support

[Unreleased]: https://github.com/ElecTwix/db-catalyst/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/ElecTwix/db-catalyst/releases/tag/v0.5.0
[0.4.0]: https://github.com/ElecTwix/db-catalyst/releases/tag/v0.4.0
[0.3.0]: https://github.com/ElecTwix/db-catalyst/releases/tag/v0.3.0
[0.2.0]: https://github.com/ElecTwix/db-catalyst/releases/tag/v0.2.0
[0.1.0]: https://github.com/ElecTwix/db-catalyst/releases/tag/v0.1.0
