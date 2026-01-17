package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

func TestRunWritesMergedConfig(t *testing.T) {
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
  - db_type: TEXT
    go_type: github.com/example/types.Label
  - column:
      table: users
      column: status
    go_type:
      import: github.com/example/types
      package: types
      type: Status
      pointer: true
`
	sqlcPath := filepath.Join(dir, "sqlc.yaml")
	if err := os.WriteFile(sqlcPath, []byte(sqlc), 0o600); err != nil {
		t.Fatalf("write sqlc config: %v", err)
	}

	dbConfig := `package = "app"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]
[` + "custom_types" + `]
`
	dbPath := filepath.Join(dir, "db-catalyst.toml")
	if err := os.WriteFile(dbPath, []byte(dbConfig), 0o600); err != nil {
		t.Fatalf("write db config: %v", err)
	}

	outPath := filepath.Join(dir, "out.toml")
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--sqlc-config", sqlcPath, "--db-config", dbPath, "--out", outPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run exit = %d, stderr = %s", exitCode, stderr.String())
	}

	data, err := os.ReadFile(filepath.Clean(outPath))
	if err != nil {
		t.Fatalf("read out config: %v", err)
	}

	var merged map[string]any
	if err := toml.Unmarshal(data, &merged); err != nil {
		t.Fatalf("unmarshal merged config: %v", err)
	}

	customTypes, ok := merged["custom_types"].(map[string]any)
	if !ok {
		t.Fatalf("custom_types missing in merged config")
	}
	mappings, ok := customTypes["mapping"].([]any)
	if !ok || len(mappings) != 2 {
		t.Fatalf("expected 2 custom type mappings, got %v", customTypes)
	}
}

func TestRunDryRun(t *testing.T) {
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
  - db_type: TEXT
    go_type: github.com/example/types.Label
`
	sqlcPath := filepath.Join(dir, "sqlc.yaml")
	if err := os.WriteFile(sqlcPath, []byte(sqlc), 0o600); err != nil {
		t.Fatalf("write sqlc config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--sqlc-config", sqlcPath, "--db-config", filepath.Join(dir, "missing.toml"), "--dry-run"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run exit = %d, stderr = %s", exitCode, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected dry-run to emit config contents")
	}
}
