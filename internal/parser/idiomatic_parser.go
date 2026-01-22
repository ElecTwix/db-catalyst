// Package parser demonstrates idomatic Go patterns for database-catalyst.
// This file is a reference implementation showing best practices.
package parser

import (
	"context"
	"fmt"

	"github.com/electwix/db-catalyst/internal/schema/model"
	"log/slog"
)

// ParserOption configures parser behavior using the functional options pattern.
type ParserOption func(*Parser)

// WithDebug enables debug logging for parser operations.
func WithDebug(debug bool) ParserOption {
	return func(p *Parser) {
		p.debug = debug
	}
}

// WithMaxErrors limits the number of errors collected before aborting.
func WithMaxErrors(max int) ParserOption {
	return func(p *Parser) {
		p.maxErrors = max
	}
}

// SchemaParser defines the interface for parsing schema definitions.
// This interface allows different parsing implementations (SQLite, PostgreSQL, GraphQL, etc.)
type SchemaParser interface {
	Parse(ctx context.Context, input string) (*model.Catalog, error)
}

// Parser implements SchemaParser with idiomatic Go patterns.
type Parser struct {
	debug      bool
	maxErrors  int
	errorCount int
}

// NewParser creates a new Parser with the given options.
func NewParser(opts ...ParserOption) SchemaParser {
	p := &Parser{
		maxErrors: 10,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Parse parses the input and returns a catalog.
// Context is the first parameter and checked for cancellation.
func (p *Parser) Parse(ctx context.Context, input string) (*model.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("parse cancelled: %w", err)
	}

	catalog := model.NewCatalog()
	var parseErrors []error

	stmts, err := p.parseStatements(input)
	if err != nil {
		return nil, fmt.Errorf("parse statements: %w", err)
	}

	for _, stmt := range stmts {
		if p.errorCount >= p.maxErrors {
			p.logError("max errors (%d) reached, aborting", p.maxErrors)
			break
		}

		table, err := p.parseTable(stmt)
		if err != nil {
			parseErrors = append(parseErrors, err)
			p.errorCount++
		} else {
			catalog.Tables[table.Name] = table
		}
	}

	if len(parseErrors) > 0 {
		// Log multiple errors using slog
		logger := slog.Default()
		for i, err := range parseErrors {
			logger.Error("parse error", "index", i, "error", err)
		}
		return nil, fmt.Errorf("parse failed with %d error(s): %w", len(parseErrors), parseErrors[0])
	}

	if p.debug {
		p.logDebug("parsed catalog with %d tables", len(catalog.Tables))
	}

	return catalog, nil
}

// logDebug is a placeholder for debug logging.
func (p *Parser) logDebug(format string, args ...any) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

// logError is a placeholder for error logging.
func (p *Parser) logError(format string, args ...any) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// parseStatements is a placeholder for statement parsing logic.
func (p *Parser) parseStatements(input string) ([]string, error) {
	return []string{"CREATE TABLE users (id INTEGER, name TEXT)"}, nil
}

// parseTable is a placeholder for table parsing logic.
func (p *Parser) parseTable(stmt string) (*model.Table, error) {
	return &model.Table{
		Name: "users",
		Columns: []*model.Column{
			{Name: "id", Type: "INTEGER"},
			{Name: "name", Type: "TEXT"},
		},
	}, nil
}
