# db-catalyst

[![CI](https://github.com/ElecTwix/db-catalyst/actions/workflows/ci.yml/badge.svg)](https://github.com/ElecTwix/db-catalyst/actions)
[![Security](https://github.com/ElecTwix/db-catalyst/actions/workflows/security.yml/badge.svg)](https://github.com/ElecTwix/db-catalyst/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/ElecTwix/db-catalyst)](https://goreportcard.com/report/github.com/ElecTwix/db-catalyst)
[![Go Version](https://img.shields.io/badge/go%20version-%3E=1.25-61CFDD.svg)](https://golang.org/doc/go1.25)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`db-catalyst` turns SQL schemas and query files into deterministic, idiomatic Go 1.25+ persistence packages. The CLI keeps configuration lightweight while producing code that looks hand-written: context-first signatures, descriptive names, and zero hidden globals.

## Supported Databases

- **SQLite** (default) - Full support with modernc.org/sqlite or mattn/go-sqlite3
- **PostgreSQL** - Type mapping and code generation using pgx/v5
- **MySQL** - Basic support (proof of concept)

Configure the database in your `db-catalyst.toml`:

```toml
# For SQLite (default)
database = "sqlite"

# For PostgreSQL
database = "postgresql"

# For MySQL
database = "mysql"
```

## Requirements

- Go 1.25.3 or newer
- `goimports`: install with `go install golang.org/x/tools/cmd/goimports@latest`

## Quick Start

```bash
# Install the CLI (binary lands in $(go env GOBIN) or GOPATH/bin)
go install ./cmd/db-catalyst

# View CLI usage
db-catalyst --help

# Run the smoke test suite
make test

# Run all tests (add the race detector when touching concurrency)
go test ./...
go test -race ./...

# Execute a focused package test
go test ./internal/config -run TestLoadConfig
```

Project planning lives in [`db-catalyst-spec.md`](db-catalyst-spec.md) and `docs/`.

## PostgreSQL Support

db-catalyst now supports PostgreSQL with the pgx/v5 driver:

```toml
# db-catalyst.toml
package = "mydb"
out = "db"
database = "postgresql"
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]
```

### PostgreSQL Types

The following PostgreSQL types are mapped to Go types:

| PostgreSQL Type | Go Type | Package |
|----------------|---------|---------|
| `UUID` | `uuid.UUID` | github.com/google/uuid |
| `TEXT`, `VARCHAR` | `pgtype.Text` | github.com/jackc/pgx/v5/pgtype |
| `INTEGER`, `INT` | `pgtype.Int4` | github.com/jackc/pgx/v5/pgtype |
| `BIGINT` | `pgtype.Int8` | github.com/jackc/pgx/v5/pgtype |
| `BOOLEAN` | `pgtype.Bool` | github.com/jackc/pgx/v5/pgtype |
| `TIMESTAMPTZ` | `pgtype.Timestamptz` | github.com/jackc/pgx/v5/pgtype |
| `NUMERIC`, `DECIMAL` | `*decimal.Decimal` | github.com/shopspring/decimal |
| `JSONB` | `[]byte` | - |
| Arrays (e.g., `TEXT[]`) | `pgtype.Text` | github.com/jackc/pgx/v5/pgtype |

See the [PostgreSQL example](examples/postgresql/) for a complete working example with UUIDs, JSONB, arrays, and more.

## Migrating from sqlc

Coming from sqlc? Check out our comprehensive [migration guide](docs/migrating-from-sqlc.md) for step-by-step instructions to convert your existing sqlc projects to db-catalyst.

## Feature Flags

- [`docs/feature-flags.md`](docs/feature-flags.md) documents the configuration surface, including the `[prepared_queries]` toggles for metrics and thread-safe statement initialization.

## Documentation

### Core Documentation

- **[Schema Reference](docs/schema.md)**: Complete guide to writing SQL schemas, including data types, constraints, indexes, and foreign keys.
- **[Query Reference](docs/query.md)**: How to write SQL queries with annotations, parameters, JOINs, CTEs, and advanced features.
- **[Generated Code Reference](docs/generated-code.md)**: Understanding the generated Go code, interfaces, transactions, and usage patterns.

### Additional Documentation

- [`db-catalyst-spec.md`](db-catalyst-spec.md): high-level architecture, pipeline stages, and roadmap.
- [`docs/plans/db-catalyst-implementation-plan.md`](docs/plans/db-catalyst-implementation-plan.md): milestone-by-milestone plan and validation matrix.
- [`docs/feature-flags.md`](docs/feature-flags.md): runtime and codegen switches available in configuration.
- [`docs/migrating-from-sqlc.md`](docs/migrating-from-sqlc.md): guide for migrating existing sqlc projects.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file.
