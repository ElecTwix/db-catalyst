package bench

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/fileset"
	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/pipeline"
)

type discardWriter struct{}

func (w *discardWriter) WriteFile(_ string, data []byte) error {
	_ = data
	return nil
}

func BenchmarkPipeline(b *testing.B) {
	// Create a temporary directory with test fixtures
	tmpDir, err := os.MkdirTemp("", "db-catalyst-bench")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Copy test fixtures
	fixtureDir := "../pipeline/testdata/e2e/complex_sqlite"
	if err := copyDir(fixtureDir, tmpDir); err != nil {
		b.Fatalf("copy fixtures: %v", err)
	}

	logger := logging.New(logging.Options{Writer: io.Discard})
	env := pipeline.Environment{
		Logger:     logging.NewSlogAdapter(logger),
		FSResolver: fileset.NewOSResolver,
		Writer:     &discardWriter{},
	}
	pipe := pipeline.Pipeline{Env: env}

	ctx := context.Background()
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := pipe.Run(ctx, pipeline.RunOptions{ConfigPath: configPath})
		if err != nil {
			b.Fatalf("pipeline run: %v", err)
		}
	}
}

// BenchmarkPipelineSmall tests with minimal schema and queries
func BenchmarkPipelineSmall(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "db-catalyst-bench-small")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create minimal schema
	schemaContent := `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL
);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0600); err != nil {
		b.Fatalf("write schema: %v", err)
	}

	// Create minimal queries
	queriesContent := `
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY username;

-- name: CreateUser :one
INSERT INTO users (username, email) VALUES (?, ?) RETURNING id;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queriesContent), 0600); err != nil {
		b.Fatalf("write queries: %v", err)
	}

	// Create config
	configContent := `
package = "test"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0600); err != nil {
		b.Fatalf("write config: %v", err)
	}

	logger := logging.New(logging.Options{Writer: io.Discard})
	env := pipeline.Environment{
		Logger:     logging.NewSlogAdapter(logger),
		FSResolver: fileset.NewOSResolver,
		Writer:     &discardWriter{},
	}
	pipe := pipeline.Pipeline{Env: env}

	ctx := context.Background()
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := pipe.Run(ctx, pipeline.RunOptions{ConfigPath: configPath})
		if err != nil {
			b.Fatalf("pipeline run: %v", err)
		}
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." || strings.HasPrefix(rel, "..") {
			return nil
		}
		dstPath := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0600)
	})
}
