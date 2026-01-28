// Package parser provides a high-level SQL parser with functional options.
package parser

import (
	"context"
	"errors"
	"fmt"

	"github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Option configures a Parser using the functional options pattern.
type Option func(*Parser)

// Parser parses SQL DDL statements and produces a schema catalog.
type Parser struct {
	debug     bool
	maxErrors int
}

// NewParser creates a new Parser with the provided functional options.
func NewParser(options ...Option) *Parser {
	p := &Parser{
		debug:     false,
		maxErrors: 10,
	}

	for _, opt := range options {
		opt(p)
	}

	return p
}

// WithDebug enables or disables debug mode for the parser.
func WithDebug(enabled bool) Option {
	return func(p *Parser) {
		p.debug = enabled
	}
}

// WithMaxErrors sets the maximum number of errors to collect before stopping.
func WithMaxErrors(maxErrors int) Option {
	return func(p *Parser) {
		p.maxErrors = maxErrors
	}
}

// Parse parses the provided SQL input and returns a catalog of schema objects.
// It respects context cancellation and will return an error if the context is cancelled.
func (p *Parser) Parse(ctx context.Context, input string) (*model.Catalog, error) {
	// Check for context cancellation before starting
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("parse cancelled: %w", err)
	}

	// Tokenize the input
	tokens, err := tokenizer.Scan("", []byte(input), false)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Check for context cancellation after tokenization
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("parse cancelled: %w", err)
	}

	// Parse the tokens into a catalog
	catalog, diagnostics, err := schemaparser.Parse("", tokens)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Check for context cancellation after parsing
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("parse cancelled: %w", err)
	}

	// Count errors and check against maxErrors limit
	errorCount := 0
	for _, diag := range diagnostics {
		if diag.Severity == schemaparser.SeverityError {
			errorCount++
			if errorCount >= p.maxErrors {
				return nil, errors.New("too many parse errors")
			}
		}
	}

	// In debug mode, we could log diagnostics here
	if p.debug {
		// Debug logging would go here if a logger was injected
		_ = diagnostics
	}

	return catalog, nil
}
