// Package e2e provides end-to-end tests that verify generated code compiles and runs correctly.
//
//nolint:goconst // Test fixtures use repeated content strings
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

// TestGeneratedGoCode_CompilesAndRuns tests that generated Go code compiles and works correctly.
func TestGeneratedGoCode_CompilesAndRuns(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory for the test module
	tmpDir, err := os.MkdirTemp("", "db-catalyst-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create go.mod for the test module
	goModContent := `module testapp

go 1.23

require modernc.org/sqlite v1.34.1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0600); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a simple schema
	schemaSQL := `
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    published INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaSQL), 0600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	// Create queries
	queriesSQL := `
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES (?, ?)
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts WHERE id = ?;

-- name: ListPostsByUser :many
SELECT * FROM posts WHERE user_id = ? ORDER BY id;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queriesSQL), 0600); err != nil {
		t.Fatalf("failed to write queries: %v", err)
	}

	// Create config with relative paths
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

	// Run the pipeline using default OS writer
	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			Logger: logging.NewNopLogger(), // Suppress logs in tests
		},
	}

	opts := pipeline.RunOptions{
		ConfigPath: configPath,
	}

	summary, err := p.Run(ctx, opts)
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if len(summary.Diagnostics) > 0 {
		for _, diag := range summary.Diagnostics {
			t.Logf("Diagnostic: %s", diag.Message)
		}
	}

	if len(summary.Files) == 0 {
		t.Fatal("no files generated")
	}

	t.Logf("Generated %d files:", len(summary.Files))
	for _, f := range summary.Files {
		t.Logf("  - %s (%d bytes)", f.Path, len(f.Content))
	}

	// Verify files were written to disk
	expectedFiles := []string{
		"gen/models.gen.go",
		"gen/querier.gen.go",
		"gen/helpers.gen.go",
		"gen/query_get_user.go",
		"gen/query_list_users.go",
		"gen/query_create_user.go",
	}

	for _, filePath := range expectedFiles {
		fullPath := filepath.Join(tmpDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file not found: %s", filePath)
		}
	}

	t.Log("✅ Generated files successfully")

	// Verify the generated code compiles
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code failed to compile:\n%s", output)
	}

	t.Log("✅ Generated code compiles successfully")
}

// TestGeneratedGoCode_WithComplexSchema tests a more complex schema.
func TestGeneratedGoCode_WithComplexSchema(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "db-catalyst-e2e-complex-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	goModContent := `module testapp

go 1.23

require modernc.org/sqlite v1.34.1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0600); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Complex schema with various types and constraints
	schemaSQL := `
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    bio TEXT,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    published INTEGER NOT NULL DEFAULT 0,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER,
    FOREIGN KEY (author_id) REFERENCES authors(id) ON DELETE CASCADE
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE post_tags (
    post_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, tag_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_published ON posts(published);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaSQL), 0600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	queriesSQL := `
-- name: GetAuthorWithPosts :one
SELECT 
    a.id, a.name, a.email, a.bio,
    COUNT(p.id) as total_posts
FROM authors a
LEFT JOIN posts p ON a.id = p.author_id
WHERE a.id = ?
GROUP BY a.id;

-- name: ListPublishedPosts :many
SELECT p.*, a.name as author_name
FROM posts p
JOIN authors a ON p.author_id = a.id
WHERE p.published = 1
ORDER BY p.created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (author_id, title, content, published)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: IncrementViewCount :exec
UPDATE posts SET view_count = view_count + 1 WHERE id = ?;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queriesSQL), 0600); err != nil {
		t.Fatalf("failed to write queries: %v", err)
	}

	cfgContent := `package = "complexdb"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	if err := os.WriteFile(configPath, []byte(cfgContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			Logger: logging.NewNopLogger(),
		},
	}

	opts := pipeline.RunOptions{
		ConfigPath: configPath,
	}

	summary, err := p.Run(ctx, opts)
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	t.Logf("Generated %d files", len(summary.Files))

	// Verify files exist
	if _, err := os.Stat(filepath.Join(tmpDir, "gen", "models.gen.go")); os.IsNotExist(err) {
		t.Error("expected models.gen.go to be generated")
	}

	t.Log("✅ Complex schema generated files successfully")
}

// TestGeneratedGoCode_WithPreparedQueries tests prepared query generation.
func TestGeneratedGoCode_WithPreparedQueries(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		config    string
		needBuild bool // whether this config needs compilation test
	}{
		{
			name: "basic",
			config: `package = "productdb"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]

[prepared_queries]
enabled = true
`,
			needBuild: true,
		},
		{
			name: "threadsafe",
			config: `package = "productdb"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]

[prepared_queries]
enabled = true
thread_safe = true
`,
			needBuild: true,
		},
		{
			name: "with_metrics",
			config: `package = "productdb"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schema.sql"]
queries = ["queries.sql"]

[prepared_queries]
enabled = true
metrics = true
thread_safe = true
`,
			needBuild: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "db-catalyst-e2e-prepared-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			goModContent := `module testapp

go 1.23

require modernc.org/sqlite v1.34.1
`
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0600); err != nil {
				t.Fatalf("failed to write go.mod: %v", err)
			}

			schemaSQL := `
CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    price INTEGER NOT NULL
);
`
			if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaSQL), 0600); err != nil {
				t.Fatalf("failed to write schema: %v", err)
			}

			queriesSQL := `
-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: ListProducts :many
SELECT * FROM products ORDER BY name;

-- name: CreateProduct :exec
INSERT INTO products (sku, name, price) VALUES (?, ?, ?);
`
			if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queriesSQL), 0600); err != nil {
				t.Fatalf("failed to write queries: %v", err)
			}

			configPath := filepath.Join(tmpDir, "db-catalyst.toml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			p := &pipeline.Pipeline{
				Env: pipeline.Environment{
					Logger: logging.NewNopLogger(),
				},
			}

			opts := pipeline.RunOptions{
				ConfigPath: configPath,
			}

			_, err = p.Run(ctx, opts)
			if err != nil {
				t.Fatalf("pipeline failed: %v", err)
			}

			if tt.needBuild {
				// Verify the generated code compiles - catches variable redeclaration bugs
				cmd := exec.CommandContext(ctx, "go", "build", "./...")
				cmd.Dir = tmpDir
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("generated code failed to compile:\n%s", output)
				}
				t.Logf("✅ Prepared queries (%s) compile successfully", tt.name)
			}
		})
	}
}
