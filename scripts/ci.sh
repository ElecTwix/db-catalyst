#!/bin/bash
# Full CI check script

set -e

cd "$(dirname "$0")"

echo "=========================================="
echo "db-catalyst Full CI Check"
echo "=========================================="

echo ""
echo "[1/4] Formatting code..."
gofmt -l -w . 2>/dev/null || true
goimports -w -local github.com/electwix/db-catalyst . 2>/dev/null || true
echo "✓ Formatting complete"

echo ""
echo "[2/4] Running linter..."
golangci-lint run ./...
echo "✓ Linting complete"

echo ""
echo "[3/4] Running tests with race detector..."
go test ./... -race -count=1 -parallel=2
echo "✓ Tests with race complete"

echo ""
echo "[4/4] Running tests with coverage..."
go test ./... -coverprofile=coverage.out -count=1
go tool cover -func coverage.out | tail -5
echo "✓ Coverage complete"

echo ""
echo "=========================================="
echo "CI checks passed!"
echo "=========================================="
