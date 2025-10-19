# db-catalyst Enhancement Requirements

## Overview

This document outlines the work required to extend db-catalyst to support a complex SQLite schema and query set currently compatible with sqlc. The goal is to achieve ~90% compatibility without implementing a full macro system like sqlc.slice().

**Target Schema**: `schema.sql` (776 lines, 40+ tables)  
**Target Queries**: `query.sql` (4000+ lines, 200+ queries)  
**Current Compatibility**: ~70%  
**Target Compatibility**: ~90%  
**Estimated Effort**: 4-6 weeks

## Current Limitations

### 1. sqlc.slice() Macro System
**Problem**: Queries use `sqlc.slice('param_name')` for variable-length IN clauses
```sql
WHERE id IN (sqlc.slice('ids'));
```
**Current Behavior**: Parser treats this as literal text, not parameters
**Impact**: 11 queries affected

### 2. UPSERT Operations (ON CONFLICT)
**Problem**: SQLite UPSERT syntax not supported
```sql
INSERT INTO delta_urlenc_order(...) VALUES(...)
ON CONFLICT(example_id, ref_kind, ref_id) DO UPDATE SET rank=excluded.rank;
```
**Current Behavior**: Parser fails on ON CONFLICT clause
**Impact**: 13 queries affected

### 3. Partial Indexes
**Problem**: CREATE INDEX with WHERE clauses not parsed
```sql
CREATE INDEX delta_header_state_supp_idx ON delta_header_state (example_id) 
WHERE suppressed = TRUE;
```
**Current Behavior**: Parser generates warnings, ignores WHERE clause
**Impact**: 4 indexes affected (non-critical for code generation)

### 4. Complex DEFAULT Expressions
**Problem**: Function calls in DEFAULT values
```sql
updated BIGINT NOT NULL DEFAULT (unixepoch())
```
**Current Behavior**: May not parse correctly
**Impact**: Multiple columns affected

## Implementation Requirements

### Phase 1: Variadic IN Clause Support (Priority: HIGH)

#### 1.1 Query Parser Extensions
**File**: `internal/query/parser/parser.go`

**Task**: Extend `collectParams()` function to detect and handle IN clauses with multiple parameters

**Current Code Structure**:
```go
func collectParams(tokens []tokenizer.Token, blk block.Block) ([]Param, []Diagnostic) {
    // Existing logic for ? and :param
    // Need to add IN clause detection
}
```

**Required Changes**:
1. Detect pattern: `identifier IN (?, ?, ?, ...)`
2. Group consecutive positional parameters in IN clauses
3. Mark parameter as variadic with count
4. Update `Param` struct:

```go
type Param struct {
    Name       string
    Style      ParamStyle
    Order      int
    Line       int
    Column     int
    IsVariadic bool    // NEW: true for IN clause parameters
    VariadicCount int  // NEW: number of ? in IN clause
}
```

**Example Transformation**:
```sql
-- Input:
WHERE id IN (?, ?, ?, ?)

-- Output:
Param{Name: "ids", Style: ParamStylePositional, Order: 1, IsVariadic: true, VariadicCount: 4}
```

#### 1.2 Code Generation for Variadic Parameters
**File**: `internal/codegen/ast/builder.go`

**Task**: Generate Go methods with variadic parameters

**Required Changes**:
1. Detect variadic parameters in query analysis
2. Generate method signatures with `...Type` parameters
3. Handle parameter expansion in query execution

**Example Output**:
```go
// Instead of:
func (q *Queries) GetItemApisByIDs(ctx context.Context, arg1, arg2, arg3, arg4 []byte) error

// Generate:
func (q *Queries) GetItemApisByIDs(ctx context.Context, ids ...[]byte) error
```

#### 1.3 Query Execution Logic
**File**: Generated query files

**Task**: Expand variadic parameters into individual query parameters

**Implementation**:
```go
func (q *Queries) GetItemApisByIDs(ctx context.Context, ids ...[]byte) ([]GetItemApisByIDsRow, error) {
    // Expand ids slice into individual parameters
    args := make([]any, len(ids))
    for i, id := range ids {
        args[i] = id
    }
    
    rows, err := q.db.QueryContext(ctx, queryGetItemApisByIDs, args...)
    // ... rest of implementation
}
```

### Phase 2: UPSERT Support (Priority: HIGH)

#### 2.1 Query Parser Extensions
**File**: `internal/query/parser/parser.go`

**Task**: Add support for ON CONFLICT clauses

**Required Changes**:
1. Add new verb type:
```go
const (
    VerbUpsert Verb = iota + 5
)
```

2. Extend `determineVerb()` to detect UPSERT patterns:
```sql
INSERT INTO ... ON CONFLICT(...) DO UPDATE SET ...
```

3. Add UPSERT clause parsing:
```go
type UpsertClause struct {
    ConflictColumns []string
    UpdateColumns   []string
    UpdateExpressions []string
}

func parseUpsertClause(tokens []tokenizer.Token, start int) (UpsertClause, int, []Diagnostic) {
    // Parse: ON CONFLICT(col1, col2) DO UPDATE SET col1=excluded.col1, col2=excluded.col2
}
```

4. Update `Query` struct:
```go
type Query struct {
    Block       block.Block
    Verb        Verb
    Columns     []Column
    Params      []Param
    CTEs        []CTE
    Upsert      *UpsertClause  // NEW: UPSERT clause if present
    Diagnostics []Diagnostic
}
```

#### 2.2 Code Generation for UPSERT
**File**: `internal/codegen/ast/builder.go`

**Task**: Generate two-step UPSERT operations

**SQLite UPSERT Strategy**:
Since Go's database/sql doesn't directly support UPSERT, generate two operations:

```go
func (q *Queries) DeltaUrlencOrderUpsert(ctx context.Context, arg DeltaUrlencOrderUpsertParams) error {
    // Step 1: INSERT OR IGNORE
    _, err := q.db.ExecContext(ctx, `
        INSERT OR IGNORE INTO delta_urlenc_order(example_id, ref_kind, ref_id, rank, revision)
        VALUES (?, ?, ?, ?, ?)
    `, arg.ExampleID, arg.RefKind, arg.RefID, arg.Rank, arg.Revision)
    if err != nil {
        return err
    }
    
    // Step 2: UPDATE if insert failed (row existed)
    _, err = q.db.ExecContext(ctx, `
        UPDATE delta_urlenc_order 
        SET rank = ?, revision = ?
        WHERE example_id = ? AND ref_kind = ? AND ref_id = ?
    `, arg.Rank, arg.Revision, arg.ExampleID, arg.RefKind, arg.RefID)
    return err
}
```

#### 2.3 Parameter Struct Generation
**Task**: Generate parameter structs for UPSERT operations

**Example**:
```go
type DeltaUrlencOrderUpsertParams struct {
    ExampleID []byte
    RefKind   int64
    RefID     []byte
    Rank      string
    Revision  int64
}
```

### Phase 3: Schema Parser Enhancements (Priority: MEDIUM)

#### 3.1 Partial Index Support
**File**: `internal/schema/parser/parser.go`

**Task**: Parse WHERE clauses in CREATE INDEX statements

**Required Changes**:
1. Extend `parseCreateIndex()` to handle WHERE clauses
2. Update `Index` model:
```go
type Index struct {
    Name      string
    Unique    bool
    Columns   []string
    Predicate string  // NEW: WHERE clause predicate
    Span      tokenizer.Span
}
```

**Implementation**:
```go
func parseCreateIndex(createTok tokenizer.Token, unique bool) {
    // ... existing parsing ...
    
    if p.matchKeyword("WHERE") {
        whereStart := p.advance()
        predicateTokens := make([]tokenizer.Token, 0)
        for !p.isEOF() && !p.matchSymbol(";") {
            predicateTokens = append(predicateTokens, p.current())
            p.advance()
        }
        idx.Predicate = rebuildSQL(predicateTokens)
    }
    
    // ... rest of function ...
}
```

#### 3.2 Complex DEFAULT Values
**Task**: Better parsing of function calls and expressions in DEFAULT clauses

**Required Changes**:
1. Extend `parseDefaultValue()` to handle parentheses and function calls
2. Support expressions like `(unixepoch())`, `FALSE`, etc.

### Phase 4: Testing and Validation (Priority: MEDIUM)

#### 4.1 Test Coverage Requirements
**Files to Create/Update**:
- `internal/query/parser/parser_test.go` - Add tests for IN clauses and UPSERT
- `internal/schema/parser/parser_test.go` - Add tests for partial indexes
- `internal/codegen/generator_test.go` - Add end-to-end tests

**Test Cases Required**:
1. IN clause with 2, 5, 10 parameters
2. UPSERT with single and multiple conflict columns
3. Partial index parsing
4. Complex DEFAULT values
5. Integration tests with target schema

#### 4.2 Golden File Updates
**Files**: `internal/codegen/testdata/golden/*.golden`

**Task**: Update all golden files to reflect new code generation patterns

## Implementation Strategy

### Week 1-2: Variadic IN Clauses
1. Extend query parser for IN clause detection
2. Update Param struct and collection logic
3. Implement variadic code generation
4. Add basic tests

### Week 3-4: UPSERT Support
1. Add UPSERT verb and parsing
2. Implement two-step UPSERT generation
3. Add parameter struct generation
4. Add UPSERT tests

### Week 5-6: Schema Enhancements & Testing
1. Add partial index support
2. Improve DEFAULT value parsing
3. Comprehensive testing
4. Update documentation

## Success Criteria

### Functional Requirements
- [ ] All 11 sqlc.slice() queries generate working Go code
- [ ] All 13 UPSERT queries generate working Go code
- [ ] All 4 partial indexes parse without errors
- [ ] Complex DEFAULT values parse correctly
- [ ] Generated code compiles and runs correctly

### Quality Requirements
- [ ] 90%+ test coverage for new functionality
- [ ] All existing tests continue to pass
- [ ] Generated code follows Go best practices
- [ ] Performance impact is minimal (<10% slower)

### Compatibility Requirements
- [ ] Target schema (`schema.sql`) parses completely
- [ ] Target queries (`query.sql`) generate valid Go code
- [ ] Generated code works with both modernc.org/sqlite and mattn/go-sqlite3

## Technical Notes

### SQLite Specific Considerations
1. **UPSERT**: Use `INSERT OR IGNORE` + `UPDATE` pattern
2. **Boolean**: SQLite doesn't have native boolean, use INTEGER
3. **Parameter Expansion**: Go's `...` syntax works well with database/sql

### Performance Considerations
1. **IN Clause Expansion**: Keep reasonable limits (max 100 parameters)
2. **UPSERT Operations**: Two-step pattern is standard for SQLite
3. **Generated Code**: Avoid unnecessary allocations

### Error Handling
1. **Parse Errors**: Provide clear messages about unsupported features
2. **Runtime Errors**: Generated code should handle database errors properly
3. **Validation**: Warn about potential issues during generation

## Files to Modify

### Core Parser Files
- `internal/query/parser/parser.go` - Main query parsing logic
- `internal/schema/parser/parser.go` - Schema parsing logic
- `internal/schema/tokenizer/scanner.go` - May need token updates

### Code Generation Files
- `internal/codegen/ast/builder.go` - AST construction
- `internal/codegen/render/render.go` - Code rendering
- `internal/codegen/generator.go` - Main generation orchestration

### Model Files
- `internal/query/block/block.go` - Query block model
- `internal/schema/model/model.go` - Schema model
- `internal/query/analyzer/analyzer.go` - Query analysis

### Test Files
- `internal/query/parser/parser_test.go`
- `internal/schema/parser/parser_test.go`
- `internal/codegen/generator_test.go`
- `internal/codegen/testdata/golden/*.golden`

## Dependencies

### External Dependencies
- No new external dependencies required
- Use existing Go standard library and golang.org/x/tools/imports

### Internal Dependencies
- All changes build on existing architecture
- No breaking changes to existing APIs
- Backward compatibility maintained

## Risks and Mitigations

### Technical Risks
1. **Complex Query Parsing**: IN clauses can be nested and complex
   - Mitigation: Start with simple cases, expand gradually
2. **UPSERT Complexity**: Conflict resolution can be complex
   - Mitigation: Use simple two-step pattern initially
3. **Performance Impact**: Parameter expansion could be slow
   - Mitigation: Benchmark and optimize as needed

### Compatibility Risks
1. **Breaking Changes**: New features might break existing code
   - Mitigation: Maintain backward compatibility, add feature flags
2. **SQLite Version Differences**: Different SQLite versions have different features
   - Mitigation: Target widely supported SQLite features

### Schedule Risks
1. **Underestimation**: Complexity might be higher than expected
   - Mitigation: Start with MVP, iterate quickly
2. **Integration Issues**: New features might not integrate well
   - Mitigation: Continuous integration testing

## Conclusion

This enhancement will significantly expand db-catalyst's capabilities to handle complex, real-world SQLite schemas. The focus on practical compatibility over feature completeness ensures a manageable implementation timeline while delivering substantial value to users.

The modular approach allows for incremental delivery and testing, reducing risk while maintaining progress toward the 90% compatibility target.