#!/bin/bash
# Mutation testing script for db-catalyst

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Install go-mutesting if not present
if ! command -v go-mutesting &> /dev/null; then
    echo -e "${YELLOW}Installing go-mutesting...${NC}"
    go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
fi

# Parse arguments
MODE=${1:-quick}
VERBOSE=""
if [ "$2" = "--verbose" ] || [ "$2" = "-v" ]; then
    VERBOSE="--verbose"
fi

echo -e "${GREEN}Running mutation testing (mode: $MODE)...${NC}"

case $MODE in
    quick)
        # Run on most critical package only
        echo -e "${YELLOW}Testing tokenizer...${NC}"
        go-mutesting $VERBOSE ./internal/schema/tokenizer/...
        ;;
    
    schema)
        # Run on schema-related packages
        echo -e "${YELLOW}Testing schema packages...${NC}"
        go-mutesting $VERBOSE ./internal/schema/tokenizer/...
        go-mutesting $VERBOSE ./internal/schema/parser/...
        ;;
    
    query)
        # Run on query-related packages
        echo -e "${YELLOW}Testing query packages...${NC}"
        go-mutesting $VERBOSE ./internal/query/parser/...
        go-mutesting $VERBOSE ./internal/query/analyzer/...
        go-mutesting $VERBOSE ./internal/query/block/...
        ;;
    
    core)
        # Run on core internal packages
        echo -e "${YELLOW}Testing core packages...${NC}"
        go-mutesting $VERBOSE ./internal/schema/tokenizer/...
        go-mutesting $VERBOSE ./internal/schema/parser/...
        go-mutesting $VERBOSE ./internal/query/parser/...
        go-mutesting $VERBOSE ./internal/query/analyzer/...
        go-mutesting $VERBOSE ./internal/pipeline/...
        ;;
    
    all|full)
        # Run on all packages
        echo -e "${YELLOW}Testing all packages...${NC}"
        go-mutesting $VERBOSE ./internal/...
        ;;
    
    *)
        echo -e "${RED}Unknown mode: $MODE${NC}"
        echo "Usage: $0 [quick|schema|query|core|all] [--verbose]"
        echo ""
        echo "Modes:"
        echo "  quick   - Test tokenizer only (fastest)"
        echo "  schema  - Test schema parser packages"
        echo "  query   - Test query parser packages"
        echo "  core    - Test core packages (recommended)"
        echo "  all     - Test all packages (slowest)"
        exit 1
        ;;
esac

echo -e "${GREEN}Mutation testing complete!${NC}"
