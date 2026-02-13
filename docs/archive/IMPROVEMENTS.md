# Additional Improvements for db-catalyst

## Critical Issues

### 1. Fix CGO for Race Detection
**Issue:** `go test -race` requires CGO_ENABLED=1 but Taskfile sets CGO_ENABLED=0

**Fix:** Update Taskfile.yml:
```yaml
env:
  CGO_ENABLED: '1'  # Required for race detector
```

Or create separate env for race tests:
```yaml
tasks:
  test-race:
    desc: Run tests with race detector
    env:
      CGO_ENABLED: '1'
    cmds:
      - go test -race ./...
```

---

## High Priority

### 2. Improve Test Coverage (58.8% → 80%+)

**Current:** 58.8%  
**Target:** 80%+

**Low coverage packages:**
- `internal/codegen/ast` - no tests
- `internal/codegen/render` - no tests
- `internal/schema/model` - no tests
- `internal/query/analyzer` - ~45%
- `internal/pipeline` - ~60%

**Action:** Add unit tests for untested packages.

---

### 3. Add GitHub Actions CI Workflow

**Missing:** `.github/workflows/ci.yml`

**Create:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go test ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: golangci/golangci-lint-action@v3
        with:
          version: latest

  race:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test -race ./...
```

---

### 4. Add Release Automation

**Create:** `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: task build-all
      - uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Medium Priority

### 5. Add Documentation Website

**Options:**
- GitHub Pages with MkDocs
- Read the Docs
- Simple markdown in `docs/`

**Structure:**
```
docs/
├── getting-started.md
├── configuration.md
├── sql-queries.md
├── generated-code.md
├── migration-from-sqlc.md
├── architecture.md
└── api/
    ├── pipeline.md
    ├── parser.md
    └── codegen.md
```

---

### 6. Add Example Projects

**Create:** `examples/` directory

```
examples/
├── basic/
│   ├── db-catalyst.toml
│   ├── schema.sql
│   ├── queries.sql
│   └── README.md
├── complex/
│   ├── db-catalyst.toml
│   ├── schema.sql
│   ├── queries.sql
│   └── README.md
└── prepared-queries/
    ├── db-catalyst.toml
    ├── schema.sql
    ├── queries.sql
    └── README.md
```

---

### 7. Add Docker Support

**Create:** `Dockerfile`

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o db-catalyst ./cmd/db-catalyst

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/db-catalyst /usr/local/bin/
ENTRYPOINT ["db-catalyst"]
```

**Create:** `docker-compose.yml` for development

---

### 8. Improve Error Messages

**Current:** Some errors lack context

**Target:** File:line:column for all errors

Example:
```go
// Before
return fmt.Errorf("parse failed: %w", err)

// After
return fmt.Errorf("%s:%d:%d: parse failed: %w", path, line, col, err)
```

---

### 9. Add Configuration Validation

**Create:** `internal/config/validate.go`

```go
func (c *Config) Validate() error {
    if c.Package == "" {
        return errors.New("package is required")
    }
    if len(c.Schemas) == 0 {
        return errors.New("at least one schema is required")
    }
    // ... more validation
}
```

---

### 10. Add Query Validation Against Schema

**Current:** Basic query parsing
**Target:** Validate queries match schema (table/column existence)

Example:
```sql
-- Should error: table 'usrs' doesn't exist
SELECT * FROM usrs WHERE id = ?;

-- Should error: column 'emial' doesn't exist
SELECT emial FROM users;
```

---

## Low Priority / Nice to Have

### 11. Add Watch Mode

**Feature:** Auto-regenerate on file changes

```bash
db-catalyst --watch
```

Uses `fsnotify` to watch schema/query files.

---

### 12. Add LSP (Language Server Protocol)

**Feature:** IDE support for .sql files

- Autocomplete table/column names
- Go-to-definition for queries
- Error diagnostics

---

### 13. Add Plugin System

**Feature:** Allow custom code generators

```toml
[plugins]
custom_generator = "./my-generator.so"
```

---

### 14. Add SQL Formatting

**Feature:** Format SQL in generated code

```sql
-- Before
SELECT id,name,email FROM users WHERE id=? AND active=1;

-- After
SELECT id, name, email
FROM users
WHERE id = ?
  AND active = 1;
```

---

### 15. Add Schema Diff/Migration Generation

**Feature:** Generate migrations from schema changes

```bash
db-catalyst diff schema-old.sql schema-new.sql
# Outputs: ALTER TABLE ...
```

---

## Implementation Priority

### Phase 1 (This Week)
1. Fix CGO for race detection
2. Add GitHub Actions CI
3. Improve coverage for critical packages

### Phase 2 (Next Week)
4. Add release automation
5. Create example projects
6. Add configuration validation

### Phase 3 (Future)
7. Documentation website
8. Docker support
9. Watch mode
10. LSP (if there's demand)

---

## Quick Wins (1-2 hours each)

- [ ] Fix CGO in Taskfile
- [ ] Add CI workflow
- [ ] Add CODE_OF_CONDUCT.md
- [ ] Add CONTRIBUTING.md
- [ ] Add LICENSE file
- [ ] Add CHANGELOG.md
- [ ] Add issue/PR templates
- [ ] Add badges to README
