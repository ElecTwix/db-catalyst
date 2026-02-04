// Package diagnostics provides rich diagnostic information for db-catalyst.
package diagnostics

import (
	"fmt"
	"io"
	"strings"

	queryanalyzer "github.com/electwix/db-catalyst/internal/query/analyzer"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
)

// FromQueryAnalyzer converts a query analyzer diagnostic to a rich diagnostic.
func FromQueryAnalyzer(d queryanalyzer.Diagnostic) Diagnostic {
	severity := SeverityWarning
	if d.Severity == queryanalyzer.SeverityError {
		severity = SeverityError
	}

	// Determine error code based on message content
	code := classifyQueryAnalyzerError(d.Message)

	return Diagnostic{
		Severity: severity,
		Message:  d.Message,
		Code:     code,
		Location: Location{
			Path:   d.Path,
			Line:   d.Line,
			Column: d.Column,
		},
		Source: "query-analyzer",
	}
}

// FromQueryParser converts a query parser diagnostic to a rich diagnostic.
func FromQueryParser(d queryparser.Diagnostic) Diagnostic {
	severity := SeverityWarning
	if d.Severity == queryparser.SeverityError {
		severity = SeverityError
	}

	// Determine error code based on message content
	code := classifyQueryParserError(d.Message)

	return Diagnostic{
		Severity: severity,
		Message:  d.Message,
		Code:     code,
		Location: Location{
			Path:   d.Path,
			Line:   d.Line,
			Column: d.Column,
		},
		Source: "query-parser",
	}
}

// FromSchemaParser converts a schema parser diagnostic to a rich diagnostic.
func FromSchemaParser(d schemaparser.Diagnostic) Diagnostic {
	severity := SeverityWarning
	if d.Severity == schemaparser.SeverityError {
		severity = SeverityError
	}

	// Determine error code based on message content
	code := classifySchemaParserError(d.Message)

	return Diagnostic{
		Severity: severity,
		Message:  d.Message,
		Code:     code,
		Location: Location{
			Path:   d.Path,
			Line:   d.Line,
			Column: d.Column,
		},
		Source: "schema-parser",
	}
}

// ToQueryAnalyzer converts a rich diagnostic to a query analyzer diagnostic.
// This is useful for backward compatibility.
func ToQueryAnalyzer(d Diagnostic) queryanalyzer.Diagnostic {
	severity := queryanalyzer.SeverityWarning
	if d.Severity == SeverityError {
		severity = queryanalyzer.SeverityError
	}

	return queryanalyzer.Diagnostic{
		Path:     d.Location.Path,
		Line:     d.Location.Line,
		Column:   d.Location.Column,
		Message:  d.Message,
		Severity: severity,
	}
}

// CollectionFromQueryAnalyzer converts a slice of query analyzer diagnostics to a collection.
func CollectionFromQueryAnalyzer(diags []queryanalyzer.Diagnostic) *Collection {
	c := NewCollection()
	for _, d := range diags {
		c.Add(FromQueryAnalyzer(d))
	}
	return c
}

// CollectionToQueryAnalyzer converts a collection to a slice of query analyzer diagnostics.
func CollectionToQueryAnalyzer(c *Collection) []queryanalyzer.Diagnostic {
	var result []queryanalyzer.Diagnostic
	for _, d := range c.All() {
		result = append(result, ToQueryAnalyzer(d))
	}
	return result
}

// EnrichWithContext adds code context to diagnostics that have file locations.
func EnrichWithContext(c *Collection, extractor *ContextExtractor, contextLines int) {
	all := c.All()
	c.diagnostics = c.diagnostics[:0] // Clear but keep capacity

	for _, d := range all {
		if d.HasLocation() && d.Context == "" {
			ctx, err := extractor.ExtractContext(d.Location.Path, d.Location.Line, d.Location.Column, contextLines)
			if err == nil && !ctx.IsEmpty() {
				d.Context = ctx.Format()
			}
		}
		c.Add(d)
	}
}

// EnrichWithSuggestions adds suggestions to common error patterns.
func EnrichWithSuggestions(c *Collection) {
	all := c.All()
	c.diagnostics = c.diagnostics[:0]

	for _, d := range all {
		d = addSuggestions(d)
		c.Add(d)
	}
}

func addSuggestions(d Diagnostic) Diagnostic {
	// Add suggestions based on error message patterns
	msg := strings.ToLower(d.Message)

	// Unknown column suggestions
	if strings.Contains(msg, "unknown column") {
		if len(d.Suggestions) == 0 {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message:     "Check the column name for typos",
				Replacement: "",
			})
		}
		d.Notes = append(d.Notes, "Column names are case-insensitive in SQLite")
		d.Notes = append(d.Notes, "Use schema files to define tables and columns")
	}

	// Unknown table suggestions
	if strings.Contains(msg, "unknown table") || strings.Contains(msg, "unknown relation") {
		if len(d.Suggestions) == 0 {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message:     "Verify the table name is correct",
				Replacement: "",
			})
		}
		d.Notes = append(d.Notes, "Ensure the table is defined in your schema files")
		d.Notes = append(d.Notes, "Check that the schema file is included in the 'schemas' config")
	}

	// Missing alias suggestions
	if strings.Contains(msg, "ambiguous") {
		if len(d.Suggestions) == 0 {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message:     "Add a table alias to disambiguate",
				Replacement: "table.column",
			})
		}
		d.Notes = append(d.Notes, "When multiple tables have the same column name, qualify with table alias")
	}

	// Aggregate without alias
	if strings.Contains(msg, "requires an alias") {
		if len(d.Suggestions) == 0 {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message:     "Add an alias using AS",
				Replacement: "AS alias_name",
			})
		}
		d.Notes = append(d.Notes, "Aggregates and expressions in SELECT must have an alias for result naming")
	}

	// CTE errors
	if strings.Contains(msg, "cte") {
		if strings.Contains(msg, "missing") {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message: "Ensure CTE has a SELECT body",
			})
		}
		if strings.Contains(msg, "column") {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message: "Check that CTE column list matches SELECT columns",
			})
		}
	}

	// Parameter errors
	if strings.Contains(msg, "parameter") {
		if strings.Contains(msg, "conflicting") {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message: "Use consistent parameter names for the same value",
			})
		}
		if strings.Contains(msg, "duplicate") {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message: "Remove duplicate parameter references",
			})
		}
	}

	// Schema errors
	if strings.Contains(msg, "duplicate") {
		if strings.Contains(msg, "table") {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message: "Remove duplicate table definition or rename",
			})
		}
		if strings.Contains(msg, "column") {
			d.Suggestions = append(d.Suggestions, Suggestion{
				Message: "Remove duplicate column or rename",
			})
		}
	}

	// Foreign key errors
	if strings.Contains(msg, "foreign key") {
		d.Suggestions = append(d.Suggestions, Suggestion{
			Message: "Ensure referenced table and column exist",
		})
		d.Notes = append(d.Notes, "Foreign keys must reference existing tables and columns")
	}

	// Type inference warnings
	if strings.Contains(msg, "defaulting to") && strings.Contains(msg, "interface{}") {
		d.Suggestions = append(d.Suggestions, Suggestion{
			Message: "Add explicit type casting or use a typed column",
		})
		d.Notes = append(d.Notes, "Type inference works best with schema-defined tables")
	}

	return d
}

// classifyQueryAnalyzerError determines the appropriate error code for analyzer messages.
func classifyQueryAnalyzerError(msg string) string {
	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msgLower, "unknown table") || strings.Contains(msgLower, "unknown relation"):
		return ErrQueryUnknownTable
	case strings.Contains(msgLower, "unknown column"):
		return ErrQueryUnknownColumn
	case strings.Contains(msgLower, "ambiguous"):
		return ErrQueryAmbiguousCol
	case strings.Contains(msgLower, "requires an alias"):
		return ErrQueryMissingAlias
	case strings.Contains(msgLower, "cte"):
		if strings.Contains(msgLower, "recursive") {
			return ErrQueryInvalidCTE
		}
		return ErrQueryInvalidCTE
	case strings.Contains(msgLower, "type") && strings.Contains(msgLower, "infer"):
		return WarnTypeInference
	default:
		return ""
	}
}

// classifyQueryParserError determines the appropriate error code for parser messages.
func classifyQueryParserError(msg string) string {
	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msgLower, "unsupported query"):
		return ErrQueryInvalidVerb
	case strings.Contains(msgLower, "parameter"):
		return ErrQueryInvalidParam
	case strings.Contains(msgLower, "cte") || strings.Contains(msgLower, "with clause"):
		return ErrQueryInvalidCTE
	case strings.Contains(msgLower, "alias"):
		return ErrQueryMissingAlias
	case strings.Contains(msgLower, "syntax"):
		return ErrQueryInvalidSyntax
	default:
		return ""
	}
}

// classifySchemaParserError determines the appropriate error code for schema messages.
func classifySchemaParserError(msg string) string {
	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msgLower, "duplicate table"):
		return ErrSchemaDuplicateTable
	case strings.Contains(msgLower, "duplicate view"):
		return ErrSchemaDuplicateView
	case strings.Contains(msgLower, "duplicate column"):
		return ErrSchemaDuplicateCol
	case strings.Contains(msgLower, "unknown table"):
		return ErrSchemaUnknownTable
	case strings.Contains(msgLower, "unknown column"):
		return ErrSchemaUnknownColumn
	case strings.Contains(msgLower, "foreign key"):
		return ErrSchemaInvalidFK
	case strings.Contains(msgLower, "primary key"):
		return ErrSchemaInvalidPK
	case strings.Contains(msgLower, "index"):
		return ErrSchemaInvalidIndex
	case strings.Contains(msgLower, "type"):
		return ErrSchemaInvalidType
	default:
		return ""
	}
}

// CreateConfigError creates a rich diagnostic for configuration errors.
func CreateConfigError(path string, line, column int, message string) Diagnostic {
	code := classifyConfigError(message)
	return Error(message).
		WithCode(code).
		At(path, line, column).
		WithSource("config-loader").
		Build()
}

// CreateSchemaError creates a rich diagnostic for schema parsing errors.
func CreateSchemaError(path string, line, column int, message string) Diagnostic {
	code := classifySchemaParserError(message)
	return Error(message).
		WithCode(code).
		At(path, line, column).
		WithSource("schema-parser").
		Build()
}

// CreateQueryError creates a rich diagnostic for query analysis errors.
func CreateQueryError(path string, line, column int, message string) Diagnostic {
	code := classifyQueryAnalyzerError(message)
	return Error(message).
		WithCode(code).
		At(path, line, column).
		WithSource("query-analyzer").
		Build()
}

// CreateWarning creates a rich diagnostic for warnings.
func CreateWarning(path string, line, column int, message string) Diagnostic {
	return Warning(message).
		At(path, line, column).
		WithSource("db-catalyst").
		Build()
}

// classifyConfigError determines the appropriate error code for config messages.
func classifyConfigError(msg string) string {
	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msgLower, "package"):
		return ErrConfigMissingPackage
	case strings.Contains(msgLower, "out"):
		return ErrConfigMissingOut
	case strings.Contains(msgLower, "path"):
		return ErrConfigInvalidPath
	case strings.Contains(msgLower, "unknown") && strings.Contains(msgLower, "key"):
		return ErrConfigUnknownKey
	case strings.Contains(msgLower, "driver"):
		return ErrConfigInvalidDriver
	case strings.Contains(msgLower, "language"):
		return ErrConfigInvalidLang
	case strings.Contains(msgLower, "database"):
		return ErrConfigInvalidDB
	default:
		return ErrConfigInvalid
	}
}

// EnrichDiagnostic adds context and suggestions to a single diagnostic.
// This is useful when you want to enrich diagnostics one at a time.
func EnrichDiagnostic(d Diagnostic, extractor *ContextExtractor, contextLines int) Diagnostic {
	// Add context if missing
	if d.HasLocation() && d.Context == "" && extractor != nil {
		ctx, err := extractor.ExtractContext(d.Location.Path, d.Location.Line, d.Location.Column, contextLines)
		if err == nil && !ctx.IsEmpty() {
			d.Context = ctx.Format()
		}
	}

	// Add suggestions
	d = addSuggestions(d)

	return d
}

// BatchEnrich enriches all diagnostics in a collection with context and suggestions.
// This is a convenience function that combines EnrichWithContext and EnrichWithSuggestions.
func BatchEnrich(c *Collection, extractor *ContextExtractor, contextLines int) {
	// First add suggestions
	EnrichWithSuggestions(c)

	// Then add context
	if extractor != nil {
		EnrichWithContext(c, extractor, contextLines)
	}
}

// FormatForTerminal formats diagnostics for terminal output with colors.
func FormatForTerminal(c *Collection, verbose bool) string {
	var formatter *Formatter
	if verbose {
		formatter = NewVerboseFormatter()
	} else {
		formatter = NewFormatter()
		formatter.ShowContext = false
		formatter.ShowSuggestions = true
		formatter.ShowNotes = false
		formatter.ShowRelated = false
	}
	formatter.Colorize = true

	return formatter.FormatAll(c)
}

// PrintToWriter prints formatted diagnostics to a writer.
func PrintToWriter(w io.Writer, c *Collection, verbose bool) error {
	output := FormatForTerminal(c, verbose)
	if output != "" {
		_, err := fmt.Fprintln(w, output)
		return err
	}
	return nil
}

// io.Writer interface check
var _ io.Writer = (io.Writer)(nil)
