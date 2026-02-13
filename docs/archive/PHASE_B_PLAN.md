# Execution Plan: Code Quality (Phase B)

## Phase B: Code Quality (8-12 hours)

### B1: Refactor pipeline.Run() Function (4 hours)

**Problem:** Run() is 354 lines, handling multiple phases

**Solution:** Extract into smaller methods:

```go
func (p *Pipeline) Run(ctx context.Context, opts RunOptions) (Summary, error) {
    // Setup
    summary, addDiag, finalize := p.setupDiagnostics(opts)
    defer finalize()
    
    // Phase 1: Load config
    plan, err := p.loadConfig(ctx, opts, addDiag)
    if err != nil { return summary, err }
    
    // Phase 2: Parse schemas
    catalog, err := p.parseSchemas(ctx, plan, addDiag)
    if err != nil { return summary, err }
    
    // Phase 3: Analyze queries
    analyses, err := p.analyzeQueries(ctx, plan, catalog, addDiag)
    if err != nil { return summary, err }
    
    // Phase 4: Generate code
    files, err := p.generateCode(ctx, plan, catalog, analyses, addDiag)
    if err != nil { return summary, err }
    
    // Phase 5: Write files
    return p.writeFiles(ctx, opts, files, summary)
}
```

**Sub-agent 1:** Extract setup and config loading
**Sub-agent 2:** Extract schema parsing
**Sub-agent 3:** Extract query analysis
**Sub-agent 4:** Extract code generation and file writing

### B2: Refactor buildPreparedFile() (3 hours)

**Problem:** 305 lines of string building

**Solution:** Break into methods for each component:
- buildPreparedStruct()
- buildPrepareMethod()
- buildQueryMethods()
- etc.

**Sub-agent 5:** Refactor buildPreparedFile

### B3: Deduplicate Type Conversion (2 hours)

**Problem:** Two nearly identical SQLiteTypeToGo functions

**Solution:** Extract shared mapping, have one call the other

**Sub-agent 6:** Deduplicate type conversion

## Commit Strategy

1. After B1: `refactor(pipeline): extract phases from Run function`
2. After B2: `refactor(codegen): break buildPreparedFile into smaller methods`
3. After B3: `refactor(analyzer): deduplicate type conversion logic`

## Verification After Each Commit

- `task test` - All tests pass
- `task lint` - No lint issues
- `go build ./cmd/db-catalyst` - Builds successfully
