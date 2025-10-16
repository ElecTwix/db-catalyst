# db-catalyst Implementation & Validation Plan

## Guiding Principles

- Uphold the spec motto: prefer clarity, SQLite-only scope, idiomatic Go 1.25+ output.
- Build each stage as an immutable, testable package with zero hidden globals.
- Lean on new Go 1.25 facilities (iterators, slices/maps helpers, telemetry toggles) when they improve readability without magic.
- Keep deterministic behavior front-of-mind: stable ordering, reproducible file writes, fixed seeds in tests.
- Seed observability and tooling (lint, vet, staticcheck) early so validation is habitual.

## Milestone Flow (each milestone ends with passing tests and docs updates)

1. **Project Skeleton** – Module setup, CI scaffolding, shared test fixtures, linters, docs baseline.
2. **Configuration Ingest** – Parse TOML, resolve globs, validate config, expose pipeline job description.
3. **Schema Catalog** – Tokenize, parse DDL subset, build catalog structs with diagnostics.
4. **Query Analysis** – Slice query blocks, infer params/results, reconcile with catalog, emit warnings.
5. **Code Generation** – Assemble AST, render Go files, ensure deterministic formatting/imports.
6. **CLI & E2E** – Wire command, dry-run/list-queries flows, end-to-end golden tests, docs refresh.

## Package-by-Package Plan

### cmd/db-catalyst

#### Responsibilities
- Main entrypoint; wires CLI flags to internal pipeline.
- Provides structured logging and telemetry toggles; sets adaptive `runtime` defaults if needed.

#### Implementation Steps
- Scaffold `main.go` with `cobra`-free flag parsing (std `flag` + helpers).
- Build command graph: `generate`, `--dry-run`, `--list-queries`.
- Invoke pipeline orchestrator with context cancellation and slog logger.

#### Tests & Validation
- Use `testing` with table cases invoking `main()` via `testmain` harness.
- Add CLI golden tests under `testdata/cli` capturing stdout/stderr (use `synctest` when concurrency emerges).
- Success when CLI invocations behave deterministically across flags and propagate exit codes per spec.

### internal/cli

#### Responsibilities
- Shared flag parsing helpers, slog setup, exit code mapping.

#### Implementation Steps
- Define `Options` struct, parse from args, validate, convert to pipeline config.
- Provide human-friendly logging via `slog.NewTextHandler`.

#### Tests & Validation
- Unit tests covering flag precedence, defaulting, error messages; use `cmp.Diff` to compare structs.
- Success when every flag combination returns expected `Options` and diagnostics include arg positions.

### internal/config

#### Responsibilities
- Load TOML, validate schema per spec, resolve globs into ordered file lists.

#### Implementation Steps
- Use `encoding/toml` or `github.com/pelletier/go-toml`? Prefer std-compatible parser (e.g., `github.com/pelletier/go-toml/v2`).
- Normalize paths relative to config file using `os.Root` (Go 1.25) when running under sandbox contexts.
- Detect unknown keys; support `--strict-config` toggles via options.
- Expose `JobPlan` struct consumed by downstream stages.

#### Tests & Validation
- Table-driven tests with fixtures in `testdata/config`; include error golden files.
- Property-style test ensuring glob ordering uses `slices.SortStableFunc` and remains deterministic.
- Success when invalid configs surface precise errors with file/line references and valid configs produce canonicalized `JobPlan`.

### internal/fileset

#### Responsibilities
- Resolve glob patterns, ensure readable files, de-duplicate, provide deterministic order.

#### Implementation Steps
- Utilize `fs.WalkDir` and `iter.Seq` to stream matches, collecting via `slices.Collect`.
- Cache file metadata (modtime, size) for optional future incremental builds.

#### Tests & Validation
- Fake filesystem tests using `fstest.MapFS`; ensure globbing matches spec.
- Success when `schemas` and `queries` each produce non-empty ordered slices or descriptive errors.

### internal/schema/tokenizer

#### Responsibilities
- Convert schema SQL into token sequences with positions, comment capture, minimal allocations.

#### Implementation Steps
- Implement single-pass scanner using rune iteration; pre-size `[]Token` capacity with `len(src)/4` heuristic.
- Emit doc comments tied to following `CREATE` statements.
- Provide iterator-friendly API: `func (t *Tokenizer) Seq() iter.Seq2[int, Token]` for streaming analysis.

#### Tests & Validation
- Unit tests covering identifiers, string literals, comments, UTF-8 edge cases.
- Benchmark tests (`testing.B.Loop`) verifying performance budgets; assert no heap growth via `-benchmem`.
- Success when tokens match expected snapshot and position info is precise to column level.

### internal/schema/parser

#### Responsibilities
- Parse token stream into AST for supported DDL subset; report multiple errors per file when possible.

#### Implementation Steps
- Use Pratt or recursive-descent parsing tailored to subset; integrate `Unique` handles for IDs.
- Translate AST into `internal/schema/model` structures.
- Accumulate diagnostics with context hints.

#### Tests & Validation
- Golden tests comparing parsed AST (serialized via `spew` or bespoke formatter) vs expected.
- Negative tests capturing error aggregation.
- Success when all supported constructs parse, unsupported syntax yields actionable diagnostics.

### internal/schema/model

#### Responsibilities
- Define `Catalog`, `Table`, `Column`, constraint structs with source spans.

#### Implementation Steps
- Implement builder functions to convert parser output, enforce invariants (e.g., unique column names).
- Provide lookup helpers using `maps.Clone`/`maps.Values` for deterministic iteration.

#### Tests & Validation
- Unit tests verifying builder rejects invalid states, handles `WITHOUT ROWID`, FKs.
- Success when catalog exposes stable maps/slices consumable by query analysis.

### internal/query/block

#### Responsibilities
- Slice query files into named blocks; retain doc comments and SQL body ranges.

#### Implementation Steps
- Use tokenizer to find `-- name:` markers; preserve whitespace for code generation.
- Attach file path and span metadata for diagnostics.

#### Tests & Validation
- Unit tests using multi-query fixtures, covering doc comment capture, EOF handling.
- Success when every block returns name, command, raw SQL, doc comment.

### internal/query/parser

#### Responsibilities
- Tokenize block SQL, determine verb, extract column expressions, parameters.

#### Implementation Steps
- Reuse schema tokenizer to avoid duplication.
- Implement heuristics for SELECT column extraction, alias resolution, join handling.
- Map named parameters to Go identifiers (camel case) and positional parameters to sequential `argN`.

#### Tests & Validation
- Golden tests for parameter detection, alias inference, ambiguous cases.
- Property tests verifying consistent ordering of parameters using `quick.Check`.
- Success when analyzer receives structured query metadata with zero ambiguous cases.

### internal/query/analyzer

#### Responsibilities
- Reconcile query metadata with schema catalog; infer result types/nullable flags; emit warnings/errors.

#### Implementation Steps
- Build resolution pipeline using iterator functions (`iter.Pull`) to match columns to tables.
- Apply SQLite affinity mapping table to produce Go types; fall back to `interface{}` with warnings.
- Track diagnostics in `[]Diagnostic` with severity levels.

#### Tests & Validation
- Unit tests covering happy paths and failure cases (unknown columns, missing aliases, mixed param styles).
- Golden warnings snapshot tests to ensure wording stability.
- Success when analyzer either returns fully-typed query descriptors or precise fatal diagnostics.

### internal/codegen/ast

#### Responsibilities
- Convert catalog + query descriptors into Go AST nodes, ready for rendering.

#### Implementation Steps
- Define templates using `go/ast` constructors; reuse helper functions to avoid duplication.
- Ensure struct field ordering deterministic via `slices.SortFunc` on column names.
- Provide toggles (e.g., `EmitJSONTags`) from config.

#### Tests & Validation
- Unit tests verifying AST structure using `ast.Inspect`; ensure tags/field types correct.
- Success when AST output, upon printing, equals golden Go sources for sample fixtures.

### internal/codegen/render

#### Responsibilities
- Format AST with `go/printer`, run through `imports.Process`, and write to disk.

#### Implementation Steps
- Implement writer that writes to temporary buffer, compares with existing file to avoid unnecessary updates.
- Support `--dry-run` by returning diff summary.
- Use `io/fs` safe writes with `atomic` rename to guarantee determinism.

#### Tests & Validation
- Integration tests that generate fixtures into temp dir and compare to `testdata/golden` using `cmp.Diff`.
- Success when repeated runs are byte-identical and file system operations are atomic.

### internal/pipeline

#### Responsibilities
- Orchestrate stages: config -> catalog -> analysis -> generation; manage diagnostics and logging.

#### Implementation Steps
- Define `Pipeline` struct with dependencies injected for testability.
- Compose stages using clear data structs; no global state.
- Expose `Run(ctx context.Context, job JobPlan) error`.

#### Tests & Validation
- End-to-end tests using sample project; run pipeline in dry-run and full modes.
- Use `testing/synctest` if any parallelism introduced for file writes.
- Success when pipeline returns first fatal diagnostic, aggregates warnings, and surfaces success metrics.

### internal/logging (optional helper)

#### Responsibilities
- Centralize slog setup, log attributes (timings, counts).

#### Implementation Steps
- Provide `NewLogger(verbose bool)` returning configured slog logger.
- Emit stage timing with `slog.Group`.

#### Tests & Validation
- Minimal tests ensuring handler respects verbosity; rely on integration tests otherwise.

## Testing & Validation Matrix

- **Unit Tests:** tokenizer, parsers, analyzers, config loader; run via `go test ./internal/...`.
- **Golden Tests:** query analysis, codegen; fixture directories under `testdata/<stage>` with `.golden` outputs.
- **End-to-End:** CLI invocations generating Go code compared against golden packages; include dry-run/list modes.
- **Benchmarks:** tokenizer, parser, analyzer using `testing.B.Loop`; track metrics in `internal/bench`.
- **Diagnostics Quality:** dedicated tests verifying error strings include file:line:column and hint text.
- **Determinism Checks:** run generator twice in tests, ensure identical outputs via hash comparison.
- **Static Analysis:** enable `go vet`, `staticcheck`, `golangci-lint` (iterate, waitgroup, hostport linters) in CI.

## Tooling & Operational Readiness

- **Go Toolchain:** Require Go 1.25; document need for `go install golang.org/x/tools/cmd/goimports@latest`.
- **Telemetry:** opt-in `go telemetry` reporting documented; default off with environment flag.
- **CI:** GitHub Actions workflow running unit tests, linters, end-to-end fixtures on Linux/macOS; include `GOEXPERIMENT=jsonv2` builds to catch future parser impacts.
- **Release Artifacts:** `goreleaser` or simple script to produce binaries; ensure reproducibility with `-trimpath`.
- **Docs Updates:** After each milestone, update relevant docs (`docs/schema.md`, `docs/query.md`, `docs/generated-code.md`, `docs/roadmap.md`).

## Success Criteria Summary

- Each milestone completes with green unit + integration tests, updated docs, and deterministic outputs.
- CLI delivers clear diagnostics with positional context and conforms to exit code contract.
- Generated code compiles with `go build` across both supported SQLite drivers without manual edits.
- Performance targets met: <200ms for small fixtures, <2s for moderate ones; benchmarks tracked in CI.
- Codebase remains dependency-light (std lib + imports formatter) and idiomatic, enabling onboarding in a day.
