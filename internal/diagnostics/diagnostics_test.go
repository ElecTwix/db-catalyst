package diagnostics

import (
	"fmt"
	"strings"
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Severity
	}{
		{"info", SeverityInfo},
		{"INFO", SeverityInfo},
		{"warning", SeverityWarning},
		{"warn", SeverityWarning},
		{"WARNING", SeverityWarning},
		{"error", SeverityError},
		{"err", SeverityError},
		{"ERROR", SeverityError},
		{"unknown", SeverityWarning},
		{"", SeverityWarning},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SeverityFromString(tt.input)
			if got != tt.want {
				t.Errorf("SeverityFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDiagnosticHasLocation(t *testing.T) {
	tests := []struct {
		name string
		diag Diagnostic
		want bool
	}{
		{
			name: "has location",
			diag: Diagnostic{Location: Location{Path: "test.go", Line: 1, Column: 1}},
			want: true,
		},
		{
			name: "no path",
			diag: Diagnostic{Location: Location{Line: 1, Column: 1}},
			want: false,
		},
		{
			name: "no line",
			diag: Diagnostic{Location: Location{Path: "test.go", Column: 1}},
			want: false,
		},
		{
			name: "empty location",
			diag: Diagnostic{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diag.HasLocation()
			if got != tt.want {
				t.Errorf("HasLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiagnosticSeverityChecks(t *testing.T) {
	tests := []struct {
		severity  Severity
		isError   bool
		isWarning bool
		isInfo    bool
	}{
		{SeverityInfo, false, false, true},
		{SeverityWarning, false, true, false},
		{SeverityError, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.severity.String(), func(t *testing.T) {
			d := Diagnostic{Severity: tt.severity}
			if d.IsError() != tt.isError {
				t.Errorf("IsError() = %v, want %v", d.IsError(), tt.isError)
			}
			if d.IsWarning() != tt.isWarning {
				t.Errorf("IsWarning() = %v, want %v", d.IsWarning(), tt.isWarning)
			}
			if d.IsInfo() != tt.isInfo {
				t.Errorf("IsInfo() = %v, want %v", d.IsInfo(), tt.isInfo)
			}
		})
	}
}

func TestDiagnosticError(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityError,
		Message:  "test error",
		Code:     "E001",
		Location: Location{Path: "test.go", Line: 10, Column: 5},
	}

	got := d.Error()
	want := "test.go:10:5: [E001] error: test error"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestDiagnosticErrorWithoutCode(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityWarning,
		Message:  "test warning",
		Location: Location{Path: "test.go", Line: 10, Column: 5},
	}

	got := d.Error()
	want := "test.go:10:5: warning: test warning"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestDiagnosticString(t *testing.T) {
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

	got := d.String()

	// Check that all components are present
	checks := []string{
		"test.sql:5:10:",
		"error:",
		"undefined column",
		"[E001]",
		"(query-analyzer)",
		"SELECT unknown_col FROM users",
		"suggestion:",
		"Did you mean 'username'?",
		"note:",
		"Column must be defined in the schema",
		"related:",
		"schema.sql:10:5:",
		"table defined here",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("String() missing expected content %q in:\n%s", check, got)
		}
	}
}

func TestBuilder(t *testing.T) {
	d := Error("test error").
		WithCode("E001").
		At("test.go", 10, 5).
		WithContext("foo.bar()").
		WithSource("parser").
		WithSuggestion("Use foo.baz() instead", "foo.baz()").
		WithNote("This is deprecated").
		WithRelated("other.go", 20, 3, "original definition").
		Build()

	if d.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", d.Severity, SeverityError)
	}
	if d.Message != "test error" {
		t.Errorf("Message = %q, want %q", d.Message, "test error")
	}
	if d.Code != "E001" {
		t.Errorf("Code = %q, want %q", d.Code, "E001")
	}
	if d.Location.Path != "test.go" || d.Location.Line != 10 || d.Location.Column != 5 {
		t.Errorf("Location = %+v, want test.go:10:5", d.Location)
	}
	if d.Context != "foo.bar()" {
		t.Errorf("Context = %q, want %q", d.Context, "foo.bar()")
	}
	if d.Source != "parser" {
		t.Errorf("Source = %q, want %q", d.Source, "parser")
	}
	if len(d.Suggestions) != 1 {
		t.Errorf("Suggestions length = %d, want 1", len(d.Suggestions))
	}
	if len(d.Notes) != 1 {
		t.Errorf("Notes length = %d, want 1", len(d.Notes))
	}
	if len(d.Related) != 1 {
		t.Errorf("Related length = %d, want 1", len(d.Related))
	}
}

func TestBuilderConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name     string
		builder  *Builder
		expected Severity
	}{
		{"Error", Error("test"), SeverityError},
		{"Warning", Warning("test"), SeverityWarning},
		{"Info", Info("test"), SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.builder.Build()
			if d.Severity != tt.expected {
				t.Errorf("Severity = %v, want %v", d.Severity, tt.expected)
			}
		})
	}
}

func TestCollection(t *testing.T) {
	c := NewCollection()

	// Add diagnostics
	c.Add(Error("error 1").At("a.go", 1, 1).Build())
	c.Add(Warning("warning 1").At("b.go", 2, 2).Build())
	c.Add(Error("error 2").At("c.go", 3, 3).Build())
	c.Add(Info("info 1").Build())

	// Test HasErrors
	if !c.HasErrors() {
		t.Error("HasErrors() = false, want true")
	}

	// Test counts
	if c.Len() != 4 {
		t.Errorf("Len() = %d, want 4", c.Len())
	}

	// Test filtering
	errs := c.Errors()
	if len(errs) != 2 {
		t.Errorf("Errors() length = %d, want 2", len(errs))
	}

	warns := c.Warnings()
	if len(warns) != 1 {
		t.Errorf("Warnings() length = %d, want 1", len(warns))
	}

	// Test summary
	summary := c.Summary()
	if summary.Total != 4 {
		t.Errorf("Summary.Total = %d, want 4", summary.Total)
	}
	if summary.Errors != 2 {
		t.Errorf("Summary.Errors = %d, want 2", summary.Errors)
	}
	if summary.Warnings != 1 {
		t.Errorf("Summary.Warnings = %d, want 1", summary.Warnings)
	}
	if summary.Infos != 1 {
		t.Errorf("Summary.Infos = %d, want 1", summary.Infos)
	}
}

func TestCollectionNoErrors(t *testing.T) {
	c := NewCollection()
	c.Add(Warning("warning").Build())
	c.Add(Info("info").Build())

	if c.HasErrors() {
		t.Error("HasErrors() = true, want false")
	}

	errs := c.Errors()
	if len(errs) != 0 {
		t.Errorf("Errors() length = %d, want 0", len(errs))
	}
}

func TestCollectionBySeverity(t *testing.T) {
	c := NewCollection()
	c.Add(Error("e1").Build())
	c.Add(Error("e2").Build())
	c.Add(Warning("w1").Build())

	errs := c.BySeverity(SeverityError)
	if len(errs) != 2 {
		t.Errorf("BySeverity(Error) length = %d, want 2", len(errs))
	}

	warns := c.BySeverity(SeverityWarning)
	if len(warns) != 1 {
		t.Errorf("BySeverity(Warning) length = %d, want 1", len(warns))
	}
}

func TestCollectionBySource(t *testing.T) {
	c := NewCollection()
	c.Add(Error("e1").WithSource("parser").Build())
	c.Add(Error("e2").WithSource("analyzer").Build())
	c.Add(Warning("w1").WithSource("parser").Build())

	parserDiags := c.BySource("parser")
	if len(parserDiags) != 2 {
		t.Errorf("BySource(parser) length = %d, want 2", len(parserDiags))
	}
}

func TestCollectionSortByLocation(t *testing.T) {
	c := NewCollection()
	c.Add(Error("z").At("z.go", 1, 1).Build())
	c.Add(Error("a").At("a.go", 1, 1).Build())
	c.Add(Error("b2").At("b.go", 2, 1).Build())
	c.Add(Error("b1").At("b.go", 1, 1).Build())

	c.SortByLocation()

	all := c.All()
	if len(all) != 4 {
		t.Fatalf("expected 4 diagnostics, got %d", len(all))
	}

	// Check order: a.go, b.go:1, b.go:2, z.go
	if all[0].Location.Path != "a.go" {
		t.Errorf("first = %s, want a.go", all[0].Location.Path)
	}
	if all[1].Location.Path != "b.go" || all[1].Location.Line != 1 {
		t.Errorf("second = %s:%d, want b.go:1", all[1].Location.Path, all[1].Location.Line)
	}
	if all[2].Location.Path != "b.go" || all[2].Location.Line != 2 {
		t.Errorf("third = %s:%d, want b.go:2", all[2].Location.Path, all[2].Location.Line)
	}
	if all[3].Location.Path != "z.go" {
		t.Errorf("fourth = %s, want z.go", all[3].Location.Path)
	}
}

func TestCollectionAddAll(t *testing.T) {
	c1 := NewCollection()
	c1.Add(Error("e1").Build())
	c1.Add(Warning("w1").Build())

	c2 := NewCollection()
	c2.Add(Error("e2").Build())
	c2.AddAll(c1)

	if c2.Len() != 3 {
		t.Errorf("Len() = %d, want 3", c2.Len())
	}
}

func TestCollectionFilter(t *testing.T) {
	c := NewCollection()
	c.Add(Error("e1").At("test.go", 1, 1).Build())
	c.Add(Error("e2").At("other.go", 1, 1).Build())
	c.Add(Warning("w1").At("test.go", 2, 1).Build())

	// Filter for errors in test.go
	filtered := c.Filter(func(d Diagnostic) bool {
		return d.IsError() && d.Location.Path == "test.go"
	})

	if len(filtered) != 1 {
		t.Errorf("Filter() length = %d, want 1", len(filtered))
	}
	if filtered[0].Message != "e1" {
		t.Errorf("Filter()[0].Message = %q, want %q", filtered[0].Message, "e1")
	}
}

func TestCompareLocation(t *testing.T) {
	tests := []struct {
		a    Location
		b    Location
		want int
	}{
		{Location{"a.go", 1, 1, 0}, Location{"b.go", 1, 1, 0}, -1},
		{Location{"b.go", 1, 1, 0}, Location{"a.go", 1, 1, 0}, 1},
		{Location{"a.go", 1, 1, 0}, Location{"a.go", 2, 1, 0}, -1},
		{Location{"a.go", 2, 1, 0}, Location{"a.go", 1, 1, 0}, 1},
		{Location{"a.go", 1, 1, 0}, Location{"a.go", 1, 2, 0}, -1},
		{Location{"a.go", 1, 2, 0}, Location{"a.go", 1, 1, 0}, 1},
		{Location{"a.go", 1, 1, 0}, Location{"a.go", 1, 1, 0}, 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%d:%d vs %s:%d:%d", tt.a.Path, tt.a.Line, tt.a.Column, tt.b.Path, tt.b.Line, tt.b.Column), func(t *testing.T) {
			got := compareLocation(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareLocation() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDiagnosticStringWithoutLocation(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityWarning,
		Message:  "general warning",
	}

	got := d.String()
	want := "warning: general warning"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestDiagnosticHasSpan(t *testing.T) {
	tests := []struct {
		name string
		diag Diagnostic
		want bool
	}{
		{
			name: "has span",
			diag: Diagnostic{
				Span: &Span{
					Start: Location{Path: "test.go", Line: 1, Column: 1},
					End:   Location{Path: "test.go", Line: 2, Column: 10},
				},
			},
			want: true,
		},
		{
			name: "nil span",
			diag: Diagnostic{},
			want: false,
		},
		{
			name: "empty span",
			diag: Diagnostic{
				Span: &Span{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diag.HasSpan()
			if got != tt.want {
				t.Errorf("HasSpan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectionByCode(t *testing.T) {
	c := NewCollection()
	c.Add(Error("e1").WithCode(ErrSchemaParseError).Build())
	c.Add(Error("e2").WithCode(ErrQueryParseError).Build())
	c.Add(Error("e3").WithCode(ErrSchemaParseError).Build())

	schemaErrors := c.ByCode(ErrSchemaParseError)
	if len(schemaErrors) != 2 {
		t.Errorf("ByCode(ErrSchemaParseError) length = %d, want 2", len(schemaErrors))
	}

	queryErrors := c.ByCode(ErrQueryParseError)
	if len(queryErrors) != 1 {
		t.Errorf("ByCode(ErrQueryParseError) length = %d, want 1", len(queryErrors))
	}
}

func TestCodeDescription(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{ErrSchemaParseError, "Schema parsing failed"},
		{ErrQueryUnknownTable, "Reference to unknown table"},
		{ErrConfigInvalid, "Invalid configuration"},
		{"UNKNOWN_CODE", "Unknown error code"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := CodeDescription(tt.code)
			if got != tt.want {
				t.Errorf("CodeDescription(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestCollectionCategorize(t *testing.T) {
	c := NewCollection()
	c.Add(Error("schema error").WithCode(ErrSchemaParseError).Build())
	c.Add(Error("query error").WithCode(ErrQueryParseError).Build())
	c.Add(Error("config error").WithCode(ErrConfigInvalid).Build())
	c.Add(Error("codegen error").WithCode(ErrCodeGenFailed).Build())
	c.Add(Warning("warning").WithCode(WarnTypeInference).Build())
	c.Add(Info("info").WithCode("I001").Build())
	c.Add(Error("uncategorized").Build())

	cat := c.Categorize()

	if len(cat.SchemaErrors) != 1 {
		t.Errorf("SchemaErrors length = %d, want 1", len(cat.SchemaErrors))
	}
	if len(cat.QueryErrors) != 1 {
		t.Errorf("QueryErrors length = %d, want 1", len(cat.QueryErrors))
	}
	if len(cat.ConfigErrors) != 1 {
		t.Errorf("ConfigErrors length = %d, want 1", len(cat.ConfigErrors))
	}
	if len(cat.CodegenErrors) != 1 {
		t.Errorf("CodegenErrors length = %d, want 1", len(cat.CodegenErrors))
	}
	if len(cat.Warnings) != 1 {
		t.Errorf("Warnings length = %d, want 1", len(cat.Warnings))
	}
	if len(cat.Infos) != 1 {
		t.Errorf("Infos length = %d, want 1", len(cat.Infos))
	}
	if len(cat.Uncategorized) != 1 {
		t.Errorf("Uncategorized length = %d, want 1", len(cat.Uncategorized))
	}
}

func TestCategorizedSummaryHasDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		cat  CategorizedSummary
		want bool
	}{
		{
			name: "has schema errors",
			cat:  CategorizedSummary{SchemaErrors: []Diagnostic{{}}},
			want: true,
		},
		{
			name: "has query errors",
			cat:  CategorizedSummary{QueryErrors: []Diagnostic{{}}},
			want: true,
		},
		{
			name: "empty",
			cat:  CategorizedSummary{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cat.HasDiagnostics()
			if got != tt.want {
				t.Errorf("HasDiagnostics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategorizedSummaryTotal(t *testing.T) {
	cat := CategorizedSummary{
		SchemaErrors:  []Diagnostic{{}, {}},
		QueryErrors:   []Diagnostic{{}},
		Warnings:      []Diagnostic{{}},
		Uncategorized: []Diagnostic{{}},
	}

	if got := cat.Total(); got != 5 {
		t.Errorf("Total() = %d, want 5", got)
	}
}

func TestCategorizedSummaryErrorCount(t *testing.T) {
	cat := CategorizedSummary{
		SchemaErrors:  []Diagnostic{{}, {}},
		QueryErrors:   []Diagnostic{{}},
		Warnings:      []Diagnostic{{}},
		Uncategorized: []Diagnostic{{}},
	}

	if got := cat.ErrorCount(); got != 4 {
		t.Errorf("ErrorCount() = %d, want 4 (excluding warnings)", got)
	}
}
