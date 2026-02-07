# AGENTS

Repository for db-catalyst - language-agnostic SQL code generator. Clarity over cleverness.

## Toolchain

- **Go**: 1.25.3+
- **Task Runner**: `go install github.com/go-task/task/v3/cmd/task@latest`
- **Linter**: golangci-lint (configured in .golangci.yml)
- **Formatter**: gofmt + goimports

## Build Commands

```bash
task quick                                          # Quick build + test
go build -o db-catalyst ./cmd/db-catalyst           # Build CLI
go build -ldflags="-s -w -trimpath" -o db-catalyst ./cmd/db-catalyst  # Release
go install ./cmd/db-catalyst                        # Install to $GOPATH/bin
```

## Test Commands

```bash
go test ./...                                       # Run all tests
go test ./internal/config -run TestLoadConfig -v    # Run single test
go test -race ./...                                 # Race detector
go test -coverprofile=coverage.out ./...            # Coverage
UPDATE_GOLDEN=1 go test ./internal/pipeline -v      # Update goldens
```

## Benchmark Commands (Local Only)

Since this is a single-person project, benchmarks run locally rather than in CI.

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkTokenizer -benchmem ./internal/schema/tokenizer

# Save benchmark results with date stamp
go test -bench=. -benchmem -count=5 ./... > bench-$(date +%Y%m%d).txt

# Compare benchmarks (install benchstat first: go install golang.org/x/perf/cmd/benchstat@latest)
benchstat old.txt new.txt
```

### Key Benchmarks

- `BenchmarkPipeline` - Full pipeline execution time
- `BenchmarkTokenizer` - SQL tokenization performance
- `BenchmarkSchemaParser` - DDL parsing speed
- `BenchmarkQueryParser` - Query analysis performance

Run benchmarks before major changes to detect performance regressions.

## Lint Commands

```bash
task lint-all                   # All linters
golangci-lint run ./...         # Main linter
go vet ./...                    # Go vet
staticcheck ./...               # Static analysis
task security                   # gosec checks
```

## Code Style Guidelines

### Imports
- Group: stdlib / internal / external (enforced by goimports)
- Never use relative imports (e.g., `../pkg`)
- No blank lines between import groups manually

### Formatting
- Always run `gofmt` and `goimports` before committing
- Use `-trimpath` for release builds

### Naming Conventions
- **Packages**: lowercase, single word (`config`, `parser`, `codegen`)
- **Exported**: PascalCase (`ParseSchema`, `GenerateCode`)
- **Unexported**: camelCase (`parseImpl`, `validateInput`)
- **Acronyms**: Only first letter capitalized (`HTTPClient`, `JSONDecoder`)
- **Constants**: PascalCase (`DefaultTimeout`, `MaxRetries`)

### Function Signatures
- Context-first: `func (ctx context.Context, arg string) error`
- Error-last: `func DoSomething() (Result, error)`
- No named result parameters
- Options pattern for configuration:
  ```go
  func NewParser(opts ...ParserOption) *Parser
  func WithDialect(d string) ParserOption
  ```

### Error Handling
- Wrap with `%w`: `fmt.Errorf("parse schema: %w", err)`
- Use `errors.Join` for multiple errors (Go 1.20+)
- Define sentinel errors: `var ErrInvalidSchema = errors.New("invalid schema")`
- Check with `errors.Is()`, not string matching

### Types & Structs
- Prefer plain structs and slices
- Use Go 1.25 iter helpers only when clearer
- Generics for type safety, not overuse
- Avoid global state; inject dependencies

### Generated Code Standards
- Must look hand-written by experienced Go engineer
- All exported identifiers documented
- Descriptive parameter names (e.g., `userID` not `id`)
- JSON tags on all struct fields

## Testing Patterns

### Table-Driven Tests
```go
func TestParser(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Result
        wantErr bool
    }{
        {name: "simple", input: "...", want: expected},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !cmp.Equal(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Helpers
- Mark with `t.Helper()`
- Use `cmp.Diff` for comparisons
- Target 80%+ coverage for new code

## Multi-Database & Multi-Language Support

**Supported Databases:**
- **SQLite** (fully supported) - Primary dialect
- **PostgreSQL** - Type mapping and code generation using pgx/v5
- **MySQL** - Basic support

**Supported Languages:**
- **Go** (AST-based generation) - Default
- **Rust** (template-based with sqlx)
- **TypeScript** (template-based with pg)

Configure in `db-catalyst.toml`:
```toml
database = "postgresql"  # or "sqlite" (default), "mysql"
language = "rust"        # or "go" (default), "typescript"
```

## Project Architecture

```
cmd/              CLI entry points
internal/         Private packages
  codegen/        Code generation (go, rust, typescript)
  config/         Configuration parsing
  parser/         SQL parsers (grammars/, dialects/)
  pipeline/       Build pipeline
  query/          Query analysis
  schema/         Schema parsing
  transform/      Type transformations
docs/             Documentation
examples/         Example projects
```

## Resource Management

- Always use `defer` for cleanup
- Check context at operation starts: `if err := ctx.Err(); err != nil { return err }`
- Pass context as first parameter, propagate through all layers

## Determinism

- Sort maps/slices before writing files or comparing goldens
- Tests use `cmp.Diff` and golden fixtures

## Deterministic Caching

For faster incremental builds, db-catalyst supports caching parsed ASTs. Target: <200ms for small-to-medium projects.

```toml
[cache]
enabled = true
dir = ".db-catalyst-cache"  # Default: .db-catalyst-cache in project root
```

Cache invalidation is based on:
- File modification times
- Schema/query file hashes
- db-catalyst version

When enabled, the cache stores:
- Tokenized schema files
- Parsed schema ASTs
- Parsed query blocks
- Analyzed query results

Clear cache manually: `rm -rf .db-catalyst-cache`

## Commit Style

Follow Conventional Commits:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `refactor:` Code restructuring
- `test:` Tests
- `chore:` Maintenance

## Pre-Commit Checklist

- [ ] `task quick` passes (build + test)
- [ ] `task lint-all` passes
- [ ] `go vet ./...` clean
- [ ] Code formatted with goimports
- [ ] Tests added for new functionality
- [ ] Documentation updated if needed

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- `docs/idiomatic-go-guidelines.md` - Detailed style guide
