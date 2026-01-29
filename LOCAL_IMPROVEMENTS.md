# Local Development Improvements

No CI needed - everything runs locally with `task`.

## Quick Wins (1-2 hours)

### 1. Add Pre-commit Hooks

**File:** `.pre-commit-config.yaml`

```yaml
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: go fmt ./...
        language: system
        types: [go]
        pass_filenames: false
      
      - id: go-vet
        name: go vet
        entry: go vet ./...
        language: system
        types: [go]
        pass_filenames: false
      
      - id: golangci-lint
        name: golangci-lint
        entry: golangci-lint run ./...
        language: system
        types: [go]
        pass_filenames: false
      
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        types: [go]
        pass_filenames: false
```

**Install:** `pip install pre-commit && pre-commit install`

---

### 2. Add Git Hooks (Alternative to pre-commit)

**File:** `.githooks/pre-commit`

```bash
#!/bin/bash
echo "Running pre-commit checks..."

task lint-all || exit 1
task test || exit 1

echo "All checks passed!"
```

**Enable:** `git config core.hooksPath .githooks`

---

### 3. Add Example Projects

**Structure:**
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

### 4. Add Configuration Validation

**File:** `internal/config/validate.go`

```go
package config

import (
    "errors"
    "fmt"
)

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
    if c.Package == "" {
        return errors.New("package is required")
    }
    
    if len(c.Schemas) == 0 {
        return errors.New("at least one schema file is required")
    }
    
    if c.Out == "" {
        return errors.New("output directory is required")
    }
    
    for i, ct := range c.CustomTypes {
        if ct.SQLType == "" {
            return fmt.Errorf("custom_types[%d]: sql_type is required", i)
        }
        if ct.GoType == "" {
            return fmt.Errorf("custom_types[%d]: go_type is required", i)
        }
    }
    
    return nil
}
```

---

### 5. Add Query Validation Against Schema

**Feature:** Validate that queries reference existing tables/columns.

Example:
```sql
-- Error: table 'usrs' doesn't exist
SELECT * FROM usrs WHERE id = ?;

-- Error: column 'emial' doesn't exist
SELECT emial FROM users;
```

---

### 6. Add Better Error Messages

**Current:** `parse failed: syntax error`

**Target:** `schema.sql:15:23: syntax error near "CREAT"`

Add file:line:column to all errors.

---

### 7. Add Shell Completions

**File:** `cmd/db-catalyst/completions.go`

```go
func generateCompletions(shell string) string {
    // Generate bash/zsh/fish completions
}
```

---

### 8. Add Man Page

**File:** `docs/db-catalyst.1`

```
.TH DB-CATALYST 1 "2026-01-29" "v0.2.0" "db-catalyst Manual"
.SH NAME
db-catalyst \- Generate type-safe Go code from SQL
```

---

### 9. Add Watch Mode

**Feature:** Auto-regenerate on file changes.

```bash
db-catalyst --watch
```

Uses `fsnotify` to watch schema/query files.

---

### 10. Add SQL Formatting

**Feature:** Format SQL in generated code.

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

## Medium Priority

### 11. Add More Integration Tests

Test edge cases:
- Very long SQL queries
- Unicode in identifiers
- Reserved words as identifiers
- Complex nested CTEs
- Window functions

---

### 12. Add Performance Profiling

**Task:** `task profile`

```yaml
  profile:
    desc: Run CPU and memory profiling
    cmds:
      - go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./internal/pipeline/
      - go tool pprof -svg cpu.prof > cpu.svg
```

---

### 13. Add Code Complexity Check

**Task:** `task complexity`

```yaml
  complexity:
    desc: Check code complexity
    cmds:
      - which gocyclo || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
      - gocyclo -over 15 .
```

---

### 14. Add Dead Code Detection

**Task:** `task deadcode`

```yaml
  deadcode:
    desc: Find dead code
    cmds:
      - which deadcode || go install golang.org/x/tools/cmd/deadcode@latest
      - deadcode ./...
```

---

### 15. Add Spell Checking

**Task:** `task spellcheck`

```yaml
  spellcheck:
    desc: Check spelling
    cmds:
      - which codespell || pip install codespell
      - codespell --skip="*.mod,*.sum,*.git,*.sql" .
```

---

## Implementation Priority

### Do First (High Value, Low Effort)
1. **Pre-commit hooks** - Catch issues before commit
2. **Example projects** - Help users get started
3. **Configuration validation** - Better error messages
4. **Better error messages** - File:line:column

### Do Second (Medium Value)
5. **Query validation** - Catch errors early
6. **Shell completions** - Better UX
7. **More integration tests** - Reliability

### Do Later (Nice to Have)
8. **Watch mode** - Convenience
9. **SQL formatting** - Polish
10. **Man page** - Documentation

---

## Current Status

✅ **Completed:**
- All lint issues fixed (0 issues)
- Test coverage: 66.1%
- Task-based build system
- Fuzz testing
- Benchmarks
- Integration tests
- Documentation (LICENSE, CHANGELOG, CONTRIBUTING)

🎯 **Next:** Pick from "Do First" list above
