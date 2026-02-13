# db-catalyst Context

## Project Overview

`db-catalyst` is a focused SQL-to-Go code generator tailored exclusively for SQLite schemas and queries. It aims to provide type-safe persistence code without the complexity of larger tools like `sqlc`.

**Core Principles:**
*   **Single Engine:** SQLite only.
*   **Single Language:** Idiomatic Go 1.25+.
*   **Simplicity:** Minimal configuration, understandable codebase.
*   **Determinism:** Byte-for-byte identical output for the same inputs.

## Architecture

The CLI follows a four-stage pipeline:
1.  **Configuration Ingest:** Reads `db-catalyst.toml` to build an in-memory job plan.
2.  **Schema Catalog Construction:** Tokenizes and parses DDL statements to build a normalized catalog of tables and constraints. No database connection or migrations are involved.
3.  **Query Analysis:** Parses query blocks, extracts metadata, and resolves columns against the catalog.
4.  **Code Generation:** Emits Go packages using `go/ast` and `go/printer`.

## Directory Structure

*   `cmd/db-catalyst/`: Main CLI entry point.
*   `internal/`: Core application logic.
    *   `cli/`: CLI options and flags.
    *   `config/`: Configuration loading and validation.
    *   `schema/`: DDL parsing and catalog construction.
    *   `query/`: Query parsing and analysis.
    *   `codegen/`: Go code generation logic.
    *   `pipeline/`: Orchestration of the generation stages.
*   `docs/`: Project documentation and plans.
    *   `plans/`: Implementation plans and status.
*   `testdata/`: Golden files and fixtures for testing.

## Building and Running

*   **Install:** `go install ./cmd/db-catalyst`
*   **Run:** `db-catalyst generate` (assumes `db-catalyst.toml` exists)
*   **Test:** `make test` or `go test ./...`
*   **Lint:** `make lint` (requires `golangci-lint`)

## Configuration (`db-catalyst.toml`)

A typical configuration looks like this:

```toml
package = "bookstore"
out = "internal/db"
schemas = ["schema/*.sql"]
queries = ["query/*.sql"]

[prepared_queries]
enabled = true
```

## Development Conventions

*   **Idiomatic Go:** Code should look handwritten. Context is always the first argument.
*   **Testing:** Comprehensive unit tests and end-to-end golden file tests in `testdata`.
*   **No Magic:** Explicit names and types. No hidden globals.
*   **Error Handling:** Errors bubble up with contextual information (file, line, column).
