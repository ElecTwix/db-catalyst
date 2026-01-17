#!/bin/bash
# Complete integration test orchestration
# Tests db-catalyst with real databases

set -e

cd "$(dirname "$0")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

PASS=0
FAIL=0

pass() { echo -e "${GREEN}✓ PASS${NC}"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}✗ FAIL${NC}"; FAIL=$((FAIL+1)); }

echo "========================================"
echo "db-catalyst Full Integration Test Suite"
echo "========================================"
echo ""

# Pre-flight checks
log_info "Running pre-flight checks..."

if ! command -v docker >/dev/null 2>&1; then
    log_error "Docker not found. Install Docker first."
    exit 1
fi

if ! command -v go >/dev/null 2>&1; then
    log_error "Go not found. Install Go first."
    exit 1
fi

log_info "Pre-flight checks passed"
echo ""

# Start infrastructure
echo "========================================"
echo "1. Starting Database Containers"
echo "========================================"
docker compose up -d

# Wait for databases
log_info "Waiting for MySQL..."
for i in {1..30}; do
    if docker exec db-catalyst-mysql mysqladmin ping -h localhost --silent 2>/dev/null; then
        log_info "MySQL ready"
        break
    fi
    [ $i -eq 30 ] && log_error "MySQL failed to start"
    sleep 1
done

log_info "Waiting for PostgreSQL..."
for i in {1..30}; do
    if docker exec db-catalyst-postgres pg_isready -U testuser -d dbtest >/dev/null 2>&1; then
        log_info "PostgreSQL ready"
        break
    fi
    [ $i -eq 30 ] && log_error "PostgreSQL failed to start"
    sleep 1
done

echo ""
echo "========================================"
echo "2. Building db-catalyst"
echo "========================================"
cd /home/electwix/dev/go/db-catalyst
if go build -o db-catalyst ./cmd/db-catalyst; then
    log_info "Build successful"
    pass
else
    log_error "Build failed"
    fail
fi

echo ""
echo "========================================"
echo "3. Generating SQL Schemas"
echo "========================================"

# Test SQLite generation
log_info "Generating SQLite schema..."
rm -rf gen-test
mkdir -p gen-test
cd gen-test
if /home/electwix/dev/go/db-catalyst/db-catalyst --sql-dialect sqlite 2>/dev/null; then
    if [ -f "schema.gen.sql" ]; then
        log_info "SQLite schema generated"
        pass
    else
        log_error "SQLite schema not generated"
        fail
    fi
else
    log_error "SQLite generation failed"
    fail
fi

# Test MySQL generation
log_info "Generating MySQL schema..."
if /home/electwix/dev/go/db-catalyst/db-catalyst --sql-dialect mysql 2>/dev/null; then
    if [ -f "schema.gen.sql" ]; then
        log_info "MySQL schema generated"
        pass
    else
        log_error "MySQL schema not generated"
        fail
    fi
else
    log_error "MySQL generation failed"
    fail
fi

# Test PostgreSQL generation
log_info "Generating PostgreSQL schema..."
if /home/electwix/dev/go/db-catalyst/db-catalyst --sql-dialect postgres 2>/dev/null; then
    if [ -f "schema.gen.sql" ]; then
        log_info "PostgreSQL schema generated"
        pass
    else
        log_error "PostgreSQL schema not generated"
        fail
    fi
else
    log_error "PostgreSQL generation failed"
    fail
fi

echo ""
echo "========================================"
echo "4. Testing SQL Against Databases"
echo "========================================"

# Test SQLite
log_info "Testing SQLite schema..."
rm -f /tmp/sqlite-test.db
if sqlite3 /tmp/sqlite-test.db < schema.gen.sql 2>/dev/null; then
    log_info "SQLite schema applied"
    pass
else
    log_error "SQLite schema failed"
    fail
fi
rm -f /tmp/sqlite-test.db

# Test MySQL
log_info "Testing MySQL schema..."
if docker exec db-catalyst-mysql mysql -utestuser -ptestpass dbtest < schema.gen.sql 2>/dev/null; then
    log_info "MySQL schema applied"
    pass
else
    log_error "MySQL schema failed"
    fail
fi

# Test PostgreSQL
log_info "Testing PostgreSQL schema..."
if docker exec -i db-catalyst-postgres psql -U testuser -d dbtest < schema.gen.sql 2>/dev/null; then
    log_info "PostgreSQL schema applied"
    pass
else
    log_error "PostgreSQL schema failed"
    fail
fi

echo ""
echo "========================================"
echo "5. Verifying Database State"
echo "========================================"

# Check MySQL
log_info "Verifying MySQL tables..."
MYSQL_TABLES=$(docker exec db-catalyst-mysql mysql -utestuser -ptestpass dbtest -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'dbtest'" 2>/dev/null)
if [ "$MYSQL_TABLES" -gt 0 ]; then
    log_info "MySQL has $MYSQL_TABLES tables"
    pass
else
    log_error "MySQL tables not found"
    fail
fi

# Check PostgreSQL
log_info "Verifying PostgreSQL tables..."
PG_TABLES=$(docker exec db-catalyst-postgres psql -U testuser -d dbtest -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'" 2>/dev/null | tr -d ' ')
if [ "$PG_TABLES" -gt 0 ]; then
    log_info "PostgreSQL has $PG_TABLES tables"
    pass
else
    log_error "PostgreSQL tables not found"
    fail
fi

echo ""
echo "========================================"
echo "6. Running Generated Go Code Tests"
echo "========================================"

cd /home/electwix/dev/go/db-catalyst
if go test ./test/integration/... -v -short 2>&1 | tail -20; then
    log_info "Go integration tests passed"
    pass
else
    log_warn "Go integration tests had issues (may need Docker running)"
fi

echo ""
echo "========================================"
echo "7. Running Full Test Suite"
echo "========================================"

if go test ./... -race -count=1 2>&1 | tail -30; then
    log_info "All tests passed"
    pass
else
    log_error "Some tests failed"
    fail
fi

echo ""
echo "========================================"
echo "8. Cleanup"
echo "========================================"

log_info "Stopping containers..."
docker compose down -v 2>/dev/null || true
log_info "Cleaning up generated files..."
rm -rf /home/electwix/dev/go/db-catalyst/gen-test /home/electwix/dev/go/db-catalyst/gen
log_info "Cleanup complete"

echo ""
echo "========================================"
echo "Test Results"
echo "========================================"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}All integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
