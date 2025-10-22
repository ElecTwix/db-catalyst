package overrides_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/sqlfix/overrides"
	"github.com/electwix/db-catalyst/internal/sqlfix/sqlcconfig"
)

func loadTestConfig(t *testing.T) sqlcconfig.Config {
	t.Helper()
	dir := t.TempDir()

	schema := "CREATE TABLE users (id INTEGER PRIMARY KEY, status TEXT);"
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(schema), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sqlc := `version: 2
sql:
  - schema: ["schema.sql"]
    queries: ["queries.sql"]
overrides:
  - db_type: TEXT
    go_type:
      import: github.com/example/types
      package: types
      type: Label
  - column:
      table: users
      column: status
    go_type: github.com/example/types.Status
`
	configPath := filepath.Join(dir, "sqlc.yaml")
	if err := os.WriteFile(configPath, []byte(sqlc), 0o644); err != nil {
		t.Fatalf("write sqlc config: %v", err)
	}

	cfg, err := sqlcconfig.Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestConvertOverrides(t *testing.T) {
	cfg := loadTestConfig(t)

	mappings, warnings := overrides.ConvertOverrides(cfg)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(mappings) != 2 {
		t.Fatalf("mappings length = %d, want 2", len(mappings))
	}

	// Find db_type mapping
	var dbType config.CustomTypeMapping
	var column config.CustomTypeMapping
	for _, m := range mappings {
		switch m.SQLiteType {
		case "TEXT":
			if m.GoType == "Label" {
				dbType = m
			} else if m.GoType == "Status" {
				column = m
			}
		}
	}

	if dbType.CustomType == "" {
		t.Fatalf("db_type mapping missing")
	}
	if dbType.Pointer {
		t.Fatalf("db_type mapping pointer should be false")
	}
	if dbType.GoImport != "github.com/example/types" {
		t.Fatalf("db_type GoImport = %q, want github.com/example/types", dbType.GoImport)
	}

	if column.CustomType == "" {
		t.Fatalf("column mapping missing")
	}
	if column.Pointer {
		t.Fatalf("column mapping pointer should be false")
	}
	if column.CustomType == dbType.CustomType {
		t.Fatalf("column mapping should generate unique custom type name")
	}
}

func TestMergeMappings(t *testing.T) {
	existing := []config.CustomTypeMapping{
		{
			CustomType: "text_label",
			SQLiteType: "TEXT",
			GoType:     "Label",
			GoImport:   "github.com/example/types",
			GoPackage:  "types",
		},
	}
	additions := overrides.Mappings{
		{
			CustomType: "text_label",
			SQLiteType: "TEXT",
			GoType:     "Label",
			GoImport:   "github.com/example/types",
			GoPackage:  "types",
		},
		{
			CustomType: "users_status_status",
			SQLiteType: "TEXT",
			GoType:     "Status",
			GoImport:   "github.com/example/types",
			GoPackage:  "types",
			Pointer:    true,
		},
	}

	merged, warnings := overrides.MergeMappings(existing, additions)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings when mappings identical")
	}
	if len(merged) != 2 {
		t.Fatalf("merged length = %d, want 2", len(merged))
	}

	conflicting := overrides.Mappings{
		{
			CustomType: "text_label",
			SQLiteType: "TEXT",
			GoType:     "Other",
		},
	}

	merged, warnings = overrides.MergeMappings(existing, conflicting)
	if len(warnings) != 1 {
		t.Fatalf("expected conflict warning, got %v", warnings)
	}
	if len(merged) != 1 {
		t.Fatalf("conflicting merge should keep original entry only")
	}
}
