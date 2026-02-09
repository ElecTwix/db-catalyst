# Mutation Testing

This directory contains configuration and scripts for mutation testing using [go-mutesting](https://github.com/zimmski/go-mutesting).

## What is Mutation Testing?

Mutation testing introduces small changes (mutations) to your code and checks if your tests catch them. If a test fails after a mutation, the mutation was "killed." If all tests pass, the mutation "survived," indicating a potential gap in test coverage.

## Installation

```bash
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
```

## Usage

### Run mutation testing on all packages:

```bash
# Install go-mutesting if not already installed
which go-mutesting || go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest

# Run mutation testing
go-mutesting --verbose ./internal/...
```

### Run on specific packages:

```bash
# Tokenizer
go-mutesting --verbose ./internal/schema/tokenizer/...

# Schema parser
go-mutesting --verbose ./internal/schema/parser/...

# Query parser
go-mutesting --verbose ./internal/query/parser/...

# Query analyzer
go-mutesting --verbose ./internal/query/analyzer/...
```

### With custom options:

```bash
# Exclude specific mutation types
go-mutesting --verbose --disable=branch ./internal/schema/parser/...

# Show only survived mutations
go-mutesting --verbose ./internal/schema/parser/... 2>&1 | grep -A 5 "SURVIVED"

# Generate detailed report
go-mutesting --verbose --output=mutation-report.txt ./internal/...
```

## Mutation Operators

go-mutesting applies various mutation operators:

- **Conditionals Boundary**: Change `>` to `>=`, `<` to `<=`, etc.
- **Conditionals Negation**: Flip `==` to `!=`, `&&` to `||`, etc.
- **Arithmetic**: Change `+` to `-`, `*` to `/`, etc.
- **Statement Removal**: Remove `return`, `break`, `continue` statements
- **Branch**: Invert `if` conditions

## Interpreting Results

```
MUTATOR  PACKAGE                         FUNCTION         LINE  STATUS
--------------------------------------------------------------------------------
branch   internal/schema/parser          parseCreateTable  170  KILLED
branch   internal/schema/parser          parseCreateTable  175  SURVIVED  ⚠️
```

- **KILLED**: Test detected the mutation - good!
- **SURVIVED**: Test didn't detect the mutation - needs more tests
- **NOT COVERED**: Code not covered by tests

## Target Mutation Score

Aim for a mutation score of **70%+** for core packages:

- `internal/schema/tokenizer`: 75%+
- `internal/schema/parser`: 70%+
- `internal/query/parser`: 70%+
- `internal/query/analyzer`: 65%+

## CI Integration

Add to your CI pipeline (GitHub Actions example):

```yaml
- name: Mutation Testing
  run: |
    go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
    go-mutesting --verbose ./internal/schema/tokenizer/... 2>&1 | tee mutation.log
    # Fail if mutation score is below 70%
    SCORE=$(grep -oP 'Score: \K[0-9.]+' mutation.log)
    if (( $(echo "$SCORE < 0.70" | bc -l) )); then
      echo "Mutation score $SCORE is below 70%"
      exit 1
    fi
```

## Exclusions

Some mutations may not be meaningful to test. Add exclusions to `.mutation-exclusions`:

```
# Exclude debug/logging code
internal/schema/parser/debug.go

# Exclude test helpers
internal/testing/*

# Exclude generated code
internal/codegen/*/*.gen.go
```

## Task Integration

Use `task` to run mutation testing:

```bash
task mutation           # Run on all packages
task mutation-quick     # Run on critical packages only
task mutation-report    # Generate HTML report
```

Add to Taskfile.yml:

```yaml
tasks:
  mutation:
    desc: Run mutation testing
    cmds:
      - which go-mutesting || go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
      - go-mutesting --verbose ./internal/schema/tokenizer/... ./internal/schema/parser/... ./internal/query/parser/...
  
  mutation-quick:
    desc: Run mutation testing on critical packages only
    cmds:
      - go-mutesting --verbose ./internal/schema/tokenizer/...
  
  mutation-report:
    desc: Generate mutation testing report
    cmds:
      - go-mutesting --verbose --output=mutation-report.txt ./internal/...
      - echo "Report saved to mutation-report.txt"
```
