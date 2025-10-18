# db-catalyst Specification

> **Design motto:** keep it simple, ship fast, embrace idiomatic Go. Every decision in this document prefers clarity over cleverness, straightforward pipelines over sprawling abstractions, and SQLite-only support at launch. Complexity is deferred until there is user demand and data to justify it.

## 1. Vision and Scope

`db-catalyst` is a focused SQL-to-Go code generator tailored exclusively for SQLite schemas and queries. It targets teams who want type-safe persistence code without dragging in the full breadth of `sqlc` features. The project intentionally limits itself to a minimal, understandable core:

- **Single engine:** SQLite only (using classic SQL syntax, no migration execution).
- **Single language:** idiomatic Go 1.25+ output (no plugins, no alternate languages).
- **Single driver assumption:** Go's `database/sql` API paired with `modernc.org/sqlite` or `github.com/mattn/go-sqlite3`—output must work seamlessly with either.
- **Single binary:** a compact CLI that expects schema + query files, produces Go packages, and exits.

Future database support (PostgreSQL, MySQL) is accepted as a possibility but explicitly out of scope for the first releases. The guiding principle: a small, teachable codebase that can deliver type-safe code for common CRUD workflows within minutes.

## 2. Goals

1. **Simplicity:** The codebase should be approachable by a single Go developer in a day. Components must be linear, naming obvious, and data structures transparent.
2. **Idiomatic Go:** Generated code should look like it was handwritten by an experienced Go engineer: context-first signatures, zero magic globals, narrow interfaces, and clear error handling.
3. **Low friction:** Minimal configuration, intuitive CLI defaults, and legible error messages. Users should get from SQL files to usable Go in a single command with sensible defaults.
4. **Speed:** Compilation must be fast, but not at the expense of clarity. Favor simple, single-pass parsers over complex runtime introspection as long as performance stays responsive on typical projects (< 5k lines SQL).
5. **Determinism:** Given the same inputs, the tool must produce byte-for-byte identical output. Determinism matters for reproducible builds and code review sanity.

Non-goals:
- Supporting SQL dialects beyond SQLite.
- Running migrations or connecting to live databases.
- Emitting code for other languages/runtimes.
- Implementing plugin systems or remote execution.

## 3. High-Level Architecture

The CLI orchestrates a four-stage pipeline:

1. **Configuration ingest:** Read a single TOML file, validate required fields, resolve file globs, and build an in-memory job plan.
2. **Schema catalog construction:** Tokenize and parse DDL statements (`CREATE TABLE`, `CREATE INDEX`, `CREATE VIEW`, simple `ALTER TABLE ADD COLUMN`). Produce a normalized catalog with tables, columns, primary keys, unique indexes, and foreign keys. No migrations run; everything is derived statically.
3. **Query analysis:** Parse query blocks, extract metadata (name, result cardinality, parameters, documentation comments), and resolve column information by matching against the catalog. Fail fast on ambiguities.
4. **Code generation:** Use Go's `go/ast` + `go/printer` to emit packages containing model structs, helper types, and query methods. Output files land in a configured directory, ready for `goimports`.

Each stage consumes immutable inputs and produces immutable outputs, facilitating straightforward testing and caching. Data flows through simple structs rather than interfaces until extension points are truly needed.

## 4. Configuration Specification

- **File expectations:** Default filename `db-catalyst.toml`. Overrideable via `--config` flag.
- **Format:** TOML only. No version numbers, no alternate encodings.
- **Schema:**

```toml
package = "bookstore"        # required, Go package name for generated code
out = "internal/db"          # required, output directory relative to config file
sqlite_driver = "modernc"    # optional, enum {"modernc", "mattn"}; defaults to modernc
schemas = ["schema/*.sql"]   # glob patterns; must resolve to at least one file
queries = ["query/*.sql"]

[prepared_queries]
# optional; defaults to disabled
enabled = true
metrics = true
thread_safe = true
```

- **Validation rules:**
  - `package` must be a valid Go identifier.
  - `out` must be a clean relative path; the tool creates it if missing.
  - `schemas` and `queries` must each resolve to at least one readable file. The tool surfaces missing files with friendly error messages listing attempted paths.
  - Optional `[prepared_queries]` table configures prepared statement generation:
    - `enabled` toggles output of the prepared wrapper (default `false`).
    - `metrics` emits duration/error hooks around each prepared call (default `false`).
    - `thread_safe` guards lazy prepare/close with per-query mutexes; when `false`, statements prepare eagerly during `Prepare` (default `false`).
  - Unknown keys trigger warnings (treated as errors when `--strict-config` is set).

No nested configuration (e.g., overrides, multiple packages) is supported in v0. Every configuration generates exactly one Go package.

## 5. Schema Parsing and Catalog Model

### 5.1 Tokenization

Implement a simple tokenizer that walks UTF-8 source, recognizing:
- Identifiers (supporting double-quoted identifiers and square brackets).
- Keywords (case-insensitive, stored uppercase for comparison).
- Literals (numeric, string, blob).
- Symbols (`(`, `)`, `,`, `;`, `.`, `*`, `=`).
- Comments (`--` line comments, `/* block comments */`). Comments are discarded but stored with line offsets for error reporting.

Tokenizer requirements:
- Provide line/column metadata for each token to support precise diagnostics.
- Strip comments yet optionally expose doc comments preceding `CREATE TABLE`.
- Operate on a single pass with zero allocations beyond token slice (use `[]Token` seeded with estimated capacity from len(source)/4).

### 5.2 Grammar Coverage

Support a subset of SQLite DDL:
- `CREATE TABLE [schema.]name ( column_def [, ...] [ table_constraint [, ...] ]) [WITHOUT ROWID];`
- `column_def := name type [column_constraint ...]`
- `type :=` one or two identifiers (e.g., `INTEGER`, `DOUBLE PRECISION`). Collect raw type string for mapping.
- Column constraints: `PRIMARY KEY`, `NOT NULL`, `DEFAULT literal`, `REFERENCES table(column)`.
- Table constraints: `PRIMARY KEY (column list)`, `UNIQUE (column list)`, `FOREIGN KEY (column list) REFERENCES table(column list)`.
- `CREATE UNIQUE INDEX` / `CREATE INDEX` on single table columns.
- Optional limited support for `CREATE VIEW` with stored SQL string (no attempt to parse the view body—store raw string for documentation; treat view columns like query metadata only when user queries them).
- `ALTER TABLE <table> ADD COLUMN <column_def>`.

Anything outside this subset triggers a descriptive error urging the user to simplify their schema files. The tool never attempts to handle triggers, virtual tables, FTS, or generated columns in v0.

### 5.3 Catalog Data Structures

Define small Go structs in `internal/schema`:

```go
type Catalog struct {
    Tables map[string]*Table
    Views  map[string]*View
}

type Table struct {
    Name        string
    Columns     []*Column
    PrimaryKey  *PrimaryKey // nil if none
    UniqueKeys  []*UniqueKey
    ForeignKeys []*ForeignKey
    WithoutRowID bool
}

type Column struct {
    Name       string
    Type       SQLiteType // raw normalized string; mapping deferred
    NotNull    bool
    Default    *Value
    References *ForeignKeyRef // optional single-column FK
}
```

All structs hold source position metadata for user-facing errors. Keep types plain—no interfaces or generics beyond Go's basic collections.

## 6. Query Parsing and Analysis

### 6.1 Query File Format

`query/*.sql` files adopt the familiar block structure:

```sql
-- name: GetAuthor :one
SELECT id, name FROM authors WHERE id = ?;

-- name: ListAuthors :many
SELECT id, name FROM authors ORDER BY name;

-- name: CreateAuthor :exec
INSERT INTO authors (name) VALUES (?);
```

Rules:
- Blocks must start with `-- name: IDENTIFIER :COMMAND`.
- `:COMMAND` is one of `:one`, `:many`, `:exec`, `:execresult`. No copy/batch semantics.
- Optional doc comments immediately preceding the `-- name` line are captured and surfaced in generated code as docstrings.
- Named parameters `:param` are allowed; positional parameters `?` and `?NNN` also supported. The analyzer maps them to Go method parameters in the order they appear (named params sorted by first appearance).

### 6.2 Parsing Strategy

Reuse the schema tokenizer to avoid duplication. Query parsing occurs in two passes:

1. **Block slicing:** Scan file for `-- name:` markers, capture the SQL span until the next marker or EOF.
2. **Statement analysis:** Tokenize the SQL body. Determine primary verb (SELECT/INSERT/UPDATE/DELETE). For `SELECT`, derive output columns; for DML, determine input parameters only.

Rather than full SQL parsing, rely on heuristics anchored on SQLite semantics:
- For `SELECT`, collect column expressions up to the `FROM`. Track explicit aliases using `AS` or implicit `columnName`. Joins and aliases are resolved against a scope that also includes earlier CTE definitions.
- CTE prologues (`WITH` / `WITH RECURSIVE`) are parsed ahead of the main statement; each CTE records its body SQL, column list, and diagnostics. Recursive terms must return the same column count as the anchor.
- Subqueries and expressions default to user-defined column names; if a column lacks an alias and we cannot infer a simple column name, raise a “missing column alias” diagnostic.
- ORDER BY/GROUP BY clauses are identified and preserved but do not affect metadata.

### 6.3 Parameter Extraction

As tokens stream, capture parameters:
- `?` or `?NNN`: assign sequential argument names `arg1`, `arg2`, etc., unless `-- dbcat:argname=name` directives override them.
- `:identifier`: use the identifier as Go parameter name (converted to `camelCase`). If duplicates appear, ensure consistent types.

For each parameter, store:
- Name (Go-friendly).
- Source position (for diagnostics).
- Inferred type hint: attempt to map using surrounding context (`column = ?` + column metadata). If inference fails, default to `interface{}` but emit a warning encouraging explicit casts or parameter hints, e.g., `CAST(? AS TEXT)`.

### 6.4 Result Column Resolution

For SELECT queries, build a slice of result columns:
- Each item contains `Name`, `Type`, `Nullable`, `Doc`.
- `Name` uses the alias if present; otherwise use raw column name after `table.` removal.
- `Type` derives from the catalog column (including columns referenced through CTEs) when resolved. Aggregates (`COUNT`, `SUM`, `MIN`, `MAX`, `AVG`) map to concrete Go types based on their operands, and require aliases so generated code surfaces stable identifiers. If inference still fails, mark as `interface{}` and warn.

Errors are fatal when:
- A referenced table/column is unknown.
- A column lacks alias and cannot be inferred.
- Mixed parameter naming styles create ambiguous method signatures.

## 7. Go Code Generation

### 7.1 Output Layout

Generated package contains:

- `models.go`: Go structs representing tables referenced by queries. Each struct contains exported fields with JSON tags by default (toggle via `--no-json-tags`). Fields use pointer/nullable types per column nullability.
- `querier.go`: Defines `type Querier interface` with methods per query, and a `type Queries struct { db DBTX }` implementation where `DBTX` is `interface{ ExecContext, QueryContext, QueryRowContext }`. Generated method signatures mirror analyzer metadata so CTE/aggregate outputs surface their concrete Go types.
- `query_<name>.go`: One file per query block containing the SQL string constant and method implementation. This keeps diff noise minimal.
- `_helpers.go`: Shared utilities for scanning rows, wrappers for optional transactions, and error helper `wrapError` (only if needed).

### 7.2 Method Signatures

- All methods accept `context.Context` as first argument.
- Input parameters follow, typed using Go primitive or `sql.Null*` equivalents aligned with column metadata.
- Return values reflect command type:
  - `:one`: `(Struct, error)`.
  - `:many`: `([]Struct, error)` with pre-sized slice if `LIMIT` is numeric.
  - `:exec`: `(sql.Result, error)`.
  - `:execresult`: `(QueryResult, error)` where `QueryResult` is generated struct exposing relevant metadata (rows affected, last insert ID) when necessary.

### 7.3 Implementation Details

- Use `stmt := rawQuery` string constants; no prepared statement caching in v0 (keep simple). Encourage users to wrap the generated code with their own prep layer if desired.
- For SELECT queries, method uses `rows, err := q.db.QueryContext(ctx, stmt, args...)` followed by `defer rows.Close()`. Scanning leverages generated functions `scanListAuthorsRow(rows *sql.Rows) (Author, error)` for clarity.
- Nullable columns map to `sql.Null*` types by default; optional config `emit_pointers_for_null` toggles pointer style (planned but not in first release).
- Error wrapping: simply `return err` without additional context. Keep stack surfaces simple. Provide a `WrapErr` hook in future if needed.

### 7.4 Formatting & Imports

- Build AST nodes with `go/ast` and `ast.GenDecl` to minimize manual string formatting.
- Run output through `go/printer` followed by `imports.Process` (from `golang.org/x/tools/imports`) to ensure canonical formatting. Handle errors gracefully.

## 8. Command-Line Interface

Binary name: `db-catalyst`.

Primary command:

```
db-catalyst generate [--config path] [--out dir]
```

Flags:
- `--config`: path to config file (default `db-catalyst.toml`).
- `--out`: override output directory; if absent, use config `out`.
- `--dry-run`: parse and analyze, but do not write files; prints summary.
- `--list-queries`: list discovered queries with input/output summary.
- `--strict-config`: treat unknown configuration keys as hard errors.
- `--verbose`: enable debug logging (errors show token positions by default).

Exit codes:
- `0`: success.
- `1`: configuration or parsing error.
- `2`: code generation failure (e.g., file write permissions).

All CLI interactions must produce succinct logs, using `log/slog` with human-friendly text handler by default. No spinners, no color unless environment variable opt-in.

## 9. Error Handling Strategy

- Errors bubble with contextual messages including file path, line, column, and a short hint. Example: `query/users.sql:12:8: unknown column email on table users (did you mean email_address?)`.
- During schema parsing, accumulate multiple errors where feasible (e.g., missing columns in a table) before aborting; keep the list short to avoid overwhelming the user.
- CLI prints errors to stderr and exits; no panic.

## 10. Performance Expectations

- Target generation time under 200ms for small projects (<25 queries) and under 2s for moderate projects (<200 queries, <5000 lines SQL). Keep benchmarks in `internal/bench` to track regressions.
- Memory footprint should stay below 100MB even on large inputs; avoid storing entire file lists in multiple formats.
- Cache parsed schema tokens per file so that reruns only re-tokenize modified files (optional enhancement guarded behind simple hash-based cache in `~/.cache/db-catalyst`). Cache is not part of v0 but should be easy to add given the stateless architecture.

## 11. Testing Strategy

- Unit tests for tokenizer, schema parser, and query analyzer with golden files.
- End-to-end tests using small fixtures under `testdata/`: run `db-catalyst generate`, format output, compare to expected `.golden` Go files.
- Use Go 1.25+ `testing/synctest` where applicable to deterministically test concurrent file writes (if concurrency added later).
- Integrate `go vet`, `staticcheck`, and `golangci-lint` focusing on iterator usage and error handling (drives code quality without bloating runtime code).

## 12. Documentation Deliverables

- `README.md`: quickstart, installation, minimal config example.
- `docs/schema.md`: supported SQLite syntax, with examples and limitations.
- `docs/query.md`: query block format, parameter rules, alias requirements.
- `docs/generated-code.md`: overview of produced files and integration tips.
- `docs/roadmap.md`: track upcoming features (nullable pointer options, caching, additional drivers).

All docs must repeat the simplicity mantra: small scope, SQLite-only, Go-first.

## 13. Roadmap (Post-v0)

1. **v0.1.0** – Initial release: config parsing, schema catalog, query analysis, Go generation, CLI.
2. **v0.2.0** – Quality polish: better parameter name inference, detailed diagnostics, improved docs.
3. **v0.3.0** – Optional features: pointer nullables, JSON tag toggles, deterministic caching.
4. **v0.4.0** – Extensibility groundwork: introduce engine interface boundaries to explore PostgreSQL without rewriting the pipeline.

Each release must keep the core simple—prefer additive flags over sweeping refactors. When complexity creeps in, reassess requirements before merging.

## 14. Contribution Guidelines (Internal)

- Pull requests limited to one focused change.
- All new functionality must include end-to-end fixture updates and unit tests.
- Keep third-party dependencies minimal: standard library + `golang.org/x/tools/imports`. Avoid heavy ORMs or parser generators.
- Follow `gofmt` and idiomatic Go naming conventions. Use `context.Context` first parameter, `error` last return value.
- Document exported identifiers with concise comments. Avoid gratuitous helper abstractions.

## 15. Summary

`db-catalyst` deliberately chooses the narrow path: a SQLite-only, Go-only code generator that favors explicitness, deterministic output, and a codebase small enough to reason about in an afternoon. By concentrating on a single database and language, we can deliver a tool that developers trust, extend, and debug without feeling overwhelmed. Simplicity is the feature; every spec detail above enforces that ethos. Future growth is welcome, but not at the expense of the core promise: keep it simple, keep it idiomatic, keep shipping.
