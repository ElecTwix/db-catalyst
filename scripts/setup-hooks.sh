#!/bin/bash
# Setup git pre-commit hooks locally

cd "$(dirname "$0")/.git/hooks"

echo "Setting up pre-commit hooks..."

# Create pre-commit hook
cat > pre-commit << 'HOOK'
#!/bin/bash
# pre-commit hook for db-catalyst

set -e

echo "[pre-commit] Running checks..."

# Check formatting
echo "  Checking format..."
if ! gofmt -l . | grep -q '\.go$'; then
    echo "  ✗ Code is not formatted. Run 'make fmt' or 'gofmt -w .'"
    exit 1
fi

# Run linter (quick mode)
echo "  Running linter..."
if ! golangci-lint run --timeout=2m ./... > /dev/null 2>&1; then
    echo "  ✗ Linting failed. Run 'make lint' for details"
    exit 1
fi

# Quick test
echo "  Running tests..."
if ! go test ./... -count=1 -short > /dev/null 2>&1; then
    echo "  ✗ Tests failed. Run 'make test' for details"
    exit 1
fi

echo "  ✓ All checks passed"
exit 0
HOOK

chmod +x pre-commit

echo ""
echo "Pre-commit hook installed!"
echo ""
echo "The hook will run before each commit:"
echo "  - Format check"
echo "  - Linting"
echo "  - Short tests"
echo ""
echo "To skip the hook, use: git commit --no-verify"
