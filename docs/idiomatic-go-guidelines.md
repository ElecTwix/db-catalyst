# Idiomatic Go Guidelines for db-catalyst

## Overview

This document outlines the idiomatic Go patterns and conventions to follow when contributing to db-catalyst. These guidelines ensure code is maintainable, readable, and follows Go community best practices.

## Table of Contents

1. [Generated Code](#generated-code)
2. [Interface Design](#interface-design)
3. [Error Handling](#error-handling)
4. [Context Propagation](#context-propagation)
5. [Options Pattern](#options-pattern)
6. [Resource Management](#resource-management)
7. [Naming Conventions](#naming-conventions)
8. [Testing Patterns](#testing-patterns)

---

## Generated Code

Generated code must look hand-written by an experienced Go engineer.

### 1.1 Naming

✅ **DO:**
```go
type User struct {
    ID       int64  `json:"id"`       // PascalCase for exported
    Email    string `json:"email"`    // Descriptive names
    Username string `json:"username"`
}

func (q *Queries) GetUserByID(ctx context.Context, userID int64) (User, error) {
    // Descriptive parameter names
}
```

❌ **DON'T:**
```go
type User struct {
    Field1 string `json:"field1"`    // Generic names
    Field2 string `json:"field2"`
}

func (q *Queries) GetUser(ctx context.Context, id int64) (User, error) {
    // Ambiguous parameter names
}
```

### 1.2 Signatures

✅ **DO:**
```go
// Context-first, error-last
func (q *Queries) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)

// Return descriptive types
func (q *Queries) ListUsers(ctx context.Context) ([]User, error)
```

❌ **DON'T:**
```go
// Missing context
func (q *Queries) CreateUser(req CreateUserRequest) (*User, error)

// Context in wrong position
func (q *Queries) CreateUser(req CreateUserRequest, ctx context.Context) (*User, error)

// Named results (unnecessary)
func (q *Queries) CreateUser(ctx context.Context, req CreateUserRequest) (user *User, err error) {
```

### 1.3 Exported Types

✅ **DO:**
```go
// Exported types are documented
// User represents a user account in the system.
type User struct {
    // ID is the unique identifier for the user.
    ID int64 `json:"id"`
}

// Interface with clear contract
// Querier provides methods for database queries.
type Querier interface {
    // CreateUser creates a new user account.
    CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
}
```

❌ **DON'T:**
```go
type User struct {
    ID   int64 `json:"id"`    // No documentation
    Name string `json:"name"`
}

type Querier interface {
    CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)  // No docs
}
```

---

## Interface Design

Keep interfaces small, focused, and composable.

### 2.1 Interface Composition

✅ **DO:**
```go
// Small, focused interfaces
type SchemaParser interface {
    ParseSchema(ctx context.Context, sql string) (*model.Catalog, error)
}

type Validator interface {
    Validate(input string) ([]Issue, error)
}

// Compose for complex needs
type Parser interface {
    SchemaParser
    Validator
}

func (p *Parser) ParseAndValidate(ctx context.Context, sql string) (*model.Catalog, error) {
    catalog, err := p.ParseSchema(ctx, sql)
    if err != nil {
        return nil, err
    }

    issues, err := p.Validate(sql)
    if err != nil {
        return nil, err
    }

    if len(issues) > 0 {
        // Handle issues...
    }

    return catalog, nil
}
```

❌ **DON'T:**
```go
// Monolithic interface
type Parser interface {
    ParseSchema(ctx context.Context, sql string) (*model.Catalog, error)
    ParseQuery(ctx context.Context, sql string) (*model.Query, error)
    ParseFile(ctx context.Context, path string) (*model.Catalog, error)
    ValidateSchema(ctx context.Context, sql string) error
    ValidateQuery(ctx context.Context, sql string) error
    Format(sql string) string
    GetMetadata() *Metadata
    // 20 more methods...
}
```

### 2.2 Accept Interfaces, Return Concrete Types

✅ **DO:**
```go
// Accept interfaces for flexibility
func ProcessCatalog(p SchemaParser, catalog *model.Catalog) error {
    // Work with any SchemaParser implementation
}

func GenerateCode(generator CodeGenerator, catalog *model.Catalog) ([]byte, error) {
    // Use any CodeGenerator implementation
}
```

❌ **DON'T:**
```go
// Accept concrete types
func ProcessCatalog(p *SQLiteParser, catalog *model.Catalog) error {
    // Only works with SQLiteParser
}

func GenerateCode(g *GoGenerator, catalog *model.Catalog) ([]byte, error) {
    // Only works with GoGenerator
}
```

---

## Error Handling

Use Go 1.20+ features for better error aggregation and wrapping.

### 3.1 Error Wrapping

✅ **DO:**
```go
// Use %w for wrapping
if err := p.parse(); err != nil {
    return nil, fmt.Errorf("parse schema: %w", err)
}

if err := p.validate(); err != nil {
    return nil, fmt.Errorf("validate schema: %w", err)
}
```

❌ **DON'T:**
```go
// Use %v (loses stack)
if err := p.parse(); err != nil {
    return nil, fmt.Errorf("parse schema: %v", err)
}

// Silent error wrapping
catalog, err := p.parse()
if err != nil {
    // Log but return catalog anyway
    log.Println(err)
}
return catalog, nil
```

### 3.2 Multiple Errors (Go 1.20+)

✅ **DO:**
```go
// Aggregate multiple errors
func (p *Parser) ValidateAndParse(ctx context.Context, sql string) (*model.Catalog, error) {
    var errs []error
    
    catalog, err := p.parseSchema(ctx, sql)
    if err != nil {
        errs = append(errs, fmt.Errorf("parse: %w", err))
    }
    
    issues, err := p.validate(sql)
    if err != nil {
        errs = append(errs, fmt.Errorf("validate: %w", err))
    }
    
    if len(errs) > 0 {
        return nil, errors.Join(errs, "validation failed")
    }
    
    if len(issues) > 0 {
        // Log issues but don't fail
        for _, issue := range issues {
            p.log.Warn(issue.Message)
        }
    }
    
    return catalog, nil
}
```

❌ **DON'T:**
```go
// Return first error only
func (p *Parser) ValidateAndParse(ctx context.Context, sql string) (*model.Catalog, error) {
    catalog, err := p.parseSchema(ctx, sql)
    if err != nil {
        return nil, err
    }
    
    // Validation errors lost!
    _, _ = p.validate(sql)
    
    return catalog, nil
}
```

### 3.3 Error Types

✅ **DO:**
```go
// Define error types with Is()/As()
var (
    ErrInvalidSchema = errors.New("invalid schema")
    ErrQueryNotFound = errors.New("query not found")
)

// Use errors.Is for checking
if errors.Is(err, ErrInvalidSchema) {
    // Handle invalid schema
}
```

❌ **DON'T:**
```go
// String matching
if err != nil && err.Error() == "invalid schema" {
    // Brittle
}

// Hard-coded error strings
if strings.Contains(err.Error(), "invalid") {
    // Even more brittle
}
```

---

## Context Propagation

Context must be the first parameter and propagated through all layers.

### 4.1 Context as First Parameter

✅ **DO:**
```go
func (p *Parser) ParseSchema(ctx context.Context, sql string) (*model.Catalog, error) {
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    
    result, err := p.parseImpl(ctx, sql)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

func (p *Parser) parseImpl(ctx context.Context, sql string) (*model.Catalog, error) {
    // Always check context at start of long operations
    if err := ctx.Err(); err != nil {
        return nil, err
    }
    
    // Do work...
    return catalog, nil
}
```

❌ **DON'T:**
```go
// Ignoring context
func (p *Parser) ParseSchema(sql string) (*model.Catalog, error) {
    // Can't cancel long parse
}

// Creating new context
func (p *Parser) ParseSchema(parentCtx context.Context, sql string) (*model.Catalog, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    // Ignores parent's deadline!
}
```

### 4.2 Context Values (Sparingly)

✅ **DO:**
```go
// Use context values sparingly, only for request-scoped data
type contextKey string

const (
    requestIDKey contextKey = "request-id"
    tracingKey   contextKey = "tracing"
)

func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
    id, _ := ctx.Value(requestIDKey).(string)
    return id
}
```

❌ **DON'T:**
```go
// Using context for global state
const globalConfigKey contextKey = "config"

func WithConfig(ctx context.Context, cfg *Config) context.Context {
    // Config should be dependency, not context value
}
```

---

## Options Pattern

Use functional options for configuration to keep APIs clean and composable.

### 5.1 Options Pattern

✅ **DO:**
```go
// Define option type
type ParserOption func(*Parser)

// Implement options
func WithDialect(dialect string) ParserOption {
    return func(p *Parser) {
        p.dialect = dialect
    }
}

func WithDebug(enabled bool) ParserOption {
    return func(p *Parser) {
        p.debug = enabled
    }
}

func WithLogger(logger *slog.Logger) ParserOption {
    return func(p *Parser) {
        p.logger = logger
    }
}

// Constructor with options
func NewParser(opts ...ParserOption) *Parser {
    p := &Parser{
        dialect: "sqlite",  // Sensible default
        debug: false,
    }
    
    for _, opt := range opts {
        opt(p)
    }
    
    return p
}

// Usage
parser := NewParser(
    WithDialect("sqlite"),
    WithDebug(true),
    WithLogger(logger),
)
```

❌ **DON'T:**
```go
// Too many parameters
func NewParser(dialect string, debug bool, logger *slog.Logger, cache bool, timeout time.Duration, workers int) *Parser {
    // What order was this?
}

// Inconsistent option naming
type ParserConfig struct {
    Dialect       string
    DebugMode     bool  // Inconsistent
    Log           *slog.Logger
    EnableCache   bool
}
```

---

## Resource Management

Always clean up resources using `defer`. Use helper functions for test fixtures.

### 6.1 Deferred Cleanup

✅ **DO:**
```go
func ProcessFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("open %s: %w", path, err)
    }
    defer file.Close()  // Always closed
    
    // Process file...
    return nil
}

func WithTempDir(t *testing.T, fn func(string)) {
    t.Helper()
    dir, err := os.MkdirTemp("", "db-catalyst-*")
    if err != nil {
        t.Fatal(err)
    }
    
    defer os.RemoveAll(dir)  // Always cleaned up
    
    fn(dir)
}
```

❌ **DON'T:**
```go
// Manual cleanup, error-prone
func ProcessFiles(files []string) error {
    var fileHandles []*os.File
    
    for _, f := range files {
        file, _ := os.Open(f)
        fileHandles = append(fileHandles, file)
    }
    
    // What if error before close?
    // Cleanup is deferred but in wrong order
    defer func() {
        for _, fh := range fileHandles {
            fh.Close()
        }
    }()
    
    return nil
}

// Forgetting cleanup
func WithTempFile(content string) (string, error) {
    file, _ := os.CreateTemp("", "test-*")
    file.Write([]byte(content))
    file.Close()
    return file.Name(), nil  // File not cleaned up!
}
```

---

## Naming Conventions

Follow Go naming conventions consistently.

### 7.1 Package Names

✅ **DO:**
```
config/          // ✅ lowercase, single word
pipeline/        // ✅ lowercase, single word
codegen/          // ✅ lowercase, single word
```

❌ **DON'T:**
```
dbconfig/        // ❌ redundant "db"
ConfigPackage/    // ❌ mixed case
config/          // ❌ same as import
```

### 7.2 Exported Names

✅ **DO:**
```go
// PascalCase for exported
type Parser struct {}
type Catalog struct {}
func NewParser() *Parser {}

// All caps for constants
const DefaultTimeout = 30 * time.Second
const MaxRetries = 3

// Acronyms: only first letter
type HTTPClient struct {}  // ✅
type JSONDecoder struct {}  // ✅
```

❌ **DON'T:**
```go
// camelCase for exported
type parser struct {}  // ❌
type catalog struct {}  // ❌
func newParser() *Parser {}  // ❌

// All caps for variables
const defaultTimeout = 30 * time.Second  // ❌
const maxRetries = 3  // ❌

// Acronyms: all caps
type HTTP_CLIENT struct {}  // ❌
type JSON_DECODER struct {}  // ❌
```

### 7.3 Unexported Names

✅ **DO:**
```go
// camelCase for unexported
type Parser struct {
    dialect string    // ✅
    logger *slog.Logger  // ✅
    config *Config     // ✅
}

func (p *Parser) parseImpl() {  // ✅
func (p *Parser) validate() {  // ✅
}
```

❌ **DON'T:**
```go
// Snake_case for unexported
type Parser struct {
    dialect string    // ❌
    logger *slog.Logger  // ❌
}

// Underscore prefix
func _parseImpl() {  // ❌
func _validate() {  // ❌
```

---

## Testing Patterns

Use table-driven tests with proper setup and teardown.

### 8.1 Table-Driven Tests

✅ **DO:**
```go
func TestParser_ParseSchema(t *testing.T) {
    tests := []struct {
        name    string
        sql     string
        want    *model.Catalog
        wantErr bool
    }{
        {
            name: "simple table",
            sql:  "CREATE TABLE users (id INTEGER, name TEXT)",
            want: expectedCatalog,
            wantErr: false,
        },
        {
            name: "invalid syntax",
            sql:  "INVALID SQL",
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewParser()
            got, err := parser.ParseSchema(context.Background(), tt.sql)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseSchema() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr {
                if !cmp.Equal(got, tt.want) {
                    t.Errorf("ParseSchema() = %v, want %v", got, tt.want)
                }
            }
        })
    }
}
```

❌ **DON'T:**
```go
// Repetitive test setup
func TestParser_ParseSchema(t *testing.T) {
    parser := NewParser()
    
    // Test 1
    result1, err1 := parser.ParseSchema("CREATE TABLE users (id INTEGER)")
    if err1 != nil {
        t.Fatal(err1)
    }
    
    // Test 2
    parser2 := NewParser()
    result2, err2 := parser2.ParseSchema("CREATE TABLE posts (id INTEGER)")
    if err2 != nil {
        t.Fatal(err2)
    }
    // ...
}
```

### 8.2 Test Helpers

✅ **DO:**
```go
// Mark helpers with t.Helper()
func setupParser(t *testing.T) *Parser {
    t.Helper()  // Marks this as a helper
    return NewParser(WithDebug(false))
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
}

func assertEqual[T any](t *testing.T, got, want T) {
    t.Helper()
    if !cmp.Equal(got, want) {
        t.Errorf("not equal:\ngot:  %v\nwant: %v", got, want)
    }
}

// Fixture helpers
func WithTempFile(t *testing.T, content string) (string, func()) {
    t.Helper()
    f, err := os.CreateTemp("", "test-*.sql")
    if err != nil {
        t.Fatal(err)
    }
    
    if _, err := f.Write([]byte(content)); err != nil {
        t.Fatal(err)
    }
    
    f.Close()
    
    return f.Name(), func() {
        os.Remove(f.Name())
    }
}

// Usage
func TestProcessFile(t *testing.T) {
    tmpfile, cleanup := WithTempFile(t, "CREATE TABLE test (id INTEGER)")
    defer cleanup()
    
    result, err := ProcessFile(tmpfile)
    assertNoError(t, err)
    assertEqual(t, result.TableName, "test")
}
```

❌ **DON'T:**
```go
// Repetitive setup in each test
func TestParse(t *testing.T) {
    parser, _ := NewParser(WithDebug(false), WithLogger(nil))
    
    // Test 1
    result1, err1 := parser.Parse(sql1)
    if err1 != nil {
        t.Fatal(err1)
    }
    
    parser2, _ := NewParser(WithDebug(false), WithLogger(nil))
    result2, err2 := parser2.Parse(sql2)
    // Repetitive setup...
}

// Cleanup in each test (easy to forget)
func TestWithCleanup(t *testing.T) {
    dir, _ := os.MkdirTemp("", "test-*")
    defer os.RemoveAll(dir)  // Easy to forget if return early
    
    // If early return, cleanup still runs
    if someCondition {
        t.SkipNow()
    }
}
```

### 8.3 Subtests

✅ **DO:**
```go
func TestParser(t *testing.T) {
    t.Run("ParseSchema", TestParser_ParseSchema)
    t.Run("Validate", TestParser_Validate)
    t.Run("TypeMapping", TestParser_TypeMapping)
}

func TestParser_ParseSchema(t *testing.T) {
    t.Run("SimpleTable", func(t *testing.T) {
        parser := setupParser(t)
        catalog, err := parser.ParseSchema(context.Background(), "...")
        assertNoError(t, err)
        assertEqual(t, catalog.Tables["users"].Name, "users")
    })
    
    t.Run("ComplexTable", func(t *testing.T) {
        // ...
    })
}
```

❌ **DON'T:**
```go
// Flat test names
func TestParser(t *testing.T) {
    t.Run("ParseSchema_SimpleTable", func(t *testing.T) { ... })
    t.Run("ParseSchema_ComplexTable", func(t *testing.T) { ... })
    t.Run("Validate_Simple", func(t *testing.T) { ... })
    
    // Harder to run specific tests
}
```

---

## Use Standard Library

Prefer standard library over custom implementations.

### 9.1 Standard Library vs Custom

✅ **DO:**
```go
// Use os.ReadFile
data, err := os.ReadFile(path)

// Use strings.Builder
var b strings.Builder
for _, s := range strings {
    b.WriteString(s)
}

// Use io/fs for file operations
func readAll(fs fs.FS, path string) ([]byte, error) {
    return fs.ReadFile(path)
}
```

❌ **DON'T:**
```go
// Reimplement stdlib
func readFile(path string) ([]byte, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var buf []byte
    for {
        b := make([]byte, 1024)
        n, err := file.Read(b)
        if err != nil {
            return nil, err
        }
        buf = append(buf, b[:n]...)
    }
    return buf, nil
}

// String concatenation in loops
var s string
for _, part := range parts {
    s += part  // Inefficient
}
```

---

## Generics for Type Safety

Use generics where they add type safety and clarity (Go 1.18+).

### 10.1 Generic Functions

✅ **DO:**
```go
// Generic parser interface
type SchemaParser[T any] interface {
    Parse(ctx context.Context, input string) (*T, error)
}

// Generic function
func ParseSchema[T SchemaParser[T]](ctx context.Context, parser T, input string) (*T, error) {
    return parser.Parse(ctx, input)
}

// Type-safe usage
type SQLiteSchemaParser struct{}
type GraphQLSchemaParser struct{}

func TestParsers(t *testing.T) {
    sqliteParser := &SQLiteSchemaParser{}
    graphqlParser := &GraphQLSchemaParser{}
    
    catalog1, err1 := ParseSchema(ctx, sqliteParser, sql)
    catalog2, err2 := ParseSchema(ctx, graphqlParser, schema)
}
```

❌ **DON'T:**
```go
// Type-unsafe functions
func ParseSchema(ctx context.Context, parser interface{}, input string) (*model.Catalog, error) {
    // What if parser returns something else?
    p := parser.(SchemaParser)
    return p.Parse(ctx, input)  // Panic if wrong type
}

// Unnecessary generics
func Parse[T any](val T) T {
    return val  // No type safety benefit
}
```

---

## Summary

| Area | Key Points |
|------|-----------|
| **Generated Code** | Descriptive names, context-first, documented, no magic |
| **Interface Design** | Small, focused, composable, accept interfaces |
| **Error Handling** | `%w` wrapping, `errors.Join`, define error types |
| **Context** | First parameter, propagate, don't create unless needed |
| **Options Pattern** | Functional options, sensible defaults, composable |
| **Resources** | Always `defer` cleanup, helper functions for tests |
| **Naming** | Package: lowercase single word, Exported: PascalCase, Acronyms: HTTPClient |
| **Testing** | Table-driven, `t.Helper()`, subtests, fixtures |
| **Stdlib** | Use over custom implementations |
| **Generics** | Where type-safety matters, not overuse |

---

## Code Review Checklist

- [ ] Context is first parameter
- [ ] Error is last return value
- [ ] Errors use `%w` for wrapping
- [ ] Multiple errors use `errors.Join`
- [ ] Exported types are documented
- [ ] Package name is lowercase, single word
- [ ] No exported identifiers start with underscore
- [ ] Test helpers marked with `t.Helper()`
- [ ] Resources cleaned with `defer`
- [ ] Stdlib used instead of custom implementations
- [ ] Options pattern for configuration
- [ ] Interfaces are small and focused

---

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Go 1.20 Release Notes](https://go.dev/doc/go1.20)
