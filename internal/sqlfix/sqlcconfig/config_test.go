package sqlcconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/electwix/db-catalyst/internal/sqlfix/sqlcconfig"
)

func TestLoadConfigParsesOverrides(t *testing.T) {
	dir := t.TempDir()

	schema := "CREATE TABLE users (id INTEGER PRIMARY KEY, status TEXT);"
	if err := os.WriteFile(filepath.Join(dir, "schema.sql"), []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	sqlc := `version: 2
sql:
  - schema: ["schema.sql"]
    queries: ["queries.sql"]
overrides:
  - db_type: "INTEGER"
    go_type: github.com/example/types.ID
  - column:
      table: users
      column: status
    go_type:
      import: github.com/example/types
      package: types
      type: Status
      pointer: true
`
	configPath := filepath.Join(dir, "sqlc.yaml")
	if err := os.WriteFile(configPath, []byte(sqlc), 0o600); err != nil {
		t.Fatalf("write sqlc config: %v", err)
	}

	cfg, err := sqlcconfig.Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := len(cfg.Overrides), 2; got != want {
		t.Fatalf("Overrides length = %d, want %d", got, want)
	}

	if len(cfg.SchemaWarnings()) != 0 {
		t.Fatalf("expected no schema warnings, got %v", cfg.SchemaWarnings())
	}

	typ, ok := cfg.ColumnType(sqlcconfig.ColumnRef{Table: "users", Column: "status"})
	if !ok {
		t.Fatalf("ColumnType returned false")
	}
	if typ != "TEXT" {
		t.Fatalf("ColumnType = %q, want TEXT", typ)
	}

	col := cfg.Overrides[1].Column
	if col.Table != "users" || col.Name != "status" {
		t.Fatalf("unexpected column target: %+v", col)
	}

	info, err := cfg.Overrides[0].GoType.Normalize()
	if err != nil {
		t.Fatalf("Normalize go_type: %v", err)
	}
	if info.TypeName != "ID" {
		t.Fatalf("go type TypeName = %q, want ID", info.TypeName)
	}
	if info.Pointer {
		t.Fatalf("go type pointer should be false")
	}

	info, err = cfg.Overrides[1].GoType.Normalize()
	if err != nil {
		t.Fatalf("Normalize go_type: %v", err)
	}
	if !info.Pointer {
		t.Fatalf("column override pointer flag should be true")
	}
}
