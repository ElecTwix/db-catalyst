# Contributing to db-catalyst

Thank you for your interest in contributing to db-catalyst! This document provides guidelines for contributing.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues. When creating a bug report, include:

- **Use a clear descriptive title**
- **Describe the exact steps to reproduce**
- **Provide specific examples** (SQL schema, queries, config)
- **Describe the behavior you observed**
- **Explain which behavior you expected**

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. Include:

- **Use a clear descriptive title**
- **Provide a step-by-step description**
- **Provide specific examples**
- **Explain why this enhancement would be useful**

### Pull Requests

1. Fork the repository
2. Create a branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`task test-all`)
5. Run linting (`task lint-all`)
6. Commit (`git commit -m 'feat: add amazing feature'`)
7. Push (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## Development Setup

### Prerequisites

- Go 1.25+
- Task (`go install github.com/go-task/task/v3/cmd/task@latest`)
- golangci-lint

### Quick Start

```bash
# Clone
git clone https://github.com/ElecTwix/db-catalyst.git
cd db-catalyst

# Install dependencies
task install-deps

# Build
task build

# Test
task test

# Run full quality check
task quality
```

## Style Guidelines

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` and `goimports`
- Pass `golangci-lint` checks
- Use context-first, error-last
- Avoid global state

### Commits

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `style:` Formatting
- `refactor:` Code restructuring
- `test:` Tests
- `chore:` Maintenance

Example:
```
feat(parser): add support for PostgreSQL arrays

- Add ArrayType to type definitions
- Update parser to handle array syntax
- Add tests for array parsing
```

### Testing

- Write table-driven tests
- Use `cmp.Diff` for comparisons
- Target 80%+ coverage for new code
- Add integration tests for features

## Project Structure

```
cmd/           CLI commands
internal/      Private packages
  cache/       Caching infrastructure
  codegen/     Code generation
  config/      Configuration parsing
  parser/      SQL parsers
  pipeline/    Build pipeline
  query/       Query analysis
  schema/      Schema parsing
  transform/   Type transformations
docs/          Documentation
test/          Integration tests
```

## Questions?

Feel free to open an issue for questions or join discussions.
