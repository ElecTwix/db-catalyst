# Comprehensive Improvement Plan

Based on deep code analysis, architecture review, and security audit.

## Critical Issues (Fix Immediately)

### 1. Remove All Panics (Security/Robustness)
**Files:**
- `internal/parser/dialects/parsers.go:102,150,198`
- `internal/parser/languages/graphql/parser.go:62`
- `internal/codegen/ast/builder.go:1122,1125,1129`
- `internal/codegen/ast/naming.go:159`

**Problem:** Panics crash the entire process.

**Solution:** Return errors instead:
```go
// Before
if err != nil {
    panic(fmt.Sprintf("failed to build parser: %v", err))
}

// After
if err != nil {
    return nil, fmt.Errorf("failed to build parser: %w", err)
}
```

---

### 2. Fix Unchecked Tokenizer Error (Error Handling)
**File:** `internal/query/analyzer/analyzer.go:183-186`

**Problem:** Tokenizer error is silently ignored.

**Solution:**
```go
tokens, err := tokenizer.Scan(q.Block.Path, []byte(q.Block.SQL), false)
if err != nil {
    addDiag(Diagnostic{
        Path:     q.Block.Path,
        Line:     q.Block.Line,
        Column:   q.Block.Column,
        Message:  fmt.Sprintf("tokenization failed: %v", err),
        Severity: SeverityError,
    })
    return Result{Diagnostics: diags}
}
```

---

### 3. Refactor Long Functions (Code Quality)
**Files:**
- `internal/pipeline/pipeline.go:147-501` (354 lines)
- `internal/codegen/ast/builder.go:633-938` (305 lines)

**Problem:** Functions are too long and do too much.

**Solution:** Extract into smaller methods (see detailed plan below).

---

## High Priority (Fix This Week)

### 4. Add File Size Limits (Security)
**File:** `internal/pipeline/pipeline.go`

```go
const maxFileSize = 100 * 1024 * 1024 // 100MB

func checkFileSize(path string) error {
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    if info.Size() > maxFileSize {
        return fmt.Errorf("file %s exceeds maximum size of %d bytes", path, maxFileSize)
    }
    return nil
}
```

---

### 5. Deduplicate Type Conversion (Code Smell)
**File:** `internal/query/analyzer/analyzer.go:2037-2084`

**Problem:** Two nearly identical functions.

**Solution:**
```go
var defaultTypeMapping = map[string]string{
    "INTEGER": "int64",
    "REAL":    "float64",
    "TEXT":    "string",
    "BLOB":    "[]byte",
    "NUMERIC": "string",
}

func SQLiteTypeToGo(sqliteType string) string {
    base := normalizeSQLiteType(sqliteType)
    if typ, ok := defaultTypeMapping[base]; ok {
        return typ
    }
    return "interface{}"
}

func (a *Analyzer) SQLiteTypeToGo(sqliteType string) string {
    if a.CustomTypes != nil {
        normalizedType := normalizeSQLiteType(sqliteType)
        if mapping, exists := a.CustomTypes[normalizedType]; exists {
            return mapping.GoType
        }
    }
    return SQLiteTypeToGo(sqliteType)
}
```

---

### 6. Add Pre-commit Hooks (Developer Experience)
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
      
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        types: [go]
        pass_filenames: false
```

---

## Medium Priority (Fix Next Week)

### 7. Add Configuration Validation
**File:** `internal/config/validate.go` (new)

```go
func (c *Config) Validate() error {
    if c.Package == "" {
        return errors.New("package is required")
    }
    if len(c.Schemas) == 0 {
        return errors.New("at least one schema file is required")
    }
    // ... more validation
}
```

---

### 8. Add Query Validation Against Schema
**Feature:** Validate queries reference existing tables/columns.

Example:
```sql
-- Error: table 'usrs' doesn't exist
SELECT * FROM usrs WHERE id = ?;
```

---

### 9. Improve Error Messages
Add file:line:column to all errors:
```
Before: parse failed: syntax error
After:  schema.sql:15:23: syntax error near "CREAT"
```

---

### 10. Add Example Projects
**Structure:**
```
examples/
├── basic/
├── complex/
└── prepared-queries/
```

---

## Low Priority (Nice to Have)

### 11. Add Watch Mode
```bash
db-catalyst --watch
```

### 12. Add Shell Completions
```bash
db-catalyst completion bash
```

### 13. Add SQL Formatting
Format SQL in generated code.

### 14. Add Performance Profiling
```yaml
  profile:
    desc: Run CPU and memory profiling
    cmds:
      - go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
```

---

## Implementation Order

### Phase 1: Critical (This Week)
1. Remove all panics
2. Fix unchecked tokenizer error
3. Add file size limits

### Phase 2: High Priority (Next Week)
4. Refactor long functions
5. Deduplicate type conversion
6. Add pre-commit hooks

### Phase 3: Medium Priority (Following Week)
7. Add config validation
8. Add query validation
9. Improve error messages
10. Add examples

### Phase 4: Low Priority (Future)
11-14. Nice-to-have features

---

## Current Status

| Metric | Value | Target |
|--------|-------|--------|
| Lint Issues | 0 | 0 ✅ |
| Test Coverage | 66.1% | 80% |
| Panics | 5 | 0 |
| Long Functions (>100 lines) | 3 | 0 |

---

## Estimated Effort

| Phase | Hours | Commits |
|-------|-------|---------|
| Phase 1 | 4-6 | 3 |
| Phase 2 | 8-12 | 4 |
| Phase 3 | 12-16 | 5 |
| Phase 4 | 8-12 | 4 |
| **Total** | **32-46** | **16** |
