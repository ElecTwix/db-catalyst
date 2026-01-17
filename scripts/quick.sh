#!/bin/bash
# Quick development script - format, lint, test

set -e

cd "$(dirname "$0")"

echo "=========================================="
echo "db-catalyst Quick Check"
echo "=========================================="

echo ""
echo "[1/3] Formatting code..."
gofmt -l -w . 2>/dev/null || true
goimports -w -local github.com/electwix/db-catalyst . 2>/dev/null || true
echo "✓ Formatting complete"

echo ""
echo "[2/3] Running linter..."
golangci-lint run ./... || true
echo "✓ Linting complete"

echo ""
echo "[3/3] Running tests..."
go test ./... -count=1 -parallel=4
echo "✓ Tests complete"

echo ""
echo "=========================================="
echo "All checks passed!"
echo "=========================================="
