# Generic DDL Parser Architecture Design

**Version:** 1.0  
**Status:** Design Phase  
**Target:** v0.6.0  
**Author:** db-catalyst Team

## Executive Summary

This document proposes a **generic DDL parsing architecture** that enables db-catalyst to support any SQL dialect through a pluggable dialect system. Instead of maintaining separate parsers for each database (current approach), we will create a single dialect-agnostic parser core with database-specific adapters.

## Problem Statement

### Current Architecture (v0.5.0)
- **3 separate parsers**: SQLite, PostgreSQL, MySQL
- **Duplicated logic**: Each parses CREATE TABLE, constraints, relationships
- **Type mapping scattered**: Across engine packages
- **High maintenance**: Adding new database = ~2000 lines of new parser code
- **Inconsistency**: Same SQL features parsed differently per dialect

### Issues
1. Bug fixes must be applied to N parsers
2. New features (e.g., CTEs) implemented N times
3. Type system logic duplicated in type mappers AND parsers
4. Cannot easily add Oracle, SQL Server, CockroachDB, etc.

## Proposed Solution

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    SQL DDL Input                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Tokenizer (Shared)                             │
│  - Keywords, identifiers, literals, symbols                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Generic Parser Core                            │
│  - Parse: CREATE, ALTER, DROP                               │
│  - Parse: Table structure, column names                     │
│  - Delegate: Types, constraints → Dialect                   │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
         ▼             ▼             ▼
┌────────────┐ ┌────────────┐ ┌────────────┐
│  SQLite    │ │ PostgreSQL │ │   MySQL    │
│  Dialect   │ │  Dialect   │ │  Dialect   │
│            │ │            │ │            │
│ - Types    │ │ - Types    │ │ - Types    │
│ - Constraints│ - Constraints│ - Constraints│
│ - Options  │ │ - Options  │ │ - Options  │
└─────┬──────┘ └─────┬──────┘ └─────┬──────┘
      │              │              │
      └──────────────┼──────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Canonical AST                                  │
│  - Dialect-independent representation                       │
│  - Normalized types (INTEGER, TEXT, DECIMAL, etc.)          │
│  - Standardized constraints                                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Model Builder                                  │
│  - Convert AST → model.Catalog                              │
│  - Cross-reference validation                               │
│  - Relationship resolution                                  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Type Mapper                                    │
│  - Canonical Type → Go Type                                 │
│  - Database-specific optimizations                          │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Generated Code                                 │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Generic Parser Core

**Location:** `internal/schema/parser/core/`

**Responsibilities:**
- Parse SQL structure (CREATE, ALTER, DROP statements)
- Parse table/column names
- Parse basic constraints (NOT NULL, PRIMARY KEY, UNIQUE, FOREIGN KEY)
- Delegate dialect-specific parsing to dialect adapters
- Build generic AST

**Key Types:**

```go
// Parser is the dialect-agnostic parsing engine
type Parser struct {
    dialect Dialect
}

// Parse parses DDL and returns a generic AST
func (p *Parser) Parse(ctx context.Context, path string, content []byte) (*AST, error)

// AST is the generic abstract syntax tree
type AST struct {
    Statements []Statement
}

type Statement interface {
    statementNode()
    Pos() Position
}

type CreateTableStmt struct {
    Pos         Position
    Name        string
    IfNotExists bool
    Columns     []*ColumnDef
    Constraints []Constraint
    Options     []TableOption
}

type ColumnDef struct {
    Pos          Position
    Name         string
    Type         TypeInfo      // Delegated to dialect
    Constraints  []ColumnConstraint
    Default      *Value
    Comment      string
}
```

### 2. Dialect Interface

**Location:** `internal/schema/parser/dialect/`

**Interface Definition:**

```go
// Dialect handles database-specific parsing rules
type Dialect interface {
    // Metadata
    Name() string
    Version() string
    
    // Type Parsing
    // Parse a type declaration and return generic TypeInfo
    ParseType(tokens []Token) (TypeInfo, error)
    
    // Parse type arguments like (255) in VARCHAR(255)
    ParseTypeArgs(tokens []Token) ([]string, error)
    
    // Column Constraints
    // Parse column-level constraints (CHECK, DEFAULT, etc.)
    ParseColumnConstraint(tokens []Token) (ColumnConstraint, error)
    
    // Table Constraints
    // Parse table-level constraints
    ParseTableConstraint(tokens []Token) (TableConstraint, error)
    
    // Table Options
    // Parse table options (ENGINE, CHARSET, etc.)
    ParseTableOption(tokens []Token) (TableOption, error)
    
    // Keywords
    // Check if word is reserved keyword
    IsReservedKeyword(word string) bool
    
    // Case sensitivity for identifiers
    AreIdentifiersCaseSensitive() bool
    
    // Quote character for identifiers
    IdentifierQuote() rune
}

// TypeInfo represents a parsed type
type TypeInfo struct {
    Name        string            // Original type name
    Args        []string          // Type arguments
    Attributes  []string          // UNSIGNED, ZEROFILL, etc.
    Nullable    *bool             // Explicit NULL/NOT NULL
    Default     *Value
    Collate     string
    CharacterSet string
}

// Constraint types
type ColumnConstraint interface {
    constraintNode()
}

type NotNullConstraint struct{}
type DefaultConstraint struct{ Value *Value }
type CheckConstraint struct{ Expr string }
type AutoIncrementConstraint struct{}
type CollateConstraint struct{ Collation string }
type CommentConstraint struct{ Text string }
```

### 3. Type System

**Location:** `internal/schema/parser/types/`

**Purpose:** Map dialect-specific types to canonical types

```go
// CanonicalTypeCategory represents dialect-independent type categories
type CanonicalTypeCategory int

const (
    CategoryUnknown CanonicalTypeCategory = iota
    
    // Integers
    CategoryTinyInteger
    CategorySmallInteger
    CategoryInteger
    CategoryMediumInteger
    CategoryBigInteger
    CategorySerial
    CategoryBigSerial
    
    // Decimals
    CategoryDecimal
    CategoryNumeric
    CategoryFloat
    CategoryDouble
    CategoryReal
    
    // Strings
    CategoryChar
    CategoryVarchar
    CategoryText
    CategoryTinyText
    CategoryMediumText
    CategoryLongText
    
    // Binary
    CategoryBinary
    CategoryVarbinary
    CategoryBlob
    CategoryTinyBlob
    CategoryMediumBlob
    CategoryLongBlob
    
    // Temporal
    CategoryDate
    CategoryTime
    CategoryDateTime
    CategoryTimestamp
    CategoryTimestampTZ
    CategoryYear
    CategoryInterval
    
    // Special
    CategoryBoolean
    CategoryUUID
    CategoryJSON
    CategoryJSONB
    CategoryXML
    CategoryEnum
    CategorySet
    CategoryArray
)

// CanonicalType is dialect-independent
type CanonicalType struct {
    Category    CanonicalTypeCategory
    Size        int           // VARCHAR(255) → Size: 255
    Precision   int           // DECIMAL(10,2) → Precision: 10
    Scale       int           // DECIMAL(10,2) → Scale: 2
    IsUnsigned  bool
    IsArray     bool
    ElementType *CanonicalType // For arrays
    EnumValues  []string       // For ENUM/SET
}

// TypeRegistry manages type mappings
type TypeRegistry struct {
    mappings map[string]DialectTypeInfo // key: "dialect:typename"
}

// RegisterType adds a type mapping
func (r *TypeRegistry) RegisterType(
    dialect string,
    typeNames []string,
    canonical CanonicalType,
)

// ResolveType converts dialect-specific type to canonical
func (r *TypeRegistry) ResolveType(
    dialect string,
    typeName string,
    args []string,
    attrs []string,
) (CanonicalType, error)

// Example registrations:
// SQLite:   "INTEGER" → CategoryBigInteger
// MySQL:    "INT" → CategoryInteger
//           "BIGINT UNSIGNED" → CategoryBigInteger + IsUnsigned
// Postgres: "SERIAL" → CategorySerial
//           "VARCHAR(255)" → CategoryVarchar + Size: 255
```

### 4. Dialect Implementations

Each dialect implements the `Dialect` interface.

#### SQLite Dialect
**Location:** `internal/schema/parser/dialect/sqlite.go`

```go
type SQLiteDialect struct{}

func (d *SQLiteDialect) Name() string { return "sqlite" }

func (d *SQLiteDialect) ParseType(tokens []Token) (TypeInfo, error) {
    // SQLite has affinity-based types
    // INTEGER, TEXT, BLOB, REAL, NUMERIC
}

func (d *SQLiteDialect) IsReservedKeyword(word string) bool {
    // SQLite keywords
}
```

#### PostgreSQL Dialect
**Location:** `internal/schema/parser/dialect/postgres.go`

```go
type PostgresDialect struct{}

func (d *PostgresDialect) ParseType(tokens []Token) (TypeInfo, error) {
    // Handle PostgreSQL-specific:
    // - SERIAL, BIGSERIAL
    // - VARCHAR, CHARACTER VARYING
    // - TIMESTAMPTZ, TIMESTAMP WITH TIME ZONE
    // - TEXT[], INTEGER[] (arrays)
    // - JSON, JSONB
    // - UUID
}
```

#### MySQL Dialect
**Location:** `internal/schema/parser/dialect/mysql.go`

```go
type MySQLDialect struct{}

func (d *MySQLDialect) ParseType(tokens []Token) (TypeInfo, error) {
    // Handle MySQL-specific:
    // - TINYINT, SMALLINT, MEDIUMINT, INT, BIGINT
    // - ENUM('a','b','c')
    // - SET('x','y','z')
    // - VARCHAR(255) with charset
    // - Attributes: UNSIGNED, ZEROFILL
}

func (d *MySQLDialect) ParseTableOption(tokens []Token) (TableOption, error) {
    // ENGINE=InnoDB, CHARSET=utf8mb4, etc.
}
```

### 5. Dialect Registry

**Location:** `internal/schema/parser/dialect/registry.go`

```go
var registry = map[string]DialectFactory{
    "sqlite":     func() Dialect { return &SQLiteDialect{} },
    "postgresql": func() Dialect { return &PostgresDialect{} },
    "postgres":   func() Dialect { return &PostgresDialect{} },
    "mysql":      func() Dialect { return &MySQLDialect{} },
}

func RegisterDialect(name string, factory DialectFactory) {
    registry[name] = factory
}

func GetDialect(name string) (Dialect, error) {
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown dialect: %s", name)
    }
    return factory(), nil
}

func SupportedDialects() []string {
    // Return list of registered dialects
}
```

## Implementation Phases

### Phase 1: Core Foundation (Week 1-2)
**Goal:** Create generic parser infrastructure

**Tasks:**
1. Create package structure
   - `internal/schema/parser/core/`
   - `internal/schema/parser/dialect/`
   - `internal/schema/parser/types/`

2. Define interfaces
   - `Dialect` interface
   - AST node types
   - Type system types

3. Implement core parser
   - Statement parsing (CREATE, ALTER, DROP)
   - Basic constraint parsing
   - Token delegation to dialects

4. Create SQLite dialect adapter
   - Move existing SQLite parser logic
   - Implement `Dialect` interface

5. Tests
   - Unit tests for core parser
   - Integration tests for SQLite

**Deliverable:** Working SQLite parser via new architecture

### Phase 2: Type System (Week 3)
**Goal:** Build canonical type system

**Tasks:**
1. Define `CanonicalTypeCategory` enum
2. Create `TypeRegistry`
3. Implement type resolution
4. Map all SQLite types → canonical
5. Map all PostgreSQL types → canonical
6. Map all MySQL types → canonical

**Deliverable:** Type registry with all three databases

### Phase 3: PostgreSQL Migration (Week 4)
**Goal:** Migrate PostgreSQL to new architecture

**Tasks:**
1. Implement `PostgresDialect`
2. Move PostgreSQL type parsing
3. Handle PostgreSQL-specific features:
   - Arrays
   - SERIAL/BIGSERIAL
   - JSON/JSONB
   - Domain types (future)
4. Update tests
5. Validate generated code quality

**Deliverable:** PostgreSQL working via generic parser

### Phase 4: MySQL Migration (Week 5)
**Goal:** Migrate MySQL to new architecture

**Tasks:**
1. Implement `MySQLDialect`
2. Move MySQL type parsing
3. Handle MySQL-specific features:
   - ENUM, SET
   - UNSIGNED, ZEROFILL
   - Table options (ENGINE, CHARSET)
   - Full-text indexes
4. Update tests
5. Fix example project

**Deliverable:** MySQL working via generic parser

### Phase 5: Polish & Cleanup (Week 6)
**Goal:** Remove old parsers, add new features

**Tasks:**
1. Deprecate old parser packages
2. Add error recovery improvements
3. Better diagnostic messages
4. Performance benchmarks
5. Documentation

**Deliverable:** Clean, documented, performant generic parser

### Phase 6: Future Dialects (Ongoing)
**Goal:** Add support for new databases

**Candidates:**
- **SQL Server**: ~2 days (T-SQL syntax)
- **Oracle**: ~3 days (PL/SQL, complex types)
- **CockroachDB**: ~1 day (PostgreSQL-compatible)
- **MariaDB**: ~1 day (MySQL-compatible)
- **Firebird**: ~2 days
- **ClickHouse**: ~2 days (analytical extensions)

Each new dialect = implement interface + register types

## Migration Strategy

### Backward Compatibility

```go
// New API (v0.6.0+)
import "github.com/electwix/db-catalyst/internal/schema/parser"

parser, err := parser.New("postgresql")
catalog, err := parser.Parse(ctx, path, content)

// Old API (v0.5.0) - Keep as wrapper
import schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"

parser, err := schemaparser.NewSchemaParser("postgresql") // Calls new API
```

### Deprecation Plan

1. **v0.6.0**: Introduce new API, mark old as deprecated
2. **v0.6.x**: Old API calls new implementation (wrapper)
3. **v0.7.0**: Remove old API

### Testing Strategy

1. **Golden file tests**: Ensure generated code unchanged
2. **Cross-validation**: Parse same schema with old & new, compare AST
3. **Fuzzing**: Random SQL generation, ensure no panics
4. **Integration**: Real-world schema files from users

## Benefits

### For Users
- **More databases**: Easy to add Oracle, SQL Server, etc.
- **Better errors**: Consistent diagnostic messages
- **Performance**: Single optimized parser core

### For Contributors
- **Lower barrier**: Add dialect = ~200 lines vs ~2000
- **Less maintenance**: Fix bug once, not N times
- **Clear architecture**: Well-defined interfaces

### For Project
- **Scalability**: Support 10+ databases easily
- **Quality**: Centralized validation logic
- **Features**: Add CTEs, window functions once

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Parser performance regression | Medium | Benchmark before/after, optimize hot paths |
| Breaking changes for users | Low | Keep backward compatibility wrappers |
| Complex dialect edge cases | Medium | Extensive test suite with real schemas |
| Time estimate too optimistic | Medium | Phase 1 MVP, iterate |
| Maintainer burnout | Low | Incremental migration, not big bang |

## Success Criteria

1. ✅ All existing tests pass without modification
2. ✅ Generated code identical (golden file tests)
3. ✅ Performance within 10% of old parsers
4. ✅ New dialect implementable in < 1 day
5. ✅ Zero lint issues
6. ✅ Documentation complete

## Appendix: Example Type Mappings

### Integer Types

| Canonical | SQLite | PostgreSQL | MySQL |
|-----------|--------|------------|-------|
| TinyInteger | INTEGER | SMALLINT | TINYINT |
| SmallInteger | INTEGER | SMALLINT | SMALLINT |
| Integer | INTEGER | INTEGER | INT |
| MediumInteger | INTEGER | INTEGER | MEDIUMINT |
| BigInteger | INTEGER | BIGINT | BIGINT |
| Serial | - | SERIAL | INT AUTO_INCREMENT |
| BigSerial | - | BIGSERIAL | BIGINT AUTO_INCREMENT |

### String Types

| Canonical | SQLite | PostgreSQL | MySQL |
|-----------|--------|------------|-------|
| Char(n) | TEXT | CHAR(n) | CHAR(n) |
| Varchar(n) | TEXT | VARCHAR(n) | VARCHAR(n) |
| Text | TEXT | TEXT | TEXT |
| LongText | TEXT | TEXT | LONGTEXT |

### Special Types

| Canonical | SQLite | PostgreSQL | MySQL |
|-----------|--------|------------|-------|
| Boolean | INTEGER | BOOLEAN | BOOLEAN/TINYINT |
| JSON | TEXT | JSON/JSONB | JSON |
| UUID | TEXT | UUID | CHAR(36) |
| Enum | TEXT | ENUM/Custom | ENUM |
| Array | - | ARRAY | - |
| DateTime | TEXT | TIMESTAMP | DATETIME |

## Conclusion

This architecture transforms db-catalyst from supporting 3 databases with 3 parsers to a generic system supporting N databases with 1 parser core + N dialect adapters. It reduces maintenance burden, enables rapid dialect addition, and provides a foundation for advanced features like CTEs and window functions.

**Estimated Effort:** 6 weeks (1 developer)  
**ROI:** High - Enables 10+ database support with minimal ongoing maintenance  
**Recommendation:** Proceed with Phase 1 after v0.5.0 release
