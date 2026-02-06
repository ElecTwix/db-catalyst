package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	queryanalyzer "github.com/electwix/db-catalyst/internal/query/analyzer"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
)

func TestFromQueryAnalyzer(t *testing.T) {
	tests := []struct {
		name     string
		input    queryanalyzer.Diagnostic
		expected Diagnostic
	}{
		{
			name: "error severity",
			input: queryanalyzer.Diagnostic{
				Path:     "test.sql",
				Line:     10,
				Column:   5,
				Message:  "test error",
				Severity: queryanalyzer.SeverityError,
			},
			expected: Diagnostic{
				Severity: SeverityError,
				Message:  "test error",
				Location: Location{Path: "test.sql", Line: 10, Column: 5},
				Source:   "query-analyzer",
			},
		},
		{
			name: "warning severity",
			input: queryanalyzer.Diagnostic{
				Path:     "test.sql",
				Line:     10,
				Column:   5,
				Message:  "test warning",
				Severity: queryanalyzer.SeverityWarning,
			},
			expected: Diagnostic{
				Severity: SeverityWarning,
				Message:  "test warning",
				Location: Location{Path: "test.sql", Line: 10, Column: 5},
				Source:   "query-analyzer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromQueryAnalyzer(tt.input)
			if got.Severity != tt.expected.Severity {
				t.Errorf("Severity = %v, want %v", got.Severity, tt.expected.Severity)
			}
			if got.Message != tt.expected.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.expected.Message)
			}
			if got.Location != tt.expected.Location {
				t.Errorf("Location = %+v, want %+v", got.Location, tt.expected.Location)
			}
			if got.Source != tt.expected.Source {
				t.Errorf("Source = %q, want %q", got.Source, tt.expected.Source)
			}
		})
	}
}

func TestFromQueryParser(t *testing.T) {
	input := queryparser.Diagnostic{
		Path:     "queries.sql",
		Line:     5,
		Column:   10,
		Message:  "parse error",
		Severity: queryparser.SeverityError,
	}

	got := FromQueryParser(input)

	if got.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", got.Severity, SeverityError)
	}
	if got.Source != "query-parser" {
		t.Errorf("Source = %q, want %q", got.Source, "query-parser")
	}
}

func TestFromSchemaParser(t *testing.T) {
	input := schemaparser.Diagnostic{
		Path:     "schema.sql",
		Line:     3,
		Column:   15,
		Message:  "syntax error",
		Severity: schemaparser.SeverityError,
	}

	got := FromSchemaParser(input)

	if got.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", got.Severity, SeverityError)
	}
	if got.Source != "schema-parser" {
		t.Errorf("Source = %q, want %q", got.Source, "schema-parser")
	}
}

func TestToQueryAnalyzer(t *testing.T) {
	input := Diagnostic{
		Severity: SeverityError,
		Message:  "test error",
		Location: Location{Path: "test.sql", Line: 10, Column: 5},
	}

	got := ToQueryAnalyzer(input)

	if got.Severity != queryanalyzer.SeverityError {
		t.Errorf("Severity = %v, want %v", got.Severity, queryanalyzer.SeverityError)
	}
	if got.Path != "test.sql" {
		t.Errorf("Path = %q, want %q", got.Path, "test.sql")
	}
	if got.Line != 10 {
		t.Errorf("Line = %d, want 10", got.Line)
	}
	if got.Column != 5 {
		t.Errorf("Column = %d, want 5", got.Column)
	}
	if got.Message != "test error" {
		t.Errorf("Message = %q, want %q", got.Message, "test error")
	}
}

func TestCollectionFromQueryAnalyzer(t *testing.T) {
	inputs := []queryanalyzer.Diagnostic{
		{Path: "a.sql", Line: 1, Message: "error 1", Severity: queryanalyzer.SeverityError},
		{Path: "b.sql", Line: 2, Message: "warning 1", Severity: queryanalyzer.SeverityWarning},
	}

	c := CollectionFromQueryAnalyzer(inputs)

	if c.Len() != 2 {
		t.Errorf("Len() = %d, want 2", c.Len())
	}

	errs := c.Errors()
	if len(errs) != 1 {
		t.Errorf("Errors() = %d, want 1", len(errs))
	}
}

func TestCollectionToQueryAnalyzer(t *testing.T) {
	c := NewCollection()
	c.Add(Error("error 1").At("a.sql", 1, 1).Build())
	c.Add(Warning("warning 1").At("b.sql", 2, 2).Build())

	result := CollectionToQueryAnalyzer(c)

	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}

	if result[0].Severity != queryanalyzer.SeverityError {
		t.Errorf("first severity = %v, want error", result[0].Severity)
	}
}

func TestEnrichWithContext(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	content := "SELECT col1\nFROM users\nWHERE id = ?"
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	c := NewCollection()
	c.Add(Error("test error").At(tmpFile, 2, 5).Build())

	extractor := NewContextExtractor()
	EnrichWithContext(c, extractor, 1)

	all := c.All()
	if len(all) != 1 {
		t.Fatal("Expected 1 diagnostic")
	}

	if all[0].Context == "" {
		t.Error("Expected context to be enriched")
	}

	if !strings.Contains(all[0].Context, "FROM users") {
		t.Errorf("Context should contain 'FROM users', got: %s", all[0].Context)
	}
}

func TestEnrichWithContextNoLocation(t *testing.T) {
	c := NewCollection()
	c.Add(Error("test error").Build()) // No location

	extractor := NewContextExtractor()
	EnrichWithContext(c, extractor, 1)

	all := c.All()
	if all[0].Context != "" {
		t.Error("Expected no context for diagnostic without location")
	}
}

func TestEnrichWithContextExistingContext(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(tmpFile, []byte("SELECT 1"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	c := NewCollection()
	c.Add(Error("test error").At(tmpFile, 1, 1).WithContext("existing context").Build())

	extractor := NewContextExtractor()
	EnrichWithContext(c, extractor, 1)

	all := c.All()
	if all[0].Context != "existing context" {
		t.Error("Should not overwrite existing context")
	}
}

func TestAddSuggestions(t *testing.T) {
	tests := []struct {
		name             string
		message          string
		expectSuggestion bool
		expectNote       bool
	}{
		{
			name:             "unknown column",
			message:          "unknown column 'foo'",
			expectSuggestion: true,
			expectNote:       true,
		},
		{
			name:             "unknown table",
			message:          "unknown table 'users'",
			expectSuggestion: true,
			expectNote:       true,
		},
		{
			name:             "ambiguous column",
			message:          "ambiguous column 'id'",
			expectSuggestion: true,
			expectNote:       true, // Now adds a note about qualifying with table alias
		},
		{
			name:             "requires alias",
			message:          "aggregate requires an alias",
			expectSuggestion: true,
			expectNote:       true, // Now adds a note about aggregates needing aliases
		},
		{
			name:             "no match",
			message:          "some other error",
			expectSuggestion: false,
			expectNote:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Error(tt.message).Build()
			enriched := addSuggestions(d)

			hasSuggestion := len(enriched.Suggestions) > 0
			if hasSuggestion != tt.expectSuggestion {
				t.Errorf("hasSuggestion = %v, want %v", hasSuggestion, tt.expectSuggestion)
			}

			hasNote := len(enriched.Notes) > 0
			if hasNote != tt.expectNote {
				t.Errorf("hasNote = %v, want %v", hasNote, tt.expectNote)
			}
		})
	}
}

func TestEnrichWithSuggestions(t *testing.T) {
	c := NewCollection()
	c.Add(Error("unknown column 'foo'").Build())
	c.Add(Error("some other error").Build())

	EnrichWithSuggestions(c)

	all := c.All()
	if len(all) != 2 {
		t.Fatal("Expected 2 diagnostics")
	}

	// First diagnostic should have suggestions
	if len(all[0].Suggestions) == 0 {
		t.Error("Expected first diagnostic to have suggestions")
	}

	// Second diagnostic should not have suggestions
	if len(all[1].Suggestions) > 0 {
		t.Error("Expected second diagnostic to have no suggestions")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"", "foo", false},
		{"foo", "", true},
		{"foo", "foo", true},
		{"hello world", "hello world", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			if got := strings.Contains(tt.s, tt.substr); got != tt.want {
				t.Errorf("strings.Contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
