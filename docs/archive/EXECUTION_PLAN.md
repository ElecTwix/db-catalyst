# Execution Plan: Critical Fixes (Phase A)

## Phase A: Critical Fixes (4-6 hours)

### A1: Remove All Panics (2 hours)

**Sub-agent 1:** Fix parser panics
- File: `internal/parser/dialects/parsers.go`
- Lines: 92-105, 137-150, 182-195
- Change: `panic()` → return error

**Sub-agent 2:** Fix GraphQL parser panic
- File: `internal/parser/languages/graphql/parser.go`
- Lines: 46-55
- Change: `panic()` → return error

**Sub-agent 3:** Fix builder panics
- File: `internal/codegen/ast/builder.go`
- Lines: 1120-1130
- Change: `panic()` → return error

**Sub-agent 4:** Fix naming panic
- File: `internal/codegen/ast/naming.go`
- Lines: 157-161
- Change: `panic()` → return error

### A2: Fix Unchecked Tokenizer Error (1 hour)

**Sub-agent 5:** Fix analyzer error handling
- File: `internal/query/analyzer/analyzer.go`
- Lines: 183-186
- Add proper error handling with diagnostics

### A3: Add File Size Limits (1-2 hours)

**Sub-agent 6:** Add file size validation
- File: `internal/pipeline/pipeline.go`
- Add: `checkFileSize()` function
- Add: 100MB limit constant
- Add: size check before reading files

## Commit Strategy

1. After A1: `fix(security): remove all panics from codebase`
2. After A2: `fix(error): handle tokenizer errors properly`
3. After A3: `feat(security): add file size limits`

## Verification After Each Commit

- `task test` - All tests pass
- `task lint` - No lint issues
- `go build ./cmd/db-catalyst` - Builds successfully
