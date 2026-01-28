# Quality Improvement Tasks

This document tracks tasks to improve code quality, testing, and reliability.

## Task 1: Fix Existing Lint Issues

**Priority:** High  
**Effort:** Low  
**Impact:** High (clean build)

### Goal
Fix all existing golangci-lint issues.

### Issues to Fix

1. **internal/bench/bench_test.go:150** - G304: Potential file inclusion via variable
2. **internal/pipeline/e2e_test.go:98** - G304: Potential file inclusion via variable  
3. **internal/pipeline/hooks_test.go:130,133,136** - G306: WriteFile permissions
4. **internal/parser/dialects/parsers.go:40-42** - composites: unkeyed fields
5. **internal/parser/dialects/parsers.go:53-68** - structtag: participle tags
6. **internal/parser/grammars/grammar.go:19** - structtag: participle tags

### Implementation

For G304 (file inclusion), use filepath.Clean:
```go
// Before
data, err := os.ReadFile(path)

// After
data, err := os.ReadFile(filepath.Clean(path))
```

For G306 (permissions), use 0600 for test files:
```go
// Before
os.WriteFile(path, data, 0o644)

// After  
os.WriteFile(path, data, 0o600)
```

For participle tags, add `//nolint:govet` comments since these are DSL tags, not reflect tags.

---

## Task 2: Add Race Detection to CI

**Priority:** High  
**Effort:** Low  
**Impact:** High (catch concurrency bugs)

### Goal
Run tests with race detector in CI.

### Implementation

Add to Makefile:
```makefile
.PHONY: test-race
test-race:
	go test -race ./...
```

Add to GitHub Actions workflow:
```yaml
- name: Race Test
  run: go test -race ./...
```

---

## Task 3: Add Code Coverage Reporting

**Priority:** Medium  
**Effort:** Low  
**Impact:** Medium (visibility into test coverage)

### Goal
Track and report code coverage.

### Implementation

Add to Makefile:
```makefile
.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: coverage-report
coverage-report:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
```

Add coverage badge to README.

---

## Task 4: Add Static Analysis Tools

**Priority:** Medium  
**Effort:** Medium  
**Impact:** High (catch bugs early)

### Goal
Add additional static analysis beyond golangci-lint.

### Tools to Add

1. **go vet** - built-in analysis
2. **staticcheck** - advanced static analysis
3. **gosec** - security analysis (already in golangci-lint)
4. **errcheck** - unchecked errors
5. **ineffassign** - ineffectual assignments

### Implementation

Add to Makefile:
```makefile
.PHONY: lint-all
lint-all: lint vet staticcheck

.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck:
	which staticcheck || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
```

---

## Task 5: Add Fuzz Testing

**Priority:** Medium  
**Effort:** Medium  
**Impact:** High (find edge cases)

### Goal
Add fuzz tests for parsers and tokenizers.

### Implementation

Add fuzz test for tokenizer:
```go
// internal/schema/tokenizer/fuzz_test.go
func FuzzScan(f *testing.F) {
    f.Add("CREATE TABLE users (id INTEGER);")
    f.Add("SELECT * FROM users WHERE id = ?;")
    
    f.Fuzz(func(t *testing.T, input string) {
        _, _ = Scan("fuzz", []byte(input), true)
        // Should not panic
    })
}
```

Add fuzz test for schema parser:
```go
// internal/schema/parser/fuzz_test.go
func FuzzParse(f *testing.F) {
    f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY);")
    
    f.Fuzz(func(t *testing.T, input string) {
        tokens, _ := tokenizer.Scan("fuzz", []byte(input), true)
        _, _, _ = Parse("fuzz", tokens)
        // Should not panic
    })
}
```

---

## Task 6: Add Benchmark Regression Testing

**Priority:** Medium  
**Effort:** Medium  
**Impact:** Medium (prevent performance regressions)

### Goal
Track performance over time.

### Implementation

Add to Makefile:
```makefile
.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

.PHONY: bench-compare
bench-compare:
	go test -bench=. -benchmem -count=5 ./... > bench-new.txt
	# Compare with previous benchmark
```

Add benchmark for pipeline:
```go
// internal/pipeline/bench_test.go
func BenchmarkPipeline_Run(b *testing.B) {
    // Setup
    for i := 0; i < b.N; i++ {
        // Run pipeline
    }
}
```

---

## Task 7: Add Integration Test Suite

**Priority:** High  
**Effort:** Medium  
**Impact:** High (end-to-end confidence)

### Goal
Add more comprehensive integration tests.

### Implementation

Create `test/integration/` with:

1. **Test different schema types**
   - Simple tables
   - Tables with all constraint types
   - Views
   - Indexes
   - Foreign keys

2. **Test different query types**
   - SELECT with all join types
   - INSERT with RETURNING
   - UPDATE with RETURNING
   - DELETE with RETURNING
   - CTEs (recursive and non-recursive)

3. **Test edge cases**
   - Empty schema
   - Empty queries
   - Very long SQL
   - Unicode in identifiers
   - Reserved words as identifiers

---

## Task 8: Add Mutation Testing

**Priority:** Low  
**Effort:** High  
**Impact:** Medium (test quality)

### Goal
Verify test quality with mutation testing.

### Implementation

Use `github.com/zimmski/go-mutesting`:

```makefile
.PHONY: mutate
mutate:
	go-mutesting ./internal/...
```

---

## Task 9: Add Dead Code Detection

**Priority:** Low  
**Effort:** Low  
**Impact:** Low (cleanup)

### Implementation

```makefile
.PHONY: deadcode
deadcode:
	which deadcode || go install golang.org/x/tools/cmd/deadcode@latest
	deadcode ./...
```

---

## Task 10: Add Dependency Vulnerability Scanning

**Priority:** High  
**Effort:** Low  
**Impact:** High (security)

### Implementation

Use `govulncheck`:

```makefile
.PHONY: vulncheck
vulncheck:
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
```

Add to CI.

---

## Task 11: Add Code Complexity Limits

**Priority:** Medium  
**Effort:** Low  
**Impact:** Medium (maintainability)

### Implementation

Use `gocyclo`:

```makefile
.PHONY: complexity
complexity:
	which gocyclo || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 15 .
```

---

## Task 12: Add Spell Checking

**Priority:** Low  
**Effort:** Low  
**Impact:** Low (polish)

### Implementation

Use `codespell`:

```makefile
.PHONY: spellcheck
spellcheck:
	which codespell || pip install codespell
	codespell --skip="*.mod,*.sum,*.git,*.sql" .
```

---

## Implementation Order

1. **Task 1** - Fix existing lint issues (immediate)
2. **Task 2** - Race detection (immediate)
3. **Task 7** - Integration tests (high value)
4. **Task 10** - Vulnerability scanning (security)
5. **Task 5** - Fuzz testing (edge cases)
6. **Task 3** - Coverage reporting (visibility)
7. **Task 4** - Static analysis (quality)
8. **Task 6** - Benchmark regression (performance)
9. **Task 11** - Complexity limits (maintainability)
10. **Task 8** - Mutation testing (test quality)
11. **Task 9** - Dead code detection (cleanup)
12. **Task 12** - Spell checking (polish)

Each task should be committed separately.
