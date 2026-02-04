// Package diagnostics provides rich diagnostic information for db-catalyst.
// It captures file locations, contextual code snippets, suggestions for fixes,
// and severity levels to help users understand and resolve issues.
package diagnostics

import (
	"fmt"
	"strings"
)

// Severity indicates the seriousness of a diagnostic.
type Severity int

const (
	// SeverityInfo indicates an informational message.
	SeverityInfo Severity = iota
	// SeverityWarning indicates a potential issue that doesn't prevent code generation.
	SeverityWarning
	// SeverityError indicates a fatal issue that prevents code generation.
	SeverityError
)

// String returns the string representation of the severity level.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// SeverityFromString parses a severity level from a string.
func SeverityFromString(s string) Severity {
	switch strings.ToLower(s) {
	case "info":
		return SeverityInfo
	case "warning", "warn":
		return SeverityWarning
	case "error", "err":
		return SeverityError
	default:
		return SeverityWarning
	}
}

// Location represents a position in a source file.
type Location struct {
	Path   string
	Line   int
	Column int
	Offset int
}

// Span represents a range of locations in a source file.
type Span struct {
	Start Location
	End   Location
}

// Suggestion represents a suggested fix for a diagnostic.
type Suggestion struct {
	Message     string
	Replacement string
	Span        Span
}

// RelatedInfo represents related context for a diagnostic.
type RelatedInfo struct {
	Location Location
	Message  string
}

// Diagnostic represents a rich diagnostic message with context.
type Diagnostic struct {
	// Core diagnostic information
	Severity Severity
	Message  string
	Code     string // Optional error code (e.g., "E001", "W003")

	// Location information
	Location Location
	Span     *Span // Optional span for multi-line diagnostics

	// Context
	Context string // Code snippet showing the problematic area

	// Help information
	Suggestions []Suggestion  // Suggested fixes
	Notes       []string      // Additional notes explaining the issue
	Related     []RelatedInfo // Related locations (e.g., previous definition)

	// Source information
	Source string // Component that produced the diagnostic (e.g., "schema-parser", "query-analyzer")
}

// HasLocation returns true if the diagnostic has a valid location.
func (d Diagnostic) HasLocation() bool {
	return d.Location.Path != "" && d.Location.Line > 0
}

// HasSpan returns true if the diagnostic has a valid span.
func (d Diagnostic) HasSpan() bool {
	return d.Span != nil && d.Span.Start.Path != ""
}

// IsError returns true if the diagnostic is an error.
func (d Diagnostic) IsError() bool {
	return d.Severity == SeverityError
}

// IsWarning returns true if the diagnostic is a warning.
func (d Diagnostic) IsWarning() bool {
	return d.Severity == SeverityWarning
}

// IsInfo returns true if the diagnostic is informational.
func (d Diagnostic) IsInfo() bool {
	return d.Severity == SeverityInfo
}

// Error implements the error interface for error-level diagnostics.
func (d Diagnostic) Error() string {
	if d.Code != "" {
		return fmt.Sprintf("%s:%d:%d: [%s] %s: %s",
			d.Location.Path, d.Location.Line, d.Location.Column,
			d.Code, d.Severity, d.Message)
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		d.Location.Path, d.Location.Line, d.Location.Column,
		d.Severity, d.Message)
}

// String returns a human-readable string representation of the diagnostic.
func (d Diagnostic) String() string {
	var b strings.Builder

	// Header: path:line:column: severity: message
	if d.HasLocation() {
		fmt.Fprintf(&b, "%s:%d:%d: ", d.Location.Path, d.Location.Line, d.Location.Column)
	}
	fmt.Fprintf(&b, "%s: %s", d.Severity, d.Message)

	// Code
	if d.Code != "" {
		fmt.Fprintf(&b, " [%s]", d.Code)
	}

	// Source
	if d.Source != "" {
		fmt.Fprintf(&b, " (%s)", d.Source)
	}

	// Context
	if d.Context != "" {
		fmt.Fprintf(&b, "\n  --> %s", d.Context)
	}

	// Suggestions
	for _, sugg := range d.Suggestions {
		fmt.Fprintf(&b, "\n  suggestion: %s", sugg.Message)
		if sugg.Replacement != "" {
			fmt.Fprintf(&b, "\n    replace with: %s", sugg.Replacement)
		}
	}

	// Notes
	for _, note := range d.Notes {
		fmt.Fprintf(&b, "\n  note: %s", note)
	}

	// Related info
	for _, rel := range d.Related {
		fmt.Fprintf(&b, "\n  related: %s:%d:%d: %s",
			rel.Location.Path, rel.Location.Line, rel.Location.Column, rel.Message)
	}

	return b.String()
}

// Builder provides a fluent API for constructing diagnostics.
type Builder struct {
	diag Diagnostic
}

// NewBuilder creates a new diagnostic builder with the given severity and message.
func NewBuilder(severity Severity, message string) *Builder {
	return &Builder{
		diag: Diagnostic{
			Severity: severity,
			Message:  message,
		},
	}
}

// Error creates a builder for an error-level diagnostic.
func Error(message string) *Builder {
	return NewBuilder(SeverityError, message)
}

// Warning creates a builder for a warning-level diagnostic.
func Warning(message string) *Builder {
	return NewBuilder(SeverityWarning, message)
}

// Info creates a builder for an info-level diagnostic.
func Info(message string) *Builder {
	return NewBuilder(SeverityInfo, message)
}

// WithCode sets the error code.
func (b *Builder) WithCode(code string) *Builder {
	b.diag.Code = code
	return b
}

// At sets the location.
func (b *Builder) At(path string, line, column int) *Builder {
	b.diag.Location = Location{
		Path:   path,
		Line:   line,
		Column: column,
	}
	return b
}

// AtLocation sets the location from a Location struct.
func (b *Builder) AtLocation(loc Location) *Builder {
	b.diag.Location = loc
	return b
}

// WithSpan sets the span.
func (b *Builder) WithSpan(start, end Location) *Builder {
	b.diag.Span = &Span{Start: start, End: end}
	return b
}

// WithContext sets the code context.
func (b *Builder) WithContext(context string) *Builder {
	b.diag.Context = context
	return b
}

// WithSource sets the source component.
func (b *Builder) WithSource(source string) *Builder {
	b.diag.Source = source
	return b
}

// WithSuggestion adds a suggestion.
func (b *Builder) WithSuggestion(message, replacement string) *Builder {
	b.diag.Suggestions = append(b.diag.Suggestions, Suggestion{
		Message:     message,
		Replacement: replacement,
	})
	return b
}

// WithNote adds a note.
func (b *Builder) WithNote(note string) *Builder {
	b.diag.Notes = append(b.diag.Notes, note)
	return b
}

// WithRelated adds related information.
func (b *Builder) WithRelated(path string, line, column int, message string) *Builder {
	b.diag.Related = append(b.diag.Related, RelatedInfo{
		Location: Location{Path: path, Line: line, Column: column},
		Message:  message,
	})
	return b
}

// Build returns the constructed diagnostic.
func (b *Builder) Build() Diagnostic {
	return b.diag
}

// Collection holds a set of diagnostics.
type Collection struct {
	diagnostics []Diagnostic
}

// NewCollection creates a new empty diagnostic collection.
func NewCollection() *Collection {
	return &Collection{
		diagnostics: make([]Diagnostic, 0),
	}
}

// Add adds a diagnostic to the collection.
func (c *Collection) Add(d Diagnostic) {
	c.diagnostics = append(c.diagnostics, d)
}

// AddAll adds all diagnostics from another collection.
func (c *Collection) AddAll(other *Collection) {
	c.diagnostics = append(c.diagnostics, other.diagnostics...)
}

// HasErrors returns true if the collection contains any errors.
func (c *Collection) HasErrors() bool {
	for _, d := range c.diagnostics {
		if d.IsError() {
			return true
		}
	}
	return false
}

// Errors returns all error-level diagnostics.
func (c *Collection) Errors() []Diagnostic {
	var errs []Diagnostic
	for _, d := range c.diagnostics {
		if d.IsError() {
			errs = append(errs, d)
		}
	}
	return errs
}

// Warnings returns all warning-level diagnostics.
func (c *Collection) Warnings() []Diagnostic {
	var warns []Diagnostic
	for _, d := range c.diagnostics {
		if d.IsWarning() {
			warns = append(warns, d)
		}
	}
	return warns
}

// All returns all diagnostics.
func (c *Collection) All() []Diagnostic {
	return append([]Diagnostic(nil), c.diagnostics...)
}

// Len returns the number of diagnostics.
func (c *Collection) Len() int {
	return len(c.diagnostics)
}

// Filter returns diagnostics matching the given predicate.
func (c *Collection) Filter(predicate func(Diagnostic) bool) []Diagnostic {
	var result []Diagnostic
	for _, d := range c.diagnostics {
		if predicate(d) {
			result = append(result, d)
		}
	}
	return result
}

// BySeverity returns diagnostics of a specific severity.
func (c *Collection) BySeverity(severity Severity) []Diagnostic {
	return c.Filter(func(d Diagnostic) bool {
		return d.Severity == severity
	})
}

// BySource returns diagnostics from a specific source.
func (c *Collection) BySource(source string) []Diagnostic {
	return c.Filter(func(d Diagnostic) bool {
		return d.Source == source
	})
}

// ByCode returns diagnostics with a specific error code.
func (c *Collection) ByCode(code string) []Diagnostic {
	return c.Filter(func(d Diagnostic) bool {
		return d.Code == code
	})
}

// SortByLocation sorts diagnostics by file path and line number.
func (c *Collection) SortByLocation() {
	// Simple bubble sort for now - could be optimized
	n := len(c.diagnostics)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if compareLocation(c.diagnostics[j].Location, c.diagnostics[j+1].Location) > 0 {
				c.diagnostics[j], c.diagnostics[j+1] = c.diagnostics[j+1], c.diagnostics[j]
			}
		}
	}
}

func compareLocation(a, b Location) int {
	if a.Path != b.Path {
		if a.Path < b.Path {
			return -1
		}
		return 1
	}
	if a.Line != b.Line {
		return a.Line - b.Line
	}
	return a.Column - b.Column
}

// Summary provides a quick overview of diagnostics.
type Summary struct {
	Total    int
	Errors   int
	Warnings int
	Infos    int
}

// Summary returns a summary of the diagnostics collection.
func (c *Collection) Summary() Summary {
	s := Summary{Total: len(c.diagnostics)}
	for _, d := range c.diagnostics {
		switch d.Severity {
		case SeverityError:
			s.Errors++
		case SeverityWarning:
			s.Warnings++
		case SeverityInfo:
			s.Infos++
		}
	}
	return s
}

// ErrorCodes provides standardized error codes for diagnostics.
// These codes help users identify and search for specific issues.
const (
	// Schema parsing errors (E1xx)
	ErrSchemaParseError     = "E101"
	ErrSchemaDuplicateTable = "E102"
	ErrSchemaDuplicateView  = "E103"
	ErrSchemaDuplicateCol   = "E104"
	ErrSchemaInvalidType    = "E105"
	ErrSchemaUnknownTable   = "E106"
	ErrSchemaUnknownColumn  = "E107"
	ErrSchemaInvalidFK      = "E108"
	ErrSchemaInvalidPK      = "E109"
	ErrSchemaInvalidIndex   = "E110"

	// Query parsing errors (E2xx)
	ErrQueryParseError    = "E201"
	ErrQueryInvalidVerb   = "E202"
	ErrQueryInvalidParam  = "E203"
	ErrQueryInvalidCTE    = "E204"
	ErrQueryMissingAlias  = "E205"
	ErrQueryAmbiguousCol  = "E206"
	ErrQueryUnknownTable  = "E207"
	ErrQueryUnknownColumn = "E208"
	ErrQueryTypeMismatch  = "E209"
	ErrQueryInvalidSyntax = "E210"

	// Configuration errors (E3xx)
	ErrConfigInvalid        = "E301"
	ErrConfigMissingPackage = "E302"
	ErrConfigMissingOut     = "E303"
	ErrConfigInvalidPath    = "E304"
	ErrConfigUnknownKey     = "E305"
	ErrConfigInvalidDriver  = "E306"
	ErrConfigInvalidLang    = "E307"
	ErrConfigInvalidDB      = "E308"

	// Code generation errors (E4xx)
	ErrCodeGenFailed      = "E401"
	ErrCodeGenWriteFailed = "E402"
	ErrCodeGenTypeError   = "E403"

	// Warnings (W1xx)
	WarnDeprecatedFeature = "W101"
	WarnUnnecessaryAlias  = "W102"
	WarnTypeInference     = "W103"
	WarnUnusedParam       = "W104"
	WarnSchemaMismatch    = "W105"
)

// CodeDescription returns a human-readable description for an error code.
func CodeDescription(code string) string {
	descriptions := map[string]string{
		// Schema errors
		ErrSchemaParseError:     "Schema parsing failed",
		ErrSchemaDuplicateTable: "Duplicate table definition",
		ErrSchemaDuplicateView:  "Duplicate view definition",
		ErrSchemaDuplicateCol:   "Duplicate column definition",
		ErrSchemaInvalidType:    "Invalid column type",
		ErrSchemaUnknownTable:   "Reference to unknown table",
		ErrSchemaUnknownColumn:  "Reference to unknown column",
		ErrSchemaInvalidFK:      "Invalid foreign key constraint",
		ErrSchemaInvalidPK:      "Invalid primary key constraint",
		ErrSchemaInvalidIndex:   "Invalid index definition",

		// Query errors
		ErrQueryParseError:    "Query parsing failed",
		ErrQueryInvalidVerb:   "Invalid or unsupported SQL verb",
		ErrQueryInvalidParam:  "Invalid parameter syntax",
		ErrQueryInvalidCTE:    "Invalid common table expression",
		ErrQueryMissingAlias:  "Missing required alias",
		ErrQueryAmbiguousCol:  "Ambiguous column reference",
		ErrQueryUnknownTable:  "Reference to unknown table",
		ErrQueryUnknownColumn: "Reference to unknown column",
		ErrQueryTypeMismatch:  "Type mismatch in expression",
		ErrQueryInvalidSyntax: "Invalid SQL syntax",

		// Config errors
		ErrConfigInvalid:        "Invalid configuration",
		ErrConfigMissingPackage: "Missing required package name",
		ErrConfigMissingOut:     "Missing required output directory",
		ErrConfigInvalidPath:    "Invalid file path",
		ErrConfigUnknownKey:     "Unknown configuration key",
		ErrConfigInvalidDriver:  "Invalid SQLite driver",
		ErrConfigInvalidLang:    "Invalid target language",
		ErrConfigInvalidDB:      "Invalid database dialect",

		// Code generation errors
		ErrCodeGenFailed:      "Code generation failed",
		ErrCodeGenWriteFailed: "Failed to write generated file",
		ErrCodeGenTypeError:   "Type error in code generation",

		// Warnings
		WarnDeprecatedFeature: "Deprecated feature used",
		WarnUnnecessaryAlias:  "Unnecessary alias",
		WarnTypeInference:     "Type inference used default",
		WarnUnusedParam:       "Unused parameter",
		WarnSchemaMismatch:    "Schema mismatch detected",
	}

	if desc, ok := descriptions[code]; ok {
		return desc
	}
	return "Unknown error code"
}

// CategorizedSummary groups diagnostics by category based on error codes.
type CategorizedSummary struct {
	SchemaErrors  []Diagnostic
	QueryErrors   []Diagnostic
	ConfigErrors  []Diagnostic
	CodegenErrors []Diagnostic
	Warnings      []Diagnostic
	Infos         []Diagnostic
	Uncategorized []Diagnostic
}

// Categorize groups diagnostics by their error code category.
func (c *Collection) Categorize() CategorizedSummary {
	var result CategorizedSummary

	for _, d := range c.diagnostics {
		if d.Code == "" {
			result.Uncategorized = append(result.Uncategorized, d)
			continue
		}

		switch {
		case strings.HasPrefix(d.Code, "E1"):
			result.SchemaErrors = append(result.SchemaErrors, d)
		case strings.HasPrefix(d.Code, "E2"):
			result.QueryErrors = append(result.QueryErrors, d)
		case strings.HasPrefix(d.Code, "E3"):
			result.ConfigErrors = append(result.ConfigErrors, d)
		case strings.HasPrefix(d.Code, "E4"):
			result.CodegenErrors = append(result.CodegenErrors, d)
		case strings.HasPrefix(d.Code, "W"):
			result.Warnings = append(result.Warnings, d)
		case strings.HasPrefix(d.Code, "I"):
			result.Infos = append(result.Infos, d)
		default:
			result.Uncategorized = append(result.Uncategorized, d)
		}
	}

	return result
}

// HasDiagnostics returns true if any category has diagnostics.
func (cs CategorizedSummary) HasDiagnostics() bool {
	return len(cs.SchemaErrors) > 0 ||
		len(cs.QueryErrors) > 0 ||
		len(cs.ConfigErrors) > 0 ||
		len(cs.CodegenErrors) > 0 ||
		len(cs.Warnings) > 0 ||
		len(cs.Infos) > 0 ||
		len(cs.Uncategorized) > 0
}

// Total returns the total number of diagnostics across all categories.
func (cs CategorizedSummary) Total() int {
	return len(cs.SchemaErrors) +
		len(cs.QueryErrors) +
		len(cs.ConfigErrors) +
		len(cs.CodegenErrors) +
		len(cs.Warnings) +
		len(cs.Infos) +
		len(cs.Uncategorized)
}

// ErrorCount returns the total number of errors (excluding warnings and infos).
func (cs CategorizedSummary) ErrorCount() int {
	return len(cs.SchemaErrors) +
		len(cs.QueryErrors) +
		len(cs.ConfigErrors) +
		len(cs.CodegenErrors) +
		len(cs.Uncategorized)
}
