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

### Phase 3: Type Inference Improvements (MEDIUM PRIORITY) ✅ DONE
- [x] Improve CTE type inference for literal columns
- [x] Better window function metadata handling

### Phase 4: Testing & Documentation (MEDIUM PRIORITY)
- [x] Add tests for new commands
- [x] Add tests for FTS5/triggers
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
- [x] :execrows command support (commit 9bca9da)
- [x] :execlastid command support (commit 9bca9da)
- [x] FTS5 virtual table support (commit 9bca9da)
- [x] TRIGGER support with warning (commit 9bca9da)
- [x] CTE type inference improvements (commit 89f2d1f)
- [x] Window function metadata handling

### In Progress
- None

### Completed All High/Medium Priority Tasks ✅

### Blocked
- None

## Verified Working

### New Commands
```go
// :execlastid - returns last insert ID
func (q *Queries) InsertItem(ctx context.Context, name string) (int64, error) {
    res, err := q.db.ExecContext(ctx, queryInsertItem, name)
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

// :execrows - returns rows affected
func (q *Queries) DeleteAllItems(ctx context.Context) (int64, error) {
    res, err := q.db.ExecContext(ctx, queryDeleteAllItems)
    if err != nil {
        return 0, err
    }
    return res.RowsAffected()
}
```

### Schema Parser
```sql
-- FTS5 virtual tables - parsed with warning
CREATE VIRTUAL TABLE posts_fts USING fts5(title, content);

-- Triggers - parsed with warning, body skipped
CREATE TRIGGER posts_ai AFTER INSERT ON posts BEGIN
    INSERT INTO posts_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;
```

## Notes
- All changes must pass existing tests
- Follow existing code patterns
- Update AGENTS.md if new test commands needed
