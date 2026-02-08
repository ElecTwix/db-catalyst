package pipeline

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/electwix/db-catalyst/internal/cache"
	"github.com/electwix/db-catalyst/internal/fileset"
	"github.com/electwix/db-catalyst/internal/logging"
)

func BenchmarkPipeline_Run(b *testing.B) {
	// Create a temporary directory with test fixtures
	tmpDir, err := os.MkdirTemp("", "db-catalyst-bench")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    published BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_posts_user ON posts (user_id);
`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (:name, :email);

-- name: GetPostsByUser :many
SELECT * FROM posts WHERE user_id = :user_id;
`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o600); err != nil {
		b.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o600); err != nil {
		b.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o600); err != nil {
		b.Fatalf("write queries: %v", err)
	}

	logger := logging.New(logging.Options{Writer: io.Discard})
	pipeline := &Pipeline{
		Env: Environment{
			FSResolver: fileset.NewOSResolver,
			Logger:     logging.NewSlogAdapter(logger),
			Writer:     &MemoryWriter{},
		},
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
		DryRun:     true,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := pipeline.Run(ctx, opts)
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkPipeline_Run_WithCache(b *testing.B) {
	// Create a temporary directory with test fixtures
	tmpDir, err := os.MkdirTemp("", "db-catalyst-bench-cache")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o600); err != nil {
		b.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o600); err != nil {
		b.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o600); err != nil {
		b.Fatalf("write queries: %v", err)
	}

	logger := logging.New(logging.Options{Writer: io.Discard})
	c := cache.NewMemoryCache()

	pipeline := &Pipeline{
		Env: Environment{
			FSResolver: fileset.NewOSResolver,
			Logger:     logging.NewSlogAdapter(logger),
			Writer:     &MemoryWriter{},
			Cache:      c,
		},
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
		DryRun:     true,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := pipeline.Run(ctx, opts)
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}
