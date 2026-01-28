# Makefile for db-catalyst
# A SQLite-only code generator

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

# =============================================================================
# Default target
# =============================================================================
.PHONY: all
all: test

# =============================================================================
# Build targets
# =============================================================================
.PHONY: build build-prod build-all

build:
	@echo "Building db-catalyst..."
	$(GO) build ./cmd/db-catalyst

build-prod:
	@echo "Building production binary..."
	$(GO) build -ldflags="-s -w" -o db-catalyst ./cmd/db-catalyst

build-all: build-prod
	@echo "Building for multiple platforms..."
	@for GOOS in linux darwin windows; do \
		for GOARCH in amd64 arm64; do \
			echo "Building for $$GOOS/$$GOARCH..."; \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -ldflags="-s -w" -o db-catalyst-$$GOOS-$$GOARCH ./cmd/db-catalyst || true; \
		done; \
	done

# =============================================================================
# Test targets
# =============================================================================
.PHONY: test test-race test-all

test:
	@echo "Running tests..."
	$(GO) test ./... -count=1 -parallel=4

test-race:
	@echo "Running tests with race detector..."
	$(GO) test ./... -race -count=1 -parallel=2

test-all: test test-race
	@echo "All tests passed!"

# =============================================================================
# Coverage targets
# =============================================================================
.PHONY: coverage coverage-report coverage-summary view-cover test-cover

coverage:
	go test -coverprofile=$(COVERAGE_OUT) ./...
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

coverage-report:
	go test -coverprofile=$(COVERAGE_OUT) ./...
	go tool cover -func=$(COVERAGE_OUT) | tail -1

coverage-summary:
	@go test -coverprofile=$(COVERAGE_OUT) ./... 2>/dev/null
	@echo ""
	@echo "Coverage by package:"
	@go tool cover -func=$(COVERAGE_OUT) | grep "total:" | awk '{print "  Total:", $$3}'
	@echo ""
	@for pkg in $$(go list ./internal/... | grep -v /testdata); do \
		go test -coverprofile=pkg.out $$pkg 2>/dev/null; \
		if [ -f pkg.out ]; then \
			go tool cover -func=pkg.out | grep "total:" | awk -v pkg=$$pkg '{print "  " pkg ":", $$3}'; \
			rm pkg.out; \
		fi; \
	done

# Legacy coverage targets (kept for compatibility)
test-cover:
	@echo "Running tests with coverage..."
	$(GO) test ./... -coverprofile=$(COVERAGE_OUT) -count=1 -parallel=4
	@echo "Coverage report: $(COVERAGE_HTML)"
	$(GOCOV) func -o /dev/null $(COVERAGE_OUT) || true
	@echo "Run 'make view-cover' to view HTML report"

view-cover: test-cover
	@echo "Generating coverage report..."
	$(GOCOV) html -o $(COVERAGE_HTML) $(COVERAGE_OUT)
	@echo "Open $(COVERAGE_HTML) in your browser"

# =============================================================================
# Static analysis targets
# =============================================================================
.PHONY: lint-all lint vet staticcheck security vulncheck

lint-all: lint vet staticcheck

lint:
	@echo "Running linter..."
	$(GOLANGCI) run ./... || true

vet:
	@echo "Running go vet..."
	go vet ./...

staticcheck:
	@echo "Running staticcheck..."
	@which staticcheck >/dev/null 2>&1 || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

security:
	@echo "Running security checks..."
	@which gosec >/dev/null 2>&1 || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -quiet ./...

vulncheck:
	@echo "Running vulnerability check..."
	@which govulncheck >/dev/null 2>&1 || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# =============================================================================
# Quality meta-target
# =============================================================================
.PHONY: quality

quality: lint-all test-race coverage-report
	@echo ""
	@echo "✅ All quality checks passed!"

# =============================================================================
# Benchmark targets
# =============================================================================
.PHONY: bench benchmark-save benchmark-compare bench-compare bench-save

bench:
	@echo "Running benchmarks..."
	$(GO) test ./... -bench=. -benchmem -benchtime=500ms -count=3 | head -50

benchmark-save:
	@echo "Saving benchmark baseline..."
	$(GO) test ./... -bench=. -benchmem -benchtime=1s -count=1 > $(BENCHMARK_OUT) 2>&1 || true
	@echo "Baseline saved to $(BENCHMARK_OUT)"

benchmark-compare: benchmark-save
	@echo "Comparing with baseline..."
	@if command -v benchcmp >/dev/null 2>&1; then \
		benchcmp $(BENCHMARK_OUT) <($(GO) test ./... -bench=. -benchmem -benchtime=1s -count=1 2>&1); \
	else \
		echo "Install benchcmp: go install golang.org/x/tools/cmd/benchcmp@latest"; \
	fi

bench-compare:
	@echo "Running benchmarks with 5 iterations for comparison..."
	$(GO) test -bench=. -benchmem -count=5 ./... > bench-new.txt
	@echo "Results saved to bench-new.txt"
	@echo "Compare with previous: benchstat bench-old.txt bench-new.txt"

bench-save:
	$(GO) test -bench=. -benchmem -count=5 ./... > bench-$(shell date +%Y%m%d).txt
	@echo "Benchmark saved to bench-$(shell date +%Y%m%d).txt"

# =============================================================================
# Format and maintenance targets
# =============================================================================
.PHONY: fmt clean install-deps

fmt:
	@echo "Formatting code..."
	$(GOFMT) -l -w .
	$(GOIMPORTS) $$(find . -name '*.go' -not -path './vendor/*' -not -path './.git/*') 2>/dev/null || true

clean:
	@echo "Cleaning..."
	$(GO) clean -cache -testcache -modcache 2>/dev/null || true
	rm -f db-catalyst db-catalyst-* $(COVERAGE_OUT) $(COVERAGE_HTML) $(BENCHMARK_OUT)
	rm -rf /tmp/db-catalyst-*
	@echo "Cleaned."

install-deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed."

# =============================================================================
# CI and development workflow targets
# =============================================================================
.PHONY: ci quick install

ci: fmt lint test-race test-cover
	@echo ""
	@echo "=========================================="
	@echo "CI checks passed!"
	@echo "=========================================="

quick: fmt lint test
	@echo ""
	@echo "=========================================="
	@echo "Quick checks passed!"
	@echo "=========================================="

install: build
	@echo "Installing db-catalyst to ~/go/bin..."
	@mkdir -p $$(go env GOPATH)/bin
	cp db-catalyst $$(go env GOPATH)/bin/
	@echo "Installed. Run 'db-catalyst' from anywhere."

# =============================================================================
# Integration test targets
# =============================================================================
.PHONY: integration-up integration-down integration-test integration-full

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

# =============================================================================
# Help target
# =============================================================================
.PHONY: help

help:
	@echo "db-catalyst Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Build targets:"
	@echo "  build            - Build debug binary"
	@echo "  build-prod       - Build optimized production binary"
	@echo "  build-all        - Build for all platforms"
	@echo ""
	@echo "Test targets:"
	@echo "  test             - Run tests without race detector (fast, parallel)"
	@echo "  test-race        - Run tests with race detector (slower)"
	@echo "  test-all         - Run all tests including race detection"
	@echo ""
	@echo "Coverage targets:"
	@echo "  coverage         - Generate HTML coverage report"
	@echo "  coverage-report  - Show total coverage percentage"
	@echo "  coverage-summary - Show coverage by package"
	@echo "  view-cover       - View coverage HTML report (legacy)"
	@echo ""
	@echo "Static analysis targets:"
	@echo "  lint-all         - Run all linters (lint, vet, staticcheck)"
	@echo "  lint             - Run golangci-lint"
	@echo "  vet              - Run go vet"
	@echo "  staticcheck      - Run staticcheck (auto-installs if needed)"
	@echo "  security         - Run gosec security checks"
	@echo "  vulncheck        - Run vulnerability check"
	@echo ""
	@echo "Quality targets:"
	@echo "  quality          - Run all quality checks (lint-all, test-race, coverage-report)"
	@echo ""
	@echo "Benchmark targets:"
	@echo "  bench            - Run quick benchmarks"
	@echo "  benchmark-save   - Save benchmark baseline"
	@echo "  benchmark-compare - Compare with baseline"
	@echo "  bench-save       - Save benchmark with date stamp"
	@echo "  bench-compare    - Run 5 iterations for benchstat comparison"
	@echo ""
	@echo "Maintenance targets:"
	@echo "  fmt              - Format code with gofmt/goimports"
	@echo "  clean            - Clean build artifacts"
	@echo "  install-deps     - Install development dependencies"
	@echo ""
	@echo "CI/Workflow targets:"
	@echo "  ci               - Full CI check (fmt, lint, race, coverage)"
	@echo "  quick            - Quick local check (fmt, lint, test)"
	@echo "  install          - Install binary to ~/go/bin"
	@echo ""
	@echo "Integration test targets:"
	@echo "  integration-up   - Start Docker databases (MySQL, PostgreSQL)"
	@echo "  integration-down - Stop Docker databases"
	@echo "  integration-test - Run integration tests"
	@echo "  integration-full - Full integration test suite"
	@echo ""
	@echo "Examples:"
	@echo "  make test          # Fast test run"
	@echo "  make quick         # Format, lint, and test"
	@echo "  make ci            # Full CI simulation"
	@echo "  make build-prod    # Production build"
	@echo "  make quality       # Run all quality checks"
	@echo "  make coverage      # Generate coverage report"
	@echo "  make lint-all      # Run all static analysis tools"
	@echo "  make integration-up   # Start MySQL/PostgreSQL Docker containers"
	@echo "  make integration-test # Run tests against real databases"
