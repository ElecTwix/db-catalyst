# db-catalyst Improvement Tasks

This document tracks planned improvements to make the codebase more flexible and testable.

## Task 1: Abstract Schema Parser Interface

**Priority:** High  
**Effort:** Low  
**Impact:** High (enables PostgreSQL support)

### Goal
Define a `SchemaParser` interface to abstract SQLite-specific parsing, allowing future PostgreSQL/MySQL parsers to be plugged in without changing pipeline logic.

### Current State
Pipeline directly calls:
```go
tokens, scanErr := schematokenizer.Scan(schemaPath, contents, true)
parsedCatalog, schemaDiags, parseErr := schemaparser.Parse(schemaPath, tokens)
```

### Implementation

1. Define interface in `internal/schema/parser/parser.go`:
```go
type Parser interface {
    Parse(ctx context.Context, path string, content []byte) (*model.Catalog, []Diagnostic, error)
}
```

2. Create SQLite implementation:
```go
type sqliteParser struct{}

func (p *sqliteParser) Parse(ctx context.Context, path string, content []byte) (*model.Catalog, []Diagnostic, error) {
    tokens, err := tokenizer.Scan(path, content, true)
    if err != nil {
        return nil, nil, err
    }
    return Parse(path, tokens)
}
```

3. Add factory function:
```go
func NewParser(dialect string) (Parser, error)
```

4. Update `Pipeline` to accept parser via `Environment`:
```go
type Environment struct {
    FSResolver   func(string) (fileset.Resolver, error)
    Logger       *slog.Logger
    Writer       Writer
    SchemaParser schemaparser.Parser  // NEW
}
```

5. Update pipeline Run method to use injected parser instead of direct calls.

6. Ensure backward compatibility: if `SchemaParser` is nil, create default SQLite parser.

### Testing
- Unit test the interface with mock implementation
- Verify existing tests still pass
- Test that nil parser falls back to SQLite

### Files to Modify
- `internal/schema/parser/parser.go` - add interface and factory
- `internal/pipeline/pipeline.go` - use interface
- `internal/pipeline/pipeline_test.go` - add tests

---

## Task 2: Memory-Based Test Utilities

**Priority:** High  
**Effort:** Low  
**Impact:** High (faster, isolated tests)

### Goal
Create memory-based implementations of `Writer` and `Resolver` for unit testing without filesystem I/O.

### Implementation

1. Create `internal/pipeline/memwriter.go`:
```go
type MemoryWriter struct {
    Files map[string][]byte
}

func (m *MemoryWriter) WriteFile(path string, data []byte) error {
    if m.Files == nil {
        m.Files = make(map[string][]byte)
    }
    m.Files[path] = data
    return nil
}
```

2. Create `internal/fileset/memory.go`:
```go
type MemoryResolver struct {
    files map[string][]byte  // path -> content
}

func NewMemoryResolver(files map[string][]byte) *MemoryResolver
func (m *MemoryResolver) Resolve(patterns []string) ([]string, error)
func (m *MemoryResolver) ReadFile(path string) ([]byte, error)
```

3. Add helper for creating test pipelines:
```go
func NewTestPipeline(files map[string][]byte) *Pipeline
```

### Testing
- Test MemoryWriter captures all writes
- Test MemoryResolver pattern matching
- Use in existing pipeline tests

### Files to Create
- `internal/pipeline/memwriter.go`
- `internal/fileset/memory.go`
- `internal/pipeline/testhelper.go` (optional)

### Files to Modify
- `internal/pipeline/pipeline_test.go` - use memory implementations

---

## Task 3: Generator Interface

**Priority:** Medium  
**Effort:** Low  
**Impact:** Medium (better testability)

### Goal
Define a `Generator` interface to allow mocking in pipeline tests.

### Implementation

1. Define interface in `internal/codegen/generator.go`:
```go
type Generator interface {
    Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error)
}

// Ensure existing Generator implements interface
var _ Generator = (*Generator)(nil)
```

2. Update `Pipeline.Environment`:
```go
type Environment struct {
    FSResolver   func(string) (fileset.Resolver, error)
    Logger       *slog.Logger
    Writer       Writer
    SchemaParser schemaparser.Parser
    Generator    codegen.Generator  // NEW
}
```

3. Update pipeline to use injected generator.

4. Create mock for testing:
```go
type MockGenerator struct {
    Files []File
    Err   error
}

func (m *MockGenerator) Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error) {
    return m.Files, m.Err
}
```

### Testing
- Test pipeline with mock generator
- Verify error handling

### Files to Modify
- `internal/codegen/generator.go` - add interface
- `internal/pipeline/pipeline.go` - use interface
- `internal/pipeline/pipeline_test.go` - add mock tests

---

## Task 4: Caching Layer Interface

**Priority:** Medium  
**Effort:** Medium  
**Impact:** Medium (performance for incremental builds)

### Goal
Implement caching interface for parsed schemas and query ASTs to speed up incremental builds.

### Implementation

1. Define interface in `internal/cache/cache.go` (already exists, needs completion):
```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
}
```

2. Create memory cache implementation:
```go
type MemoryCache struct {
    mu    sync.RWMutex
    items map[string]cacheItem
}

type cacheItem struct {
    value     interface{}
    expiresAt time.Time
}
```

3. Create file-based cache for persistence:
```go
type FileCache struct {
    dir string
}
```

4. Update `Pipeline.Environment`:
```go
type Environment struct {
    FSResolver   func(string) (fileset.Resolver, error)
    Logger       *slog.Logger
    Writer       Writer
    SchemaParser schemaparser.Parser
    Generator    codegen.Generator
    Cache        cache.Cache  // NEW
}
```

5. Use cache in pipeline for:
   - Parsed schemas (key: schema file hash)
   - Query analyses (key: query file hash)

### Testing
- Test cache hit/miss
- Test TTL expiration
- Test concurrent access

### Files to Create/Modify
- `internal/cache/memory.go` - memory implementation
- `internal/cache/file.go` - file-based implementation
- `internal/pipeline/pipeline.go` - integrate caching

---

## Task 5: Pipeline Hooks System

**Priority:** Medium  
**Effort:** Medium  
**Impact:** Medium (extensibility)

### Goal
Add hook points in the pipeline for custom processing, validation, and metrics.

### Implementation

1. Define hooks in `internal/pipeline/hooks.go`:
```go
type Hooks struct {
    BeforeParse    func(ctx context.Context, schemaPaths []string) error
    AfterParse     func(ctx context.Context, catalog *model.Catalog) error
    BeforeAnalyze  func(ctx context.Context, queries []string) error
    AfterAnalyze   func(ctx context.Context, analyses []analyzer.Result) error
    BeforeGenerate func(ctx context.Context, analyses []analyzer.Result) error
    AfterGenerate  func(ctx context.Context, files []codegen.File) error
    BeforeWrite    func(ctx context.Context, files []codegen.File) error
    AfterWrite     func(ctx context.Context, summary Summary) error
}
```

2. Update `Pipeline`:
```go
type Pipeline struct {
    Env   Environment
    Hooks Hooks  // NEW
}
```

3. Call hooks at appropriate points in `Run()` method.

4. Support hook chaining:
```go
func (h *Hooks) Chain(other Hooks) Hooks
```

### Testing
- Test each hook is called
- Test error propagation from hooks
- Test hook chaining

### Files to Create
- `internal/pipeline/hooks.go`

### Files to Modify
- `internal/pipeline/pipeline.go` - integrate hooks

---

## Task 6: Structured Logging Interface

**Priority:** Low  
**Effort:** Low  
**Impact:** Low (nice to have)

### Goal
Abstract logging to allow custom loggers (not just slog).

### Implementation

1. Define interface:
```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    With(args ...any) Logger  // for contextual logging
}
```

2. Create slog adapter:
```go
type SlogAdapter struct {
    logger *slog.Logger
}

func (s *SlogAdapter) Debug(msg string, args ...any) {
    s.logger.Debug(msg, args...)
}
// ... etc
```

3. Update `Environment`:
```go
type Environment struct {
    FSResolver   func(string) (fileset.Resolver, error)
    Logger       Logger  // was *slog.Logger
    Writer       Writer
    SchemaParser schemaparser.Parser
    Generator    codegen.Generator
    Cache        cache.Cache
}
```

### Testing
- Test adapter passes through to slog
- Test With() creates contextual logger

### Files to Create
- `internal/logging/logger.go` - interface and adapter

### Files to Modify
- `internal/pipeline/pipeline.go` - use Logger interface
- All files that use `*slog.Logger` directly

---

## Implementation Order

1. **Task 1: Schema Parser Interface** - Foundation for multi-dialect
2. **Task 2: Memory Test Utilities** - Enables better testing
3. **Task 3: Generator Interface** - Completes interface-based design
4. **Task 4: Caching Layer** - Performance improvement
5. **Task 5: Pipeline Hooks** - Extensibility
6. **Task 6: Logging Interface** - Nice to have

Each task should be:
- Implemented in isolation
- Fully tested
- Committed separately
- Backward compatible

## Commit Message Template

```
feat(<package>): <brief description>

- <specific change 1>
- <specific change 2>
- <specific change 3>

Tests: <test coverage info>
Compatibility: <backward compatibility notes>
```
