package diagnostics

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatterDefaultSettings(t *testing.T) {
	f := NewFormatter()

	if !f.ShowContext {
		t.Error("ShowContext should be true by default")
	}
	if !f.ShowSuggestions {
		t.Error("ShowSuggestions should be true by default")
	}
	if !f.ShowNotes {
		t.Error("ShowNotes should be true by default")
	}
	if !f.ShowRelated {
		t.Error("ShowRelated should be true by default")
	}
	if f.ShowSource {
		t.Error("ShowSource should be false by default")
	}
	if !f.ShowCode {
		t.Error("ShowCode should be true by default")
	}
	if f.Colorize {
		t.Error("Colorize should be false by default")
	}
	if f.ContextLines != 2 {
		t.Errorf("ContextLines = %d, want 2", f.ContextLines)
	}
}

func TestFormatterFormat(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false // Disable colors for predictable output

	d := Diagnostic{
		Severity: SeverityError,
		Message:  "undefined column",
		Code:     "E001",
		Location: Location{Path: "test.sql", Line: 5, Column: 10},
		Context:  "SELECT unknown_col FROM users",
		Source:   "query-analyzer",
		Suggestions: []Suggestion{
			{Message: "Did you mean 'username'?", Replacement: "username"},
		},
		Notes: []string{"Column must be defined in the schema"},
		Related: []RelatedInfo{
			{Location: Location{Path: "schema.sql", Line: 10, Column: 5}, Message: "table defined here"},
		},
	}

	got := f.Format(d)

	// Check that key components are present
	checks := []string{
		"test.sql:5:10:",
		"error:",
		"undefined column",
		"[E001]",
		"-->",
		"SELECT unknown_col FROM users",
		"help:",
		"Did you mean 'username'?",
		"=>",
		"username",
		"note:",
		"Column must be defined in the schema",
		"related:",
		"schema.sql:10:5:",
		"table defined here",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("Format() missing expected content %q in:\n%s", check, got)
		}
	}
}

func TestFormatterFormatWithoutOptions(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false
	f.ShowContext = false
	f.ShowSuggestions = false
	f.ShowNotes = false
	f.ShowRelated = false
	f.ShowCode = false

	d := Diagnostic{
		Severity: SeverityError,
		Message:  "test error",
		Code:     "E001",
		Location: Location{Path: "test.sql", Line: 5, Column: 10},
		Context:  "SELECT * FROM users",
		Suggestions: []Suggestion{
			{Message: "fix it"},
		},
		Notes: []string{"note"},
		Related: []RelatedInfo{
			{Location: Location{Path: "other.sql", Line: 1, Column: 1}, Message: "related"},
		},
	}

	got := f.Format(d)

	// Should only contain basic info
	if !strings.Contains(got, "test.sql:5:10:") {
		t.Error("Should contain location")
	}
	if !strings.Contains(got, "error:") {
		t.Error("Should contain severity")
	}
	if !strings.Contains(got, "test error") {
		t.Error("Should contain message")
	}

	// Should NOT contain disabled elements
	if strings.Contains(got, "[E001]") {
		t.Error("Should not contain code when ShowCode is false")
	}
	if strings.Contains(got, "SELECT * FROM users") {
		t.Error("Should not contain context when ShowContext is false")
	}
	if strings.Contains(got, "help:") {
		t.Error("Should not contain suggestions when ShowSuggestions is false")
	}
	if strings.Contains(got, "note:") {
		t.Error("Should not contain notes when ShowNotes is false")
	}
	if strings.Contains(got, "related:") {
		t.Error("Should not contain related when ShowRelated is false")
	}
}

func TestFormatterFormatWithoutLocation(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false

	d := Diagnostic{
		Severity: SeverityWarning,
		Message:  "general warning",
	}

	got := f.Format(d)
	want := "warning: general warning\n"

	if got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestFormatterFormatAll(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false
	f.ShowContext = false
	f.ShowSuggestions = false
	f.ShowNotes = false
	f.ShowRelated = false
	f.ShowCode = false

	c := NewCollection()
	c.Add(Error("error 1").At("a.go", 1, 1).Build())
	c.Add(Warning("warning 1").At("b.go", 2, 2).Build())

	got := f.FormatAll(c)

	if !strings.Contains(got, "a.go:1:1:") {
		t.Error("Should contain first diagnostic location")
	}
	if !strings.Contains(got, "b.go:2:2:") {
		t.Error("Should contain second diagnostic location")
	}
}

func TestFormatterPrintSummary(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false

	tests := []struct {
		name     string
		diags    []Diagnostic
		contains []string
	}{
		{
			name:     "empty",
			diags:    []Diagnostic{},
			contains: []string{},
		},
		{
			name: "only errors",
			diags: []Diagnostic{
				Error("e1").Build(),
				Error("e2").Build(),
			},
			contains: []string{"2 error(s)"},
		},
		{
			name: "mixed",
			diags: []Diagnostic{
				Error("e1").Build(),
				Warning("w1").Build(),
				Info("i1").Build(),
			},
			contains: []string{"1 error(s)", "1 warning(s)", "1 info(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCollection()
			for _, d := range tt.diags {
				c.Add(d)
			}

			var b strings.Builder
			f.PrintSummary(&b, c)
			got := b.String()

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("PrintSummary() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

func TestFormatterSeverityColor(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityError, colorRed},
		{SeverityWarning, colorYellow},
		{SeverityInfo, colorBlue},
		{Severity(99), colorReset},
	}

	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			got := f.severityColor(tt.severity)
			if got != tt.want {
				t.Errorf("severityColor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatterColorize(t *testing.T) {
	tests := []struct {
		colorize bool
		input    string
		color    string
		want     string
	}{
		{false, "test", colorRed, "test"},
		{true, "test", colorRed, colorRed + "test" + colorReset},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("colorize=%v", tt.colorize), func(t *testing.T) {
			f := &Formatter{Colorize: tt.colorize}
			got := f.colorize(tt.input, tt.color)
			if got != tt.want {
				t.Errorf("colorize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSimpleFormatter(t *testing.T) {
	f := &SimpleFormatter{}

	tests := []struct {
		name     string
		diag     Diagnostic
		expected string
	}{
		{
			name:     "with location",
			diag:     Error("test error").At("file.go", 10, 5).WithCode("E001").Build(),
			expected: "file.go:10:5: error: test error [E001]",
		},
		{
			name:     "without location",
			diag:     Warning("test warning").Build(),
			expected: "warning: test warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.Format(tt.diag)
			if got != tt.expected {
				t.Errorf("Format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	f := &JSONFormatter{}

	tests := []struct {
		name     string
		diag     Diagnostic
		contains []string
	}{
		{
			name: "basic",
			diag: Error("test error").Build(),
			contains: []string{
				`"severity":"error"`,
				`"message":"test error"`,
			},
		},
		{
			name: "with location",
			diag: Error("test error").At("file.go", 10, 5).Build(),
			contains: []string{
				`"severity":"error"`,
				`"message":"test error"`,
				`"location":{`,
				`"path":"file.go"`,
				`"line":10`,
				`"column":5`,
			},
		},
		{
			name: "with code and source",
			diag: Error("test error").At("file.go", 10, 5).WithCode("E001").WithSource("parser").Build(),
			contains: []string{
				`"code":"E001"`,
				`"source":"parser"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.Format(tt.diag)

			// Verify it's valid JSON-like structure (starts with { and ends with })
			if !strings.HasPrefix(got, "{") || !strings.HasSuffix(got, "}") {
				t.Errorf("Format() should return JSON-like object, got: %s", got)
			}

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Format() missing %q in:\n%s", want, got)
				}
			}
		})
	}
}

func TestFormatterFormatContext(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false

	d := Diagnostic{
		Severity: SeverityError,
		Message:  "test error",
		Context:  "SELECT col1,\n  col2,\n  col3\nFROM users",
	}

	got := f.Format(d)

	// Check that all lines of context are present
	lines := []string{
		"SELECT col1,",
		"col2,",
		"col3",
		"FROM users",
	}

	for _, line := range lines {
		if !strings.Contains(got, line) {
			t.Errorf("Format() missing context line %q in:\n%s", line, got)
		}
	}
}

func TestNewVerboseFormatter(t *testing.T) {
	f := NewVerboseFormatter()

	if !f.ShowContext {
		t.Error("ShowContext should be true for verbose")
	}
	if !f.ShowSuggestions {
		t.Error("ShowSuggestions should be true for verbose")
	}
	if !f.ShowNotes {
		t.Error("ShowNotes should be true for verbose")
	}
	if !f.ShowRelated {
		t.Error("ShowRelated should be true for verbose")
	}
	if !f.ShowSource {
		t.Error("ShowSource should be true for verbose")
	}
	if !f.ShowCode {
		t.Error("ShowCode should be true for verbose")
	}
	if !f.ShowCodeDescription {
		t.Error("ShowCodeDescription should be true for verbose")
	}
	if !f.Colorize {
		t.Error("Colorize should be true for verbose")
	}
	if f.ContextLines != 3 {
		t.Errorf("ContextLines = %d, want 3", f.ContextLines)
	}
}

func TestNewSimpleFormatter(t *testing.T) {
	f := NewSimpleFormatter()

	if f.ShowContext {
		t.Error("ShowContext should be false for simple")
	}
	if f.ShowSuggestions {
		t.Error("ShowSuggestions should be false for simple")
	}
	if f.ShowNotes {
		t.Error("ShowNotes should be false for simple")
	}
	if f.ShowRelated {
		t.Error("ShowRelated should be false for simple")
	}
	if f.ShowSource {
		t.Error("ShowSource should be false for simple")
	}
	if f.ShowCode {
		t.Error("ShowCode should be false for simple")
	}
	if f.Colorize {
		t.Error("Colorize should be false for simple")
	}
}

func TestFormatterPrintCategorizedSummary(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false

	c := NewCollection()
	c.Add(Error("schema error").WithCode(ErrSchemaParseError).Build())
	c.Add(Error("query error").WithCode(ErrQueryParseError).Build())
	c.Add(Warning("warning").WithCode(WarnTypeInference).Build())

	var b strings.Builder
	f.PrintCategorizedSummary(&b, c)
	got := b.String()

	// Check that all categories are mentioned
	checks := []string{
		"Diagnostic Summary:",
		"Schema Errors:",
		"Query Errors:",
		"Warnings:",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("PrintCategorizedSummary() missing %q in:\n%s", check, got)
		}
	}
}

func TestFormatterFormatWithCodeDescription(t *testing.T) {
	f := NewFormatter()
	f.Colorize = false
	f.ShowCode = true
	f.ShowCodeDescription = true

	d := Diagnostic{
		Severity: SeverityError,
		Message:  "test error",
		Code:     ErrSchemaParseError,
		Location: Location{Path: "test.sql", Line: 1, Column: 1},
	}

	got := f.Format(d)

	// Should contain code and description
	if !strings.Contains(got, "[E101]") {
		t.Error("Should contain error code")
	}
	if !strings.Contains(got, "Schema parsing failed") {
		t.Error("Should contain code description")
	}
}

func TestJSONFormatterFormatCollection(t *testing.T) {
	f := &JSONFormatter{Indent: false}

	c := NewCollection()
	c.Add(Error("error 1").At("a.go", 1, 1).Build())
	c.Add(Warning("warning 1").At("b.go", 2, 2).Build())

	got := f.FormatCollection(c)

	// Should be a JSON array
	if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
		t.Errorf("FormatCollection() should return JSON array, got: %s", got)
	}

	// Should contain both diagnostics
	if !strings.Contains(got, "error 1") {
		t.Error("Should contain first diagnostic message")
	}
	if !strings.Contains(got, "warning 1") {
		t.Error("Should contain second diagnostic message")
	}
}
