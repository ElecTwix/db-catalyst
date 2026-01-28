# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- SchemaParser interface for multi-dialect support
- MemoryWriter and MemoryResolver for testing
- Generator interface for testability
- Caching layer for incremental builds
- Pipeline hooks system for extensibility
- Logger interface abstraction
- Fuzz testing for tokenizer, schema parser, query parser
- Benchmark regression testing
- Integration test suite
- GitHub Actions CI/CD pipeline
- Security scanning (gosec, govulncheck)
- Task-based build system (Taskfile.yml)

### Changed
- Migrated from Make to Task (taskfile.dev)
- Improved test coverage to 58.8%

### Fixed
- Fixed broken test files
- Fixed CGO for race detection
- Fixed lint issues (G304, G306)

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

[Unreleased]: https://github.com/ElecTwix/db-catalyst/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ElecTwix/db-catalyst/releases/tag/v0.1.0
