package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMigration(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	sqlcYaml := `version: "2"
sql:
  - engine: "sqlite"
    queries: "query/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "db"

overrides:
  - go_type: "github.com/company/types.ID"
    db_type: "INTEGER"
  - go_type: "github.com/company/types.Status"
    db_type: "TEXT"
    pointer: true
`

	if err := os.WriteFile(filepath.Join(tmpDir, "sqlc.yaml"), []byte(sqlcYaml), 0644); err != nil {
		t.Fatalf("write sqlc.yaml: %v", err)
	}

	opts := options{
		DryRun: true,
	}

	err := migrate(ctx, tmpDir, opts)

	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	dbcatPath := filepath.Join(tmpDir, "db-catalyst.toml")
	if _, err := os.Stat(dbcatPath); err == nil {
		t.Error("db-catalyst.toml should not be created in dry-run mode")
	}
}

func TestMigrationWithOutput(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	sqlcYaml := `version: "2"
sql:
  - engine: "sqlite"
    queries: "query/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "db"

overrides:
  - go_type: "github.com/company/types.ID"
    db_type: "INTEGER"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "sqlc.yaml"), []byte(sqlcYaml), 0644); err != nil {
		t.Fatalf("write sqlc.yaml: %v", err)
	}

	opts := options{
		DryRun:    false,
		Overwrite: true,
	}

	err := migrate(ctx, tmpDir, opts)

	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	dbcatPath := filepath.Join(tmpDir, "db-catalyst.toml")
	data, err := os.ReadFile(dbcatPath)
	if err != nil {
		t.Fatalf("read db-catalyst.toml: %v", err)
	}

	content := string(data)
	if !bytes.Contains([]byte(content), []byte("package = \"db\"")) {
		t.Error("missing package in generated config")
	}
	if !bytes.Contains([]byte(content), []byte("sqlite_driver = \"modernc\"")) {
		t.Error("missing sqlite_driver in generated config")
	}
}
