// Package diagnostic provides shared types for schema parsing diagnostics.
//
// This package exists to avoid import cycles between the main parser package
// and dialect-specific parser implementations (e.g., PostgreSQL parser).
package diagnostic

import (
	"context"

	"github.com/electwix/db-catalyst/internal/schema/model"
)

// Severity indicates the seriousness of a diagnostic.
type Severity int

const (
	// SeverityError indicates a fatal issue that prevents code generation.
	SeverityError Severity = iota
	// SeverityWarning indicates a potential issue that doesn't prevent code generation.
	SeverityWarning
)

// Diagnostic captures parser feedback for callers to display.
type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

// SchemaParser parses SQL DDL statements and produces a schema catalog.
type SchemaParser interface {
	Parse(ctx context.Context, path string, content []byte) (*model.Catalog, []Diagnostic, error)
}
