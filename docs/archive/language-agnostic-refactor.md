# Language-Agnostic Type System Refactor Design

## Overview

This document describes the refactor needed to make db-catalyst's type system language-agnostic, enabling code generation for multiple languages (Go, Rust, TypeScript, Python, etc.).

## Current Architecture (Go-Specific)

```
SQL Type → Go Type → Generated Code
INTEGER   → int64   → struct field
TEXT      → string  → struct field
```

**Problems:**
1. Hardcoded Go types throughout analyzer and codegen
2. Custom types map directly to Go import paths
3. No separation between semantic meaning and language representation
4. Adding new languages requires rewriting large portions

## Proposed Architecture (Language-Agnostic)

```
SQL Type → Semantic Type → Language Mapper → Language Type → Generated Code
INTEGER   → Integer       → Go Mapper     → int64        → Go struct
          →               → Rust Mapper   → i64          → Rust struct
          →               → TS Mapper     → number       → TS interface
```

## Phase 1: Create Semantic Type System

### 1.1 Semantic Type Categories

```go
// internal/types/semantic.go

package types

// SemanticTypeCategory represents the semantic meaning of a type,
// independent of any programming language.
type SemanticTypeCategory int

const (
    // Numeric types
    CategoryInteger SemanticTypeCategory = iota
    CategoryBigInteger  // For int64, BIGINT
    CategorySmallInteger // For int16, SMALLINT
    CategoryDecimal     // Exact decimal numbers
    CategoryFloat       // IEEE 754 floating point
    CategoryDouble      // Double precision float
    
    // String types
    CategoryText        // Variable-length text
    CategoryChar        // Fixed-length character
    CategoryVarchar     // Variable-length with limit
    CategoryBlob        // Binary data
    
    // Temporal types
    CategoryTimestamp   // Date and time
    CategoryDate        // Date only
    CategoryTime        // Time only
    CategoryInterval    // Time duration
    
    // Boolean
    CategoryBoolean
    
    // Special types
    CategoryUUID
    CategoryJSON
    CategoryEnum        // Enumeration
    CategoryArray       // Array of another type
    CategoryCustom      // User-defined custom type
)

type SemanticType struct {
    Category   SemanticTypeCategory
    Nullable   bool
    
    // Type-specific metadata
    Precision  int           // For decimals: total digits
    Scale      int           // For decimals: digits after decimal
    MaxLength  int           // For strings: max characters (-1 = unlimited)
    EnumValues []string      // For enums: allowed values
    ElementType *SemanticType // For arrays: element type
    
    // Custom type info
    CustomName string       // Name of custom type (if CategoryCustom)
    CustomPackage string    // Package/module for custom type
}
```

### 1.2 SQLite to Semantic Mapping

```go
// internal/types/sqlite_mapping.go

package types

// SQLiteTypeMapper converts SQLite types to semantic types
type SQLiteTypeMapper struct{}

func (m *SQLiteTypeMapper) Map(sqlType string, nullable bool) SemanticType {
    upper := strings.ToUpper(sqlType)
    
    switch upper {
    case "INTEGER":
        return SemanticType{Category: CategoryInteger, Nullable: nullable}
    case "REAL", "FLOAT", "DOUBLE":
        return SemanticType{Category: CategoryFloat, Nullable: nullable}
    case "TEXT", "CHAR", "VARCHAR":
        return SemanticType{Category: CategoryText, Nullable: nullable}
    case "BLOB":
        return SemanticType{Category: CategoryBlob, Nullable: nullable}
    case "NUMERIC":
        return SemanticType{Category: CategoryDecimal, Nullable: nullable}
    case "BOOLEAN":
        return SemanticType{Category: CategoryBoolean, Nullable: nullable}
    default:
        // Check for custom type mappings
        return SemanticType{Category: CategoryCustom, CustomName: sqlType, Nullable: nullable}
    }
}
```

### 1.3 Language-Specific Type Mappers

```go
// internal/types/language.go

package types

// LanguageTypeMapper converts semantic types to language-specific types
type LanguageTypeMapper interface {
    // Map converts a semantic type to a language-specific type representation
    Map(semantic SemanticType) LanguageType
    
    // Name returns the language name ("go", "rust", "typescript", etc.)
    Name() string
}

// LanguageType represents a type in a specific language
type LanguageType struct {
    // The type name as it appears in code (e.g., "int64", "String", "number")
    Name string
    
    // Import path if external package is needed (e.g., "github.com/google/uuid")
    Import string
    
    // Package/module name for the import (e.g., "uuid", "std::collections")
    Package string
    
    // Whether this type needs special null handling
    NeedsNullWrapper bool
    
    // The null wrapper type (e.g., "sql.NullString" for Go)
    NullWrapper string
}

// GoTypeMapper implementation
// internal/types/go_mapper.go

type GoTypeMapper struct {
    customTypes map[string]config.CustomTypeMapping
}

func (m *GoTypeMapper) Map(semantic SemanticType) LanguageType {
    // Check custom types first
    if semantic.Category == CategoryCustom {
        if custom, ok := m.customTypes[semantic.CustomName]; ok {
            return LanguageType{
                Name:    custom.GoType,
                Import:  custom.GoImport,
                Package: custom.GoPackage,
            }
        }
    }
    
    // Map semantic categories to Go types
    switch semantic.Category {
    case CategoryInteger:
        return LanguageType{Name: "int64"}
    case CategoryBigInteger:
        return LanguageType{Name: "int64"}
    case CategoryText:
        if semantic.Nullable {
            return LanguageType{
                Name:         "sql.NullString",
                Import:       "database/sql",
                Package:      "sql",
                NeedsNullWrapper: true,
            }
        }
        return LanguageType{Name: "string"}
    case CategoryFloat:
        return LanguageType{Name: "float64"}
    case CategoryBoolean:
        return LanguageType{Name: "bool"}
    case CategoryTimestamp:
        return LanguageType{
            Name:    "time.Time",
            Import:  "time",
            Package: "time",
        }
    case CategoryBlob:
        return LanguageType{Name: "[]byte"}
    default:
        return LanguageType{Name: "interface{}"}
    }
}

func (m *GoTypeMapper) Name() string { return "go" }

// RustTypeMapper implementation (for future)
// internal/types/rust_mapper.go

type RustTypeMapper struct {
    customTypes map[string]config.RustCustomType // hypothetical
}

func (m *RustTypeMapper) Map(semantic SemanticType) LanguageType {
    switch semantic.Category {
    case CategoryInteger:
        return LanguageType{Name: "i32"}
    case CategoryBigInteger:
        return LanguageType{Name: "i64"}
    case CategoryText:
        return LanguageType{Name: "String"}
    case CategoryFloat:
        return LanguageType{Name: "f64"}
    case CategoryBoolean:
        return LanguageType{Name: "bool"}
    case CategoryTimestamp:
        return LanguageType{
            Name:    "chrono::DateTime<chrono::Utc>",
            Import:  "chrono",
            Package: "chrono",
        }
    default:
        return LanguageType{Name: "serde_json::Value"}
    }
}

func (m *RustTypeMapper) Name() string { return "rust" }
```

## Phase 2: Refactor Analyzer

### 2.1 Update Result Types

```go
// internal/query/analyzer/result.go

package analyzer

// ResultColumn now uses semantic types
type ResultColumn struct {
    Name       string
    Table      string
    Type       types.SemanticType  // Changed from GoType string
    GoType     string              // Kept for backward compat (deprecated)
    Nullable   bool
}

// ResultParam similarly updated
type ResultParam struct {
    Name          string
    Style         parser.ParamStyle
    Type          types.SemanticType  // Changed from GoType string
    GoType        string              // Kept for backward compat
    Nullable      bool
    // ... other fields
}
```

### 2.2 Update Analyzer

```go
// internal/query/analyzer/analyzer.go

func (a *Analyzer) Analyze(q parser.Query) Result {
    // ... existing code ...
    
    // Instead of resolving to Go types immediately, resolve to semantic types
    for _, col := range q.Columns {
        if col.Expr == "*" || strings.HasSuffix(col.Expr, ".*") {
            expanded, diags := expandStar(col, workingScope, q.Block, hasCatalog)
            result.Columns = append(result.Columns, expanded...)
            continue
        }
        
        rc, diags := resolveResultColumn(col, workingScope, q.Block, hasCatalog)
        result.Columns = append(result.Columns, rc)
    }
    
    // ... rest of analysis ...
}

func resolveResultColumn(col parser.Column, scope *queryScope, block block.Block, hasCatalog bool) (ResultColumn, []Diagnostic) {
    // ... existing resolution logic ...
    
    // Instead of:
    // goType := SQLiteTypeToGo(sqliteType)
    
    // Use:
    semanticType := sqliteMapper.Map(sqliteType, nullable)
    
    return ResultColumn{
        Name:     col.Alias,
        Table:    table,
        Type:     semanticType,
        Nullable: nullable,
    }, diags
}
```

## Phase 3: Refactor Code Generator

### 3.1 Generator Options

```go
// internal/codegen/options.go

package codegen

// Options for code generation
type Options struct {
    Package             string
    Out                 string
    Language            string              // NEW: "go", "rust", "typescript"
    TypeMapper          types.LanguageTypeMapper  // NEW: language-specific mapper
    CustomTypes         []config.CustomTypeMapping
    // ... other options
}

// NewOptions creates options with appropriate type mapper
func NewOptions(lang string, customTypes []config.CustomTypeMapping) (*Options, error) {
    var mapper types.LanguageTypeMapper
    
    switch lang {
    case "go":
        mapper = types.NewGoTypeMapper(customTypes)
    case "rust":
        mapper = types.NewRustTypeMapper(customTypes)
    case "typescript":
        mapper = types.NewTypeScriptTypeMapper(customTypes)
    default:
        return nil, fmt.Errorf("unsupported language: %s", lang)
    }
    
    return &Options{
        Language:   lang,
        TypeMapper: mapper,
        // ...
    }, nil
}
```

### 3.2 Update Builder to Use Semantic Types

```go
// internal/codegen/ast/builder.go

func (b *Builder) BuildParams(params []Param) ([]ParamInfo, error) {
    result := make([]ParamInfo, len(params))
    
    for i, p := range params {
        // Get semantic type from parameter
        semanticType := p.SemanticType
        
        // Map to language-specific type
        langType := b.opts.TypeMapper.Map(semanticType)
        
        result[i] = ParamInfo{
            Name:   p.Name,
            GoType: langType.Name,  // This is language-specific now
            Import: langType.Import,
            // ...
        }
    }
    
    return result, nil
}
```

## Phase 4: Configuration Updates

### 4.1 Multi-Target Configuration

```toml
# db-catalyst.toml

# Global settings
schemas = ["schema/*.sql"]
queries = ["queries/*.sql"]

# Multiple generation targets
[[target]]
language = "go"
package = "db"
out = "gen/go"

[[target]]
language = "rust"
module = "db"
out = "gen/rust"

# Custom types with language-specific mappings
[[custom_types]]
custom_type = "user_id"
sqlite_type = "INTEGER"

[target.types.go]
go_type = "github.com/example/types.UserID"

[target.types.rust]
rust_type = "types::UserId"

[target.types.typescript]
ts_type = "string"
```

## Phase 5: Implementation Strategy

### Step 1: Create types package (2-3 days)
- Create `internal/types/semantic.go` with SemanticType
- Create `internal/types/sqlite_mapping.go`
- Create `internal/types/go_mapper.go`
- Write tests

### Step 2: Update analyzer (2-3 days)
- Add semantic type field to ResultColumn/ResultParam
- Update analysis logic to populate semantic types
- Keep GoType field for backward compatibility
- Ensure all tests pass

### Step 3: Refactor codegen (3-4 days)
- Update TypeResolver to use type mapper
- Update Builder to use semantic types
- Update Generator to accept language options
- Regenerate all examples, verify output unchanged

### Step 4: Configuration updates (1-2 days)
- Add language field to config
- Add target array support
- Update config validation

### Step 5: Validation (2-3 days)
- Run full test suite
- Regenerate all examples
- Compare output with original
- Fix any discrepancies

## Migration Path

### Backward Compatibility
- Keep GoType fields (deprecated but functional)
- Default language is "go" if not specified
- Existing configs work without changes
- Gradual migration path for users

### Breaking Changes (if any)
- None in Phase 1
- Optional: Eventually remove GoType string fields

## Benefits

1. **Language Agnostic**: Core analysis is database-focused, not language-focused
2. **Extensible**: Adding Rust/TypeScript/Python is just adding mappers
3. **Maintainable**: Each language in its own package
4. **Testable**: Semantic types are easy to test
5. **Consistent**: Same analysis results generate equivalent code in all languages

## Next Steps

1. Review this design
2. Create a feature branch
3. Implement Phase 1 (types package)
4. Open PR for review
5. Iterate based on feedback

Total estimated time: 10-15 days for complete refactor to support Go + one additional language.
