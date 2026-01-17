#!/bin/bash
# Integration test script for db-catalyst
# Tests generated code against real databases

set -e

cd "$(dirname "$0")"

COLORS='\033[0m\033[1;32m\033[1;31m\033[1;33m\033[0m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Configuration
MYSQL_CONN="testuser:testpass@tcp(127.0.0.1:3306)/dbtest"
POSTGRES_CONN="postgres://testuser:testpass@127.0.0.1:5432/dbtest?sslmode=disable"
SQLITE_DB="/tmp/db-catalyst-test.db"

# Colors for output
pass() { echo -e "${GREEN}✓ PASS${NC}"; }
fail() { echo -e "${RED}✗ FAIL${NC}"; }

echo "========================================"
echo "db-catalyst Integration Tests"
echo "========================================"
echo ""

# Check if Docker is available
if ! command -v docker >/dev/null 2>&1; then
    log_error "Docker is not installed. Please install Docker first."
    exit 1
fi

# Start databases
log_info "Starting databases..."
docker compose up -d

# Wait for databases to be ready
log_info "Waiting for MySQL..."
for i in {1..30}; do
    if docker exec db-catalyst-mysql mysqladmin ping -h localhost --silent 2>/dev/null; then
        log_info "MySQL is ready"
        break
    fi
    sleep 1
done

log_info "Waiting for PostgreSQL..."
for i in {1..30}; do
    if docker exec db-catalyst-postgres pg_isready -U testuser -d dbtest >/dev/null 2>&1; then
        log_info "PostgreSQL is ready"
        break
    fi
    sleep 1
done

echo ""
echo "========================================"
echo "Testing SQLite Output"
echo "========================================"
log_info "Generating SQLite schema..."

# Generate SQLite schema
cd /home/electwix/dev/go/db-catalyst
go run ./cmd/db-catalyst --sql-dialect sqlite --dry-run > /dev/null 2>&1 || true

# Test SQLite
if [ -f "gen/schema.gen.sql" ]; then
    log_info "Running SQLite schema..."
    rm -f "$SQLITE_DB"
    sqlite3 "$SQLITE_DB" < gen/schema.gen.sql
    rm -f "$SQLITE_DB"
    pass
else
    log_warn "SQLite schema not generated"
fi

echo ""
echo "========================================"
echo "Testing MySQL Output"
echo "========================================"
log_info "Generating MySQL schema..."

# Generate MySQL schema
go run ./cmd/db-catalyst --sql-dialect mysql --dry-run > /dev/null 2>&1 || true

# Test MySQL
if [ -f "gen/schema.gen.sql" ]; then
    log_info "Running MySQL schema..."
    if docker exec db-catalyst-mysql mysql -utestuser -ptestpass dbtest < gen/schema.gen.sql 2>/dev/null; then
        pass
    else
        fail
        log_error "MySQL schema failed to apply"
    fi
else
    log_warn "MySQL schema not generated"
fi

echo ""
echo "========================================"
echo "Testing PostgreSQL Output"
echo "========================================"
log_info "Generating PostgreSQL schema..."

# Generate PostgreSQL schema
go run ./cmd/db-catalyst --sql-dialect postgres --dry-run > /dev/null 2>&1 || true

# Test PostgreSQL
if [ -f "gen/schema.gen.sql" ]; then
    log_info "Running PostgreSQL schema..."
    if docker exec -i db-catalyst-postgres psql -U testuser -d dbtest < gen/schema.gen.sql 2>/dev/null; then
        pass
    else
        fail
        log_error "PostgreSQL schema failed to apply"
    fi
else
    log_warn "PostgreSQL schema not generated"
fi

echo ""
echo "========================================"
echo "Testing Generated Go Code"
echo "========================================"
log_info "Building generated Go code..."

# Build generated code
cd gen
if [ -f "models.gen.go" ]; then
    if go build . 2>/dev/null; then
        pass
    else
        fail
        log_warn "Generated Go code has compilation errors"
    fi
else
    log_warn "No Go models generated"
fi
cd ..

echo ""
echo "========================================"
echo "Cleanup"
echo "========================================"
log_info "Stopping databases..."
docker compose down

log_info "Integration tests complete!"
echo ""
echo "To run again: cd test/integration && ./run-tests.sh"
