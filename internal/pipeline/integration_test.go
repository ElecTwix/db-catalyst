package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/fileset"
	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/pipeline"
	"log/slog"
)

// TestSimpleSchema tests code generation for a simple schema.
func TestSimpleSchema(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := `package = "store"
out = "generated"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE
);`
	queries := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;

-- name: ListUsers :many
SELECT * FROM users;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (:name, :email);
`

	writeFile(t, tmpDir, "db-catalyst.toml", config)
	writeFile(t, tmpDir, "schema.sql", schema)
	writeFile(t, tmpDir, "queries.sql", queries)

	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			FSResolver: fileset.NewOSResolver,
			Logger:     logging.NewSlogAdapter(slog.Default()),
			Writer:     pipeline.NewOSWriter(),
		},
	}

	_, err := p.Run(ctx, pipeline.RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Verify generated files exist
	generatedDir := filepath.Join(tmpDir, "generated")
	expectFiles := []string{
		"models.gen.go",
		"query_get_user.go",
		"query_list_users.go",
		"query_create_user.go",
	}

	for _, file := range expectFiles {
		path := filepath.Join(generatedDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", file)
		}
	}
}

// TestComplexSchema tests code generation with all constraint types.
func TestComplexSchema(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := `package = "complex"
out = "generated"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schema := `CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    bio TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    author_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    slug TEXT UNIQUE,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    view_count INTEGER DEFAULT 0,
    FOREIGN KEY (author_id) REFERENCES authors(id)
);

CREATE INDEX idx_posts_author ON posts (author_id);
CREATE INDEX idx_posts_published ON posts (published) WHERE published = TRUE;
`
	queries := `-- name: GetAuthor :one
SELECT 
    id,
    name,
    bio
FROM authors
WHERE id = :id;

-- name: ListPosts :many
SELECT * FROM posts WHERE published = TRUE ORDER BY view_count DESC;
`

	writeFile(t, tmpDir, "db-catalyst.toml", config)
	writeFile(t, tmpDir, "schema.sql", schema)
	writeFile(t, tmpDir, "queries.sql", queries)

	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			FSResolver: fileset.NewOSResolver,
			Logger:     logging.NewSlogAdapter(slog.Default()),
			Writer:     pipeline.NewOSWriter(),
		},
	}

	_, err := p.Run(ctx, pipeline.RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Verify models contain expected types
	modelsPath := filepath.Join(tmpDir, "generated", "models.gen.go")
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		t.Fatalf("failed to read models: %v", err)
	}

	expected := []string{
		"type Authors struct",
		"type Posts struct",
		"Id",
		"Name",
		"Bio",
		"CreatedAt",
		"Title",
		"Slug",
		"Content",
		"Published",
		"ViewCount",
		"AuthorId",
	}

	for _, exp := range expected {
		if !strings.Contains(string(content), exp) {
			t.Errorf("models.gen.go missing: %s", exp)
		}
	}
}

// TestCTEs tests recursive and non-recursive CTEs.
func TestCTEs(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := `package = "cte"
out = "generated"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schema := `CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id INTEGER REFERENCES categories(id)
);

CREATE TABLE employees (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    manager_id INTEGER REFERENCES employees(id)
);`
	queries := `-- name: GetCategoryTree :many
WITH RECURSIVE tree AS (
    SELECT id, name, parent_id, 0 as depth
    FROM categories
    WHERE id = :root_id

    UNION ALL

    SELECT c.id, c.name, c.parent_id, t.depth + 1
    FROM categories c
    JOIN tree t ON c.parent_id = t.id
)
SELECT * FROM tree;

-- name: GetOrgChart :many
WITH RECURSIVE org_chart AS (
    SELECT id, name, manager_id, 0 as level
    FROM employees
    WHERE manager_id IS NULL

    UNION ALL

    SELECT e.id, e.name, e.manager_id, oc.level + 1
    FROM employees e
    JOIN org_chart oc ON e.manager_id = oc.id
)
SELECT * FROM org_chart;
`

	writeFile(t, tmpDir, "db-catalyst.toml", config)
	writeFile(t, tmpDir, "schema.sql", schema)
	writeFile(t, tmpDir, "queries.sql", queries)

	p := &pipeline.Pipeline{
		Env: pipeline.Environment{
			FSResolver: fileset.NewOSResolver,
			Logger:     logging.NewSlogAdapter(slog.Default()),
			Writer:     pipeline.NewOSWriter(),
		},
	}

	_, err := p.Run(ctx, pipeline.RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Verify CTE queries were generated
	queryPath := filepath.Join(tmpDir, "generated", "query_get_category_tree.go")
	if _, err := os.Stat(queryPath); os.IsNotExist(err) {
		t.Error("expected query_get_category_tree.go to exist")
	}
}

// TestEdgeCases tests edge cases.
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		queries string
		wantErr bool
	}{
		{
			name:    "empty schema",
			schema:  "",
			queries: "",
			wantErr: false, // Empty schema is valid, just produces no output
		},
		{
			name: "unicode identifiers",
			schema: `CREATE TABLE 用户 (
    编号 INTEGER PRIMARY KEY,
    名称 TEXT
);`,
			queries: `-- name: Get用户 :one
SELECT * FROM 用户 WHERE 编号 = :id;`,
			wantErr: false,
		},
		{
			name: "reserved words as identifiers",
			schema: `CREATE TABLE "table" (
    "select" INTEGER PRIMARY KEY,
    "from" TEXT
);`,
			queries: `-- name: GetTable :one
SELECT * FROM "table" WHERE "select" = :select;`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()

			config := `package = "edge"
out = "generated"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
			writeFile(t, tmpDir, "db-catalyst.toml", config)
			writeFile(t, tmpDir, "schema.sql", tt.schema)
			writeFile(t, tmpDir, "queries.sql", tt.queries)

			p := &pipeline.Pipeline{
				Env: pipeline.Environment{
					FSResolver: fileset.NewOSResolver,
					Logger:     logging.NewNopLogger(), // Suppress logs for edge cases
					Writer:     pipeline.NewOSWriter(),
				},
			}

			_, err := p.Run(ctx, pipeline.RunOptions{
				ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
				DryRun:     true, // Don't write files for edge case tests
			})

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
