#!/bin/bash
# Benchmark comparison script

cd "$(dirname "$0")"

echo "=========================================="
echo "db-catalyst Benchmarks"
echo "=========================================="

# Save baseline if it doesn't exist
if [ ! -f benchmark.txt ]; then
    echo ""
    echo "No baseline found. Creating baseline..."
    go test ./... -bench=. -benchmem -benchtime=1s -count=1 2>&1 | tee benchmark.txt
else
    echo ""
    echo "Current benchmarks:"
    go test ./... -bench=. -benchmem -benchtime=500ms -count=3 | head -50

    echo ""
    echo "=========================================="
    echo "Benchmark Comparison"
    echo "=========================================="

    # Check if benchcmp is available
    if command -v benchcmp >/dev/null 2>&1; then
        echo ""
        benchcmp benchmark.txt <(go test ./... -bench=. -benchmem -benchtime=1s -count=1 2>&1)
    else
        echo ""
        echo "Install benchcmp for comparison: go install golang.org/x/tools/cmd/benchcmp@latest"
        echo ""
        echo "To save current as baseline: cp benchmark.txt benchmark.txt.bak && go test ./... -bench=. -benchmem -benchtime=1s -count=1 > benchmark.txt"
    fi
fi

echo ""
echo "To save as baseline: cp benchmark.txt benchmark.txt.bak && make benchmark-save"
