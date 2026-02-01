# AGENTS

Repository for db-catalyst - language-agnostic SQL code generator. Clarity over cleverness.

## Toolchain

- **Go**: 1.25.3+
- **Task Runner**: `go install github.com/go-task/task/v3/cmd/task@latest`
- **Linter**: golangci-lint (configured in .golangci.yml)
- **Formatter**: gofmt + goimports (`go install golang.org/x/tools/cmd/goimports@latest`)

## Build Commands

```bash
# Quick build + test
task quick

# Build CLI binary
go build -o db-catalyst ./cmd/db-catalyst

# Build optimized release
go build -ldflags="-s -w -trimpath" -o db-catalyst ./cmd/db-catalyst

# Install to $GOPATH/bin
go install ./cmd/db-catalyst
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run single test (example)
go test ./internal/config -run TestLoadConfig -v

# Run with race detector
go test -race ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
go test -bench=. -benchmem ./...
```

## Lint Commands

```bash
# Run all linters
task lint-all

# Or individually:
golangci-lint run ./...      # Main linter
go vet ./...                  # Go vet
staticcheck ./...             # Static analysis
```

## Code Style Guidelines

### Imports
- Group: stdlib / internal / external (enforced by goimports)
- Never use relative imports (e.g., `../pkg`)
- No blank lines between import groups manually (goimports handles it)

### Formatting
- Always run `gofmt` and `goimports` before committing
- No manual spacing or tab tweaks
- Use `-trimpath` for release builds

### Naming Conventions
- **Packages**: lowercase, single word (e.g., `config`, `parser`, `codegen`)
- **Exported**: PascalCase (e.g., `ParseSchema`, `GenerateCode`)
- **Unexported**: camelCase (e.g., `parseImpl`, `validateInput`)
- **Acronyms**: Only first letter capitalized (e.g., `HTTPClient`, `JSONDecoder`)
- **Constants**: PascalCase (e.g., `DefaultTimeout`, `MaxRetries`)

### Function Signatures
- Context-first: `func (ctx context.Context, arg string) error`
- Error-last: `func DoSomething() (Result, error)`
- No named result parameters (avoid panic recovery)
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
- Update golden fixtures intentionally

## Project Architecture

- **Grammar-driven**: SQL dialects defined in `.grammar` files (EBNF-style)
- **Parser library**: Participle for LL(k) parsing
- **Primary dialect**: SQLite (fully supported)
- **Multi-language**: Rust, TypeScript generators use templates; Go uses AST
- **No global state**: Pipeline stages are immutable

### Key Directories
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
- Check context at long operation starts: `if err := ctx.Err(); err != nil { return err }`
- Don't create new contexts unless necessary
- Pass context as first parameter, propagate through all layers

## Determinism

- Sort maps/slices before writing files or comparing goldens
- Tests use `cmp.Diff` and golden fixtures
- Prepared queries are opt-in via config (`[prepared_queries]`)

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
- `docs/grammar-parser-poc.md` - Architecture details
