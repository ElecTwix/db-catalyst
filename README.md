# db-catalyst

`db-catalyst` turns SQLite schemas and query files into deterministic, idiomatic Go 1.25+ persistence packages. The CLI keeps configuration lightweight while producing code that looks hand-written: context-first signatures, descriptive names, and zero hidden globals.

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

## Feature Flags

- [`docs/feature-flags.md`](docs/feature-flags.md) documents the configuration surface, including the `[prepared_queries]` toggles for metrics and thread-safe statement initialization.

## Documentation Map

- [`db-catalyst-spec.md`](db-catalyst-spec.md): high-level architecture, pipeline stages, and roadmap.
- [`docs/plans/db-catalyst-implementation-plan.md`](docs/plans/db-catalyst-implementation-plan.md): milestone-by-milestone plan and validation matrix.
- [`docs/feature-flags.md`](docs/feature-flags.md): runtime and codegen switches available in configuration.
