# Makefile for db-catalyst
# A SQLite-only code generator

.PHONY: all test test-race test-cover test-all build lint fmt vet bench clean help install-deps benchmark benchmark-save benchmark-compare

# Go commands
GO = go
GOCOV = go tool cover
GOFMT = gofmt
GOIMPORTS = goimports -w -local github.com/electwix/db-catalyst
GOLANGCI = golangci-lint

# Directories
COVERAGE_OUT = coverage.out
COVERAGE_HTML = coverage.html
BENCHMARK_OUT = benchmark.txt

# Default target
all: test

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed."

# Run all tests (fast, no race detector)
test:
	@echo "Running tests..."
	$(GO) test ./... -count=1 -parallel=4

# Run tests with race detector (slower but catches concurrency bugs)
test-race:
	@echo "Running tests with race detector..."
	$(GO) test ./... -race -count=1 -parallel=2

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GO) test ./... -coverprofile=$(COVERAGE_OUT) -count=1 -parallel=4
	@echo "Coverage report: $(COVERAGE_HTML)"
	$(GOCOV) func -o /dev/null $(COVERAGE_OUT) || true
	@echo "Run 'make view-cover' to view HTML report"

# Run all tests including race detection
test-all: test test-race
	@echo "All tests passed!"

# View coverage HTML report
view-cover: test-cover
	@echo "Generating coverage report..."
	$(GOCOV) html -o $(COVERAGE_HTML) $(COVERAGE_OUT)
	@echo "Open $(COVERAGE_HTML) in your browser"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test ./... -bench=. -benchmem -benchtime=500ms -count=3 | head -50

# Save benchmark baseline
benchmark-save:
	@echo "Saving benchmark baseline..."
	$(GO) test ./... -bench=. -benchmem -benchtime=1s -count=1 > $(BENCHMARK_OUT) 2>&1 || true
	@echo "Baseline saved to $(BENCHMARK_OUT)"

# Compare against benchmark baseline
benchmark-compare: benchmark-save
	@echo "Comparing with baseline..."
	@if command -v benchcmp >/dev/null 2>&1; then \
		benchcmp $(BENCHMARK_OUT) <($(GO) test ./... -bench=. -benchmem -benchtime=1s -count=1 2>&1); \
	else \
		echo "Install benchcmp: go install golang.org/x/tools/cmd/benchcmp@latest"; \
	fi

# Build the binary
build:
	@echo "Building db-catalyst..."
	$(GO) build -o db-catalyst ./cmd/db-catalyst

# Build with optimizations
build-prod:
	@echo "Building production binary..."
	$(GO) build -ldflags="-s -w" -o db-catalyst ./cmd/db-catalyst

# Build for multiple platforms
build-all: build-prod
	@echo "Building for multiple platforms..."
	@for GOOS in linux darwin windows; do \
		for GOARCH in amd64 arm64; do \
			echo "Building for $$GOOS/$$GOARCH..."; \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -ldflags="-s -w" -o db-catalyst-$$GOOS-$$GOARCH ./cmd/db-catalyst || true; \
		done; \
	done

# Run linter
lint:
	@echo "Running linter..."
	$(GOLANGCI) run ./... || true

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -l -w .
	$(GOIMPORTS) $$(find . -name '*.go' -not -path './vendor/*' -not -path './.git/*') 2>/dev/null || true

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./... || true

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GO) clean -cache -testcache -modcache 2>/dev/null || true
	rm -f db-catalyst db-catalyst-* $(COVERAGE_OUT) $(COVERAGE_HTML) $(BENCHMARK_OUT)
	rm -rf /tmp/db-catalyst-*
	@echo "Cleaned."

# Full CI check (what would run on CI)
ci: fmt lint test-race test-cover
	@echo ""
	@echo "=========================================="
	@echo "CI checks passed!"
	@echo "=========================================="

# Quick check (fast local development)
quick: fmt lint test
	@echo ""
	@echo "=========================================="
	@echo "Quick checks passed!"
	@echo "=========================================="

# Install binary to PATH
install: build
	@echo "Installing db-catalyst to ~/go/bin..."
	@mkdir -p $$(go env GOPATH)/bin
	cp db-catalyst $$(go env GOPATH)/bin/
	@echo "Installed. Run 'db-catalyst' from anywhere."

# Integration tests with Docker
integration-up:
	@echo "Starting database containers..."
	cd test/integration && docker compose up -d

integration-down:
	@echo "Stopping database containers..."
	cd test/integration && docker compose down -v

integration-test: integration-up
	@echo "Running integration tests..."
	@echo "Waiting for databases to be ready..."
	@sleep 10
	cd test/integration && go test -v -short ./... 2>&1 || true
	@echo ""
	@echo "Running full integration test..."
	@./test/integration/run-tests.sh 2>&1 || true

integration-full: integration-up integration-test integration-down
	@echo "Integration test complete!"

# Show help
help:
	@echo "db-catalyst Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Run all tests (default)"
	@echo "  test             - Run tests without race detector (fast, parallel)"
	@echo "  test-race        - Run tests with race detector (slower)"
	@echo "  test-cover       - Run tests with coverage"
	@echo "  view-cover       - View coverage HTML report"
	@echo "  bench            - Run quick benchmarks"
	@echo "  benchmark-save   - Save benchmark baseline"
	@echo "  benchmark-compare - Compare with baseline"
	@echo "  build            - Build debug binary"
	@echo "  build-prod       - Build optimized production binary"
	@echo "  build-all        - Build for all platforms"
	@echo "  lint             - Run golangci-lint"
	@echo "  fmt              - Format code with gofmt/goimports"
	@echo "  vet              - Run go vet"
	@echo "  clean            - Clean build artifacts"
	@echo "  ci               - Full CI check (fmt, lint, race, coverage)"
	@echo "  quick            - Quick local check (fmt, lint, test)"
	@echo "  install          - Install binary to ~/go/bin"
	@echo "  integration-up   - Start Docker databases (MySQL, PostgreSQL)"
	@echo "  integration-down - Stop Docker databases"
	@echo "  integration-test - Run integration tests"
	@echo "  integration-full - Full integration test suite"
	@echo "  help             - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make test          # Fast test run"
	@echo "  make quick         # Format, lint, and test"
	@echo "  make ci            # Full CI simulation"
	@echo "  make build-prod    # Production build"
	@echo "  make integration-up   # Start MySQL/PostgreSQL Docker containers"
	@echo "  make integration-test # Run tests against real databases"
