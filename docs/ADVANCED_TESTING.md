# Advanced Testing Implementation Summary

This document summarizes the advanced testing capabilities added to db-catalyst.

## 1. Enhanced Fuzz Testing

### New Fuzz Tests Created:

#### `/internal/parser/fuzz_test.go`
- `FuzzParser` - Tests high-level parser with diverse SQL inputs
- `FuzzParserWithDebug` - Tests parser in debug/non-debug modes
- `FuzzParserWithMaxErrors` - Tests parser with varying error limits

#### `/internal/schema/parser/postgres/fuzz_test.go`
- `FuzzParser` - Tests PostgreSQL DDL parsing
- `FuzzParserWithComplexTypes` - Tests arrays, ranges, geometric types
- `FuzzParserWithConstraints` - Tests various constraint combinations

#### `/internal/schema/parser/mysql/fuzz_test.go`
- `FuzzParser` - Tests MySQL DDL parsing
- `FuzzParserWithStorageEngines` - Tests ENGINE= options
- `FuzzParserWithColumnAttributes` - Tests column modifiers

#### `/internal/parser/languages/graphql/fuzz_test.go`
- `FuzzParser` - Tests GraphQL schema parsing
- `FuzzParserValidate` - Tests GraphQL validation

#### `/internal/schema/tokenizer/fuzz_extended_test.go`
- `FuzzScanExtended` - Comprehensive tokenizer testing
- `FuzzScanWithOptions` - Tests with different options

### Running Fuzz Tests:

```bash
# Quick fuzz (30 seconds each)
task fuzz-all

# Specific parsers
task fuzz-tokenizer
task fuzz-schema
task fuzz-query
task fuzz-postgres
task fuzz-mysql
task fuzz-graphql

# Extended fuzz (60 seconds each)
task fuzz-extended

# Manual execution
go test -fuzz=FuzzScan -fuzztime=30s ./internal/schema/tokenizer/
go test -fuzz=^FuzzParser$ -fuzztime=30s ./internal/schema/parser/postgres/
```

## 2. Chaos Testing

### Implementation: `/internal/testing/chaos/`

**Chaos Package (`chaos.go`):**
- `Corruptor` struct with configurable RNG seed
- 7 mutation types: ByteFlip, ByteDelete, ByteInsert, ByteReplace, Utf8Corrupt, Truncation, BitInversion
- Methods: `Corrupt()`, `CorruptN()`, `GenerateCorpus()`
- Intentionally corrupts valid inputs to test parser robustness

**Chaos Tests (`chaos_test.go`):**
- `TestTokenizerChaos` - 500 corrupted inputs per valid input
- `TestSchemaParserChaos` - Tests schema parser resilience
- `TestPostgresParserChaos` - Tests PostgreSQL parser
- `TestMySQLParserChaos` - Tests MySQL parser
- `TestQueryParserChaos` - Tests query parser
- `TestGraphQLParserChaos` - Tests GraphQL parser
- `TestChaosWithSpecificCorruptions` - Tests each mutation type
- `BenchmarkChaosCorruption` - Benchmarks corruption operations

### Running Chaos Tests:

```bash
# All chaos tests
./scripts/chaos.sh all

# Specific targets
./scripts/chaos.sh tokenizer
./scripts/chaos.sh schema
./scripts/chaos.sh postgres
./scripts/chaos.sh mysql
./scripts/chaos.sh query
./scripts/chaos.sh graphql

# Using task
task chaos
task chaos-tokenizer
task chaos-schema
task chaos-query
task chaos-benchmark
```

## 3. Mutation Testing

### Configuration: `/internal/testing/mutation/`

**Documentation (`README.md`):**
- Installation instructions for go-mutesting
- Usage examples for all packages
- CI integration guide
- Target mutation scores: 70%+

**Script (`scripts/mutation.sh`):**
- Quick mode: tokenizer only (fast)
- Schema mode: schema packages
- Query mode: query packages
- Core mode: all critical packages
- All mode: everything (slow)
- Verbose output option

### Running Mutation Tests:

```bash
# Using script
./scripts/mutation.sh quick          # Fast - tokenizer only
./scripts/mutation.sh schema         # Schema parsers
./scripts/mutation.sh query          # Query parsers
./scripts/mutation.sh core           # Core packages (recommended)
./scripts/mutation.sh all            # Everything (slow)
./scripts/mutation.sh core --verbose # With details

# Using task
task mutation          # Quick mode
task mutation-schema   # Schema packages
task mutation-query    # Query packages
task mutation-core     # Core packages
task mutation-all      # All packages

# Manual
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
go-mutesting --verbose ./internal/schema/tokenizer/...
```

## 4. Taskfile Updates

Added tasks for:

```yaml
# Fuzz testing
fuzz-postgres      # PostgreSQL parser fuzz
task fuzz-mysql    # MySQL parser fuzz
task fuzz-graphql  # GraphQL parser fuzz
task fuzz-extended # Longer fuzz runs

# Chaos testing
task chaos              # All chaos tests
task chaos-tokenizer    # Tokenizer only
task chaos-schema       # Schema parser
task chaos-query        # Query parser
task chaos-benchmark    # Benchmarks

# Mutation testing
task mutation           # Quick mode
task mutation-schema    # Schema packages
task mutation-query     # Query packages
task mutation-core      # Core packages
task mutation-all       # All packages
```

## 5. Test Coverage

### What's Tested:

1. **Panic Prevention**: All parsers must handle corrupt/malformed input without panicking
2. **Unicode Handling**: UTF-8 edge cases, invalid sequences
3. **Boundary Cases**: Empty inputs, maximum sizes, truncation
4. **Syntax Variations**: Different SQL dialects, GraphQL schemas
5. **Mutation Resilience**: Small code changes caught by tests

### Corruption Types Applied:

- **Byte-level**: Flip, delete, insert, replace
- **Bit-level**: Inversion at bit positions
- **UTF-8**: Corrupt multi-byte sequences
- **Structure**: Truncation at arbitrary points
- **Intensity**: 1-5 corruptions per input

## 6. Usage Examples

### Run All Advanced Tests:

```bash
# Fuzz tests (5 minutes)
task fuzz-all

# Chaos tests (30 seconds)
task chaos

# Mutation testing (10-30 minutes)
task mutation-core
```

### CI Integration:

```yaml
# GitHub Actions example
- name: Chaos Tests
  run: task chaos

- name: Fuzz Tests (short)
  run: |
    go test -fuzz=FuzzScan -fuzztime=30s ./internal/schema/tokenizer/
    go test -fuzz=FuzzParse -fuzztime=30s ./internal/schema/parser/

- name: Mutation Score Check
  run: |
    go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
    SCORE=$(go-mutesting ./internal/schema/tokenizer/... 2>&1 | grep -oP 'Score: \K[0-9.]+')
    if (( $(echo "$SCORE < 0.70" | bc -l) )); then
      echo "Mutation score $SCORE below 70%"
      exit 1
    fi
```

## 7. Files Added/Modified

### New Files:
- `internal/parser/fuzz_test.go`
- `internal/schema/parser/postgres/fuzz_test.go`
- `internal/schema/parser/mysql/fuzz_test.go`
- `internal/parser/languages/graphql/fuzz_test.go`
- `internal/schema/tokenizer/fuzz_extended_test.go`
- `internal/testing/chaos/chaos.go`
- `internal/testing/chaos/chaos_test.go`
- `internal/testing/mutation/README.md`
- `scripts/chaos.sh`
- `scripts/mutation.sh`

### Modified Files:
- `Taskfile.yml` - Added 13 new tasks

## 8. Next Steps

1. **Run mutation testing** to identify test gaps
2. **Add more seed corpus** for fuzz tests based on real usage
3. **Integrate into CI** for nightly runs
4. **Monitor fuzz coverage** and add targeted tests
5. **Document findings** from chaos testing in bug reports

---

**Note**: These tests are designed to be safe and never panic. If a parser panics during these tests, it's a bug that should be fixed.
