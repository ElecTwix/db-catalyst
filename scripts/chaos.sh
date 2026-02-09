#!/bin/bash
# Chaos testing script for db-catalyst

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
TARGET=${1:-all}
CORPUS_SIZE=${2:-100}
SEED=${3:-42}

echo -e "${GREEN}Running chaos tests (target: $TARGET, corpus: $CORPUS_SIZE, seed: $SEED)...${NC}"

case $TARGET in
    tokenizer)
        echo -e "${BLUE}Testing tokenizer...${NC}"
        go test -v -run TestTokenizerChaos ./internal/testing/chaos/...
        ;;
    
    schema)
        echo -e "${BLUE}Testing schema parser...${NC}"
        go test -v -run TestSchemaParserChaos ./internal/testing/chaos/...
        ;;
    
    postgres)
        echo -e "${BLUE}Testing PostgreSQL parser...${NC}"
        go test -v -run TestPostgresParserChaos ./internal/testing/chaos/...
        ;;
    
    mysql)
        echo -e "${BLUE}Testing MySQL parser...${NC}"
        go test -v -run TestMySQLParserChaos ./internal/testing/chaos/...
        ;;
    
    query)
        echo -e "${BLUE}Testing query parser...${NC}"
        go test -v -run TestQueryParserChaos ./internal/testing/chaos/...
        ;;
    
    graphql)
        echo -e "${BLUE}Testing GraphQL parser...${NC}"
        go test -v -run TestGraphQLParserChaos ./internal/testing/chaos/...
        ;;
    
    all)
        echo -e "${YELLOW}Running all chaos tests...${NC}"
        go test -v ./internal/testing/chaos/...
        ;;
    
    benchmark)
        echo -e "${YELLOW}Running chaos benchmarks...${NC}"
        go test -bench=BenchmarkChaosCorruption -benchmem ./internal/testing/chaos/...
        ;;
    
    *)
        echo -e "${RED}Unknown target: $TARGET${NC}"
        echo "Usage: $0 [tokenizer|schema|postgres|mysql|query|graphql|all|benchmark] [corpus_size] [seed]"
        echo ""
        echo "Targets:"
        echo "  tokenizer  - Test tokenizer only"
        echo "  schema     - Test schema parser"
        echo "  postgres   - Test PostgreSQL parser"
        echo "  mysql      - Test MySQL parser"
        echo "  query      - Test query parser"
        echo "  graphql    - Test GraphQL parser"
        echo "  all        - Test all parsers (default)"
        echo "  benchmark  - Run chaos benchmarks"
        exit 1
        ;;
esac

echo -e "${GREEN}Chaos testing complete!${NC}"
