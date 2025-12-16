package sqlfix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSchemaCatalog_SingleSchema(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.sql")
	schema := "CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT);"
	if err := os.WriteFile(schemaPath, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	result, err := LoadSchemaCatalog([]string{schemaPath}, nil)
	if err != nil {
		t.Fatalf("LoadSchemaCatalog: %v", err)
	}
	if result.Catalog == nil {
		t.Fatal("expected catalog to be initialized")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	table := result.Catalog.Tables["users"]
	if table == nil {
		t.Fatal("expected users table to be present in catalog")
	}
	if len(table.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(table.Columns))
	}
}

func TestLoadSchemaCatalog_DuplicateTableWarning(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "schema_one.sql")
	second := filepath.Join(dir, "schema_two.sql")
	ddl := "CREATE TABLE users (id INTEGER);"
	if err := os.WriteFile(first, []byte(ddl), 0o600); err != nil {
		t.Fatalf("write first schema: %v", err)
	}
	if err := os.WriteFile(second, []byte(ddl), 0o600); err != nil {
		t.Fatalf("write second schema: %v", err)
	}

	result, err := LoadSchemaCatalog([]string{first, second}, nil)
	if err != nil {
		t.Fatalf("LoadSchemaCatalog: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected duplicate warning, got none")
	}

	foundDuplicate := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "duplicate table") {
			foundDuplicate = true
			break
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected duplicate table warning, got %v", result.Warnings)
	}
	if len(result.Catalog.Tables) != 1 {
		t.Fatalf("expected 1 table in catalog, got %d", len(result.Catalog.Tables))
	}
}

func TestLoadSchemaCatalog_ParserWarnings(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.sql")
	schema := "CREATE TEMP TABLE things (id INTEGER);"
	if err := os.WriteFile(schemaPath, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	result, err := LoadSchemaCatalog([]string{schemaPath}, nil)
	if err != nil {
		t.Fatalf("LoadSchemaCatalog: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected parser warning, got none")
	}

	found := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "TEMP/TEMPORARY modifiers are ignored") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected TEMP warning, got %v", result.Warnings)
	}
}
