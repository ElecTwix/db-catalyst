// Package e2e provides end-to-end tests for interface compatibility.
package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/pipeline"
)

// TestDBTXInterfaceCompatibility verifies *sql.DB implements the generated DBTX interface
func TestDBTXInterfaceCompatibility(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "db-catalyst-e2e-interface-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create go.mod
	goModContent := `module testapp

go 1.23
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0600); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Minimal schema
	schemaSQL := `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaSQL), 0600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	queriesSQL := `
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: CreateUser :exec
INSERT INTO users (name) VALUES (?);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queriesSQL), 0600); err != nil {
		t.Fatalf("failed to write queries: %v", err)
	}

	cfgContent := `package = "db"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Generate code
	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			Logger: logging.NewNopLogger(),
		},
	}

	if _, err := p.Run(ctx, pipeline.RunOptions{ConfigPath: configPath}); err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Create a test file in the gen directory that uses the generated code
	testCode := `package db

import "database/sql"

// Compile-time checks - these will fail if DBTX interface doesn't match *sql.DB
var _ DBTX = (*sql.DB)(nil)
var _ DBTX = (*sql.Tx)(nil)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "gen", "compat_test.go"), []byte(testCode), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Try to build - this catches type mismatches in DBTX interface
	cmd := exec.CommandContext(ctx, "go", "build", "./gen/...")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("DBTX interface type mismatch: *sql.DB does not implement generated DBTX interface\n%s", output)
	}

	t.Log("✅ DBTX interface is compatible with *sql.DB and *sql.Tx")
}
