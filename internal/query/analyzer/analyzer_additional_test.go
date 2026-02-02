package analyzer

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestAnalyzer_SQLiteTypeToGo(t *testing.T) {
	a := New(&model.Catalog{})

	tests := []struct {
		sqlite string
		want   string
	}{
		{"INTEGER", "int64"},
		{"TEXT", "string"},
		{"REAL", "float64"},
		{"BLOB", "[]byte"},
		{"NUMERIC", "string"},
		{"UNKNOWN", "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.sqlite, func(t *testing.T) {
			got := a.SQLiteTypeToGo(tt.sqlite)
			if got != tt.want {
				t.Errorf("SQLiteTypeToGo(%q) = %q, want %q", tt.sqlite, got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	catalog := model.NewCatalog()
	a := New(catalog)

	//nolint:staticcheck // Test validates nil check before dereference
	if a == nil {
		t.Error("New() returned nil")
	}

	//nolint:staticcheck // Test validates nil check before dereference
	if a.Catalog != catalog {
		t.Error("New() did not set catalog correctly")
	}
}

func TestNewWithCustomTypes(t *testing.T) {
	catalog := model.NewCatalog()
	customTypes := map[string]config.CustomTypeMapping{
		"custom_id": {GoType: "CustomID", SQLiteType: "INTEGER"},
	}

	a := NewWithCustomTypes(catalog, customTypes)

	//nolint:staticcheck // Test validates nil check before dereference
	if a == nil {
		t.Error("NewWithCustomTypes() returned nil")
	}

	//nolint:staticcheck // Test validates nil check before dereference
	if a.Catalog != catalog {
		t.Error("NewWithCustomTypes() did not set catalog correctly")
	}

	//nolint:staticcheck // Test validates nil check before dereference
	if a.CustomTypes == nil {
		t.Error("NewWithCustomTypes() did not set custom types")
	}
}

func TestSQLiteTypeToGo_Static(t *testing.T) {
	tests := []struct {
		sqlite string
		want   string
	}{
		{"INTEGER", "int64"},
		{"TEXT", "string"},
		{"REAL", "float64"},
		{"BLOB", "[]byte"},
		{"NUMERIC", "string"},
		{"", "interface{}"},
		{"UNKNOWN_TYPE", "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.sqlite, func(t *testing.T) {
			got := SQLiteTypeToGo(tt.sqlite)
			if got != tt.want {
				t.Errorf("SQLiteTypeToGo(%q) = %q, want %q", tt.sqlite, got, tt.want)
			}
		})
	}
}

func TestSeverity(t *testing.T) {
	tests := []struct {
		severity Severity
		name     string
	}{
		{SeverityWarning, "Warning"},
		{SeverityError, "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Diagnostic{Severity: tt.severity}
			if d.Severity != tt.severity {
				t.Errorf("Severity = %v, want %v", d.Severity, tt.severity)
			}
		})
	}
}

func TestResultColumn(t *testing.T) {
	rc := ResultColumn{
		Name:     "id",
		Table:    "users",
		GoType:   "int64",
		Nullable: false,
	}

	if rc.Name != "id" {
		t.Errorf("Name = %q, want id", rc.Name)
	}
	if rc.Table != "users" {
		t.Errorf("Table = %q, want users", rc.Table)
	}
	if rc.GoType != "int64" {
		t.Errorf("GoType = %q, want int64", rc.GoType)
	}
	if rc.Nullable {
		t.Error("Nullable should be false")
	}
}

func TestResultParam(t *testing.T) {
	rp := ResultParam{
		Name:          "userId",
		GoType:        "int64",
		Nullable:      false,
		IsVariadic:    true,
		VariadicCount: 3,
	}

	if rp.Name != "userId" {
		t.Errorf("Name = %q, want userId", rp.Name)
	}
	if rp.GoType != "int64" {
		t.Errorf("GoType = %q, want int64", rp.GoType)
	}
	if rp.Nullable {
		t.Error("Nullable should be false")
	}
	if !rp.IsVariadic {
		t.Error("IsVariadic should be true")
	}
	if rp.VariadicCount != 3 {
		t.Errorf("VariadicCount = %d, want 3", rp.VariadicCount)
	}
}

func TestDiagnostic(t *testing.T) {
	d := Diagnostic{
		Path:     "test.sql",
		Line:     10,
		Column:   5,
		Message:  "test error",
		Severity: SeverityError,
	}

	if d.Path != "test.sql" {
		t.Errorf("Path = %q, want test.sql", d.Path)
	}
	if d.Line != 10 {
		t.Errorf("Line = %d, want 10", d.Line)
	}
	if d.Column != 5 {
		t.Errorf("Column = %d, want 5", d.Column)
	}
	if d.Message != "test error" {
		t.Errorf("Message = %q, want 'test error'", d.Message)
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity = %v, want SeverityError", d.Severity)
	}
}

func TestAnalyzer_SQLiteTypeToGo_WithCustomTypes(t *testing.T) {
	// Custom type key must be the custom type name (uppercase, alphanumeric + underscore only)
	customTypes := map[string]config.CustomTypeMapping{
		"CUSTOMID": {GoType: "CustomID", SQLiteType: "INTEGER"},
	}

	catalog := model.NewCatalog()
	a := NewWithCustomTypes(catalog, customTypes)

	// Test that custom type is used
	got := a.SQLiteTypeToGo("CUSTOMID")
	if got != "CustomID" {
		t.Errorf("SQLiteTypeToGo(%q) with custom types = %q, want %q", "CUSTOMID", got, "CustomID")
	}

	// Test that non-custom types still work
	got = a.SQLiteTypeToGo("INTEGER")
	if got != "int64" {
		t.Errorf("SQLiteTypeToGo(%q) = %q, want %q", "INTEGER", got, "int64")
	}
}
