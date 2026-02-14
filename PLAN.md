# db-catalyst Improvement Plan

## Goal
Improve db-catalyst to match SQLC's feature set for complex SQL projects.

## Status
- **Started**: 2024-02-14
- **Current Phase**: Implementation

## Tasks

### Phase 1: Query Commands (HIGH PRIORITY) ✅ DONE
- [x] Add `:execrows` command support - returns number of affected rows
- [x] Add `:execlastid` command support - returns last insert ID

### Phase 2: Schema Parser Enhancements (HIGH PRIORITY) ✅ DONE
- [x] Add FTS5 virtual table support
- [x] Add TRIGGER support (parse and ignore with warning)

### Phase 3: Type Inference Improvements (MEDIUM PRIORITY)
- [ ] Improve CTE type inference for literal columns
- [ ] Better window function metadata handling

### Phase 4: Testing & Documentation (MEDIUM PRIORITY)
- [ ] Add tests for new commands
- [ ] Add tests for FTS5/triggers
- [ ] Update feature comparison docs

## Files to Modify

### Query Commands
- `internal/query/block/block.go` - Add CommandExecRows, CommandExecLastID
- `internal/codegen/ast/builder.go` - Generate code for new commands
- `internal/query/block/block_test.go` - Add tests

### Schema Parser
- `internal/schema/parser/parser.go` - Add FTS5 and TRIGGER handling
- `internal/schema/parser/parser_test.go` - Add tests

### Type Inference
- `internal/query/analyzer/analyzer.go` - Improve CTE handling

## Progress

### Completed
- [x] Custom type import propagation (commit 471c463)
- [x] DBTX interface pointer types fix
- [x] Prepared query variable scoping fix
- [x] Column-level type overrides
- [x] ON CONFLICT excluded table support
- [x] Schema INSERT statements (warn instead of error)

### In Progress
- [ ] :execrows command support
- [ ] :execlastid command support

### Blocked
- None

## Notes
- All changes must pass existing tests
- Follow existing code patterns
- Update AGENTS.md if new test commands needed
