# Engine Interface Design

## Overview

The Engine Interface abstracts database-specific logic to enable true multi-database support. This is a core architectural refactor for v0.4.0 that prepares the codebase for v0.5.0's multi-database extensibility goals.

## Current Problems

1. **Scattered dialect logic**: Type mapping is split across `internal/codegen/ast/types.go` with switch statements on `config.Database`
2. **Hardcoded SQLite**: Pipeline explicitly creates `schemaparser.NewSchemaParser("sqlite")`
3. **Mixed concerns**: SQL generation, type resolution, and schema parsing all have database-specific branches
4. **No clear extension point**: Adding a new database requires changes across multiple packages

## Design Goals

1. **Single Responsibility**: Each engine encapsulates all dialect-specific behavior
2. **Clean Separation**: Core logic is database-agnostic; engines provide dialect specifics
3. **Easy Extension**: New databases can be added by implementing the Engine interface
4. **Backward Compatible**: Existing functionality continues to work unchanged
5. **Testable**: Engines can be mocked for testing

## Core Interfaces

### Engine (Main Interface)

```go
// Engine encapsulates all database-specific behavior.
type Engine interface {
    // Identity
    Name() string                    // e.g., "sqlite", "postgresql"
    
    // Type System
    TypeMapper() TypeMapper          // SQL types → language types
    
    // Schema
    SchemaParser() schemaparser.SchemaParser  // DDL parser
    
    // SQL Generation (for output schemas)
    SQLGenerator() SQLGenerator      // Generate DDL in this dialect
    
    // Query
    QueryAnalyzer() QueryAnalyzer    // Dialect-specific query validation
    
    // Metadata
    DefaultDriver() string           // Default Go driver import
    SupportsFeature(Feature) bool    // Feature detection
}
```

### TypeMapper

```go
// TypeMapper handles SQL type to language type conversions.
type TypeMapper interface {
    // SQLToGo converts a SQL type to Go type info
    SQLToGo(sqlType string, nullable bool) TypeInfo
    
    // SQLToSemantic converts SQL type to semantic type category
    SQLToSemantic(sqlType string, nullable bool) types.SemanticType
    
    // GetRequiredImports returns imports needed for generated code
    GetRequiredImports() map[string]string
    
    // SupportsPointersForNull indicates if pointer nullables work
    SupportsPointersForNull() bool
}

type TypeInfo struct {
    GoType      string
    UsesSQLNull bool
    Import      string
    Package     string
    IsPointer   bool
}
```

### SQLGenerator

```go
// SQLGenerator generates DDL in a specific dialect.
type SQLGenerator interface {
    // GenerateTable creates CREATE TABLE statement
    GenerateTable(table *model.Table) string
    
    // GenerateIndex creates CREATE INDEX statement
    GenerateIndex(index *model.Index, tableName string) string
    
    // GenerateColumnDef creates column definition
    GenerateColumnDef(column *model.Column) string
    
    // Dialect returns the target dialect
    Dialect() string
}
```

### QueryAnalyzer

```go
// QueryAnalyzer provides dialect-specific query validation.
type QueryAnalyzer interface {
    // ValidateQuery checks if query is valid for this dialect
    ValidateQuery(query string) []Diagnostic
    
    // SuggestIndexes recommends indexes for a query
    SuggestIndexes(query string, catalog *model.Catalog) []IndexSuggestion
}
```

## Package Structure

```
internal/engine/
├── engine.go           # Core interfaces
├── registry.go         # Engine registry and factory
├── features.go         # Feature flags enum
├── sqlite/
│   ├── engine.go       # SQLite engine implementation
│   ├── types.go        # SQLite type mapper
│   ├── sqlgen.go       # SQLite SQL generator
│   └── parser.go       # SQLite schema parser wrapper
├── postgres/
│   ├── engine.go       # PostgreSQL engine implementation
│   ├── types.go        # PostgreSQL type mapper
│   ├── sqlgen.go       # PostgreSQL SQL generator
│   └── parser.go       # PostgreSQL schema parser (future)
└── mysql/
    ├── engine.go       # MySQL engine implementation
    ├── types.go        # MySQL type mapper
    └── sqlgen.go       # MySQL SQL generator
```

## Migration Strategy

### Phase 1: Interface Definition
1. Create `internal/engine` package with interfaces
2. Define feature flags
3. Create engine registry

### Phase 2: SQLite Extraction
1. Move SQLite-specific logic from `ast/types.go` to `engine/sqlite/types.go`
2. Move SQLite parser wrapper to `engine/sqlite/parser.go`
3. Move SQL generation from `codegen/sql/` to `engine/sqlite/sqlgen.go`
4. Create `engine/sqlite/engine.go` that implements Engine interface

### Phase 3: PostgreSQL Extraction
1. Move PostgreSQL type logic from `ast/types.go` to `engine/postgres/types.go`
2. Create `engine/postgres/engine.go`
3. Implement PostgreSQL SQL generator

### Phase 4: Integration
1. Update `Pipeline` to accept Engine injection
2. Update `TypeResolver` to delegate to Engine.TypeMapper()
3. Update config loading to select appropriate engine
4. Remove switch statements on `config.Database`

### Phase 5: Cleanup
1. Deprecate old type resolution methods
2. Update tests to use engine interfaces
3. Add engine-specific tests

## Usage Example

```go
// Creating an engine
engine, err := engine.New("postgresql")
if err != nil {
    return err
}

// Using type mapper
typeMapper := engine.TypeMapper()
typeInfo := typeMapper.SQLToGo("UUID", false)
// Returns: TypeInfo{GoType: "uuid.UUID", Import: "github.com/google/uuid", ...}

// Using schema parser
parser := engine.SchemaParser()
catalog, diags, err := parser.Parse(ctx, path, content)

// Pipeline integration
pipeline := &pipeline.Pipeline{
    Env: pipeline.Environment{
        Engine: engine,  // Injected instead of hardcoded
    },
}
```

## Benefits

1. **Clear Boundaries**: Database logic is isolated in engine packages
2. **Testability**: Can mock Engine for testing core logic
3. **Extensibility**: Adding Oracle or SQL Server means just implementing Engine
4. **Maintainability**: Changes to PostgreSQL don't risk SQLite
5. **Type Safety**: Compiler enforces engine implementations

## Compatibility

- All existing configs continue to work
- Default behavior unchanged (SQLite is default)
- No breaking changes to public APIs
- Internal refactoring only

## Open Questions

1. Should engines be stateful or stateless?
   - Decision: Stateless for simplicity, configurable via options

2. How to handle custom type mappings per engine?
   - Decision: Pass CustomTypeMapping to engine constructor

3. Should SQLGenerator support schema migration generation?
   - Decision: Out of scope for now, may add later

4. How to handle database-specific query features?
   - Decision: QueryAnalyzer interface for validation, Parser for parsing

## Success Criteria

- [ ] All tests pass without modification
- [ ] No `switch cfg.Database` statements in core logic
- [ ] New engine can be added by implementing 1 interface
- [ ] Pipeline works with any engine
- [ ] Performance unchanged (no overhead)
