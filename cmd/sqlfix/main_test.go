package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBasic(t *testing.T) {
	tmpDir := setupTestFixtures(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, false, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "sqlfix:") {
		t.Errorf("expected output to contain 'sqlfix:', got %q", output)
	}
}

func TestRunDryRun(t *testing.T) {
	tmpDir := setupTestFixtures(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "sqlfix (dry-run):") {
		t.Errorf("expected output to contain 'sqlfix (dry-run):', got %q", output)
	}
}

func TestRunVerbose(t *testing.T) {
	tmpDir := setupTestFixtures(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, false, true, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	if stdout.Len() == 0 {
		t.Error("expected some output in verbose mode")
	}
}

func TestRunWithExplicitPaths(t *testing.T) {
	tmpDir := setupTestFixtures(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	queryPath := filepath.Join(tmpDir, "queries", "users.sql")

	exitCode := run(context.Background(), configPath, false, false, []string{queryPath}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "sqlfix:") {
		t.Errorf("expected output to contain 'sqlfix:', got %q", output)
	}
}

func TestRunWithMultiplePaths(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    title TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries1 := `-- name: ListUsers :many
SELECT id, name FROM users;
`
	queries2 := `-- name: ListPosts :many
SELECT id, title FROM posts;
`

	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries1)
	writeFile(t, filepath.Join(queriesDir, "posts.sql"), queries2)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, false, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "sqlfix:") {
		t.Errorf("expected output to contain 'sqlfix:', got %q", output)
	}
}

func TestRunMissingConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), "/nonexistent/config.toml", false, false, nil, stdout, stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for missing config, got %d", exitCode)
	}

	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "load config:") {
		t.Errorf("expected stderr to contain 'load config:', got %q", stderrOutput)
	}
}

func TestRunNoQueryFiles(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	// Create an empty queries directory - no matching files
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, false, false, nil, stdout, stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for no query files, got %d", exitCode)
	}

	stderrOutput := stderr.String()
	// Config validation catches empty queries before main logic runs
	if !strings.Contains(stderrOutput, "queries patterns matched no files") {
		t.Errorf("expected stderr to contain 'queries patterns matched no files', got %q", stderrOutput)
	}
}

func TestRunInvalidConfigPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), "", false, false, nil, stdout, stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for invalid config path, got %d", exitCode)
	}
}

func TestRunWithStarExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
SELECT * FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "expanded") {
		t.Errorf("expected output to mention star expansion, got %q", output)
	}
}

func TestRunWithAliasAddition(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUser :one
SELECT id, name FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "sqlfix (dry-run):") && !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention changes or no changes, got %q", output)
	}
}

func TestRunWithAggregateFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: CountUsers :one
SELECT COUNT(*) FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithJoinQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    title TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUserPosts :many
SELECT u.*, p.title
FROM users u
JOIN posts p ON p.user_id = u.id;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithExpressionColumns(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    salary REAL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserStats :one
SELECT id, name, salary * 1.1 FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunDedupPaths(t *testing.T) {
	tmpDir := setupTestFixtures(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	queryPath := filepath.Join(tmpDir, "queries", "users.sql")

	exitCode := run(context.Background(), configPath, false, false, []string{queryPath, queryPath}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithInvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, false, false, []string{"/nonexistent/query.sql"}, stdout, stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for invalid path, got %d", exitCode)
	}
}

func TestRunWithSchemaWarnings(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
SELECT * FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "duplicate table") {
		t.Errorf("expected stderr to contain duplicate table warning, got %q", stderrOutput)
	}
}

func TestRunWithEmptyQueryFile(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := ``
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention no changes for empty file, got %q", output)
	}
}

func TestRunWithInsertQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: CreateUser :exec
INSERT INTO users (name) VALUES (?);
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention no changes for INSERT query, got %q", output)
	}
}

func TestRunWithUpdateQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: UpdateUser :exec
UPDATE users SET name = ? WHERE id = ?;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention no changes for UPDATE query, got %q", output)
	}
}

func TestRunWithDeleteQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention no changes for DELETE query, got %q", output)
	}
}

func TestRunWithComplexQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    created_at DATETIME
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    total REAL,
    status TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserOrderSummary :many
SELECT 
    u.id,
    u.name,
    COUNT(o.id),
    SUM(o.total),
    AVG(o.total)
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.created_at > '2024-01-01'
GROUP BY u.id, u.name;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithSubquery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    title TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUsersWithPosts :many
SELECT id, name FROM users WHERE id IN (SELECT user_id FROM posts);
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithMultipleQueriesInFile(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
SELECT * FROM users;

-- name: GetUser :one
SELECT * FROM users WHERE id = 1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "expanded") {
		t.Errorf("expected output to mention star expansion, got %q", output)
	}
}

func TestRunWithCaseExpression(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    status TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserStatus :one
SELECT 
    id,
    CASE 
        WHEN status = 'active' THEN 1 
        ELSE 0 
    END 
FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithLiteralValues(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetConstants :one
SELECT 
    1,
    'hello',
    3.14,
    TRUE,
    FALSE
FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithExistingAliases(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUser :one
SELECT id AS user_id, name AS user_name FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention no changes for already-aliased columns, got %q", output)
	}
}

func TestRunWithMixedQueries(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUser :one
SELECT id, name FROM users WHERE id = 1;

-- name: CreateUser :exec
INSERT INTO users (name) VALUES (?);

-- name: ListUsers :many
SELECT * FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "expanded") {
		t.Errorf("expected output to mention star expansion, got %q", output)
	}
}

func TestRunNoSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty schema file (no tables defined)
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), "")

	queries := `-- name: ListUsers :many
SELECT * FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithView(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE VIEW active_users AS
SELECT id, name FROM users WHERE status = 'active';
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListActiveUsers :many
SELECT * FROM active_users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithUnionQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE admins (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListAllPeople :many
SELECT id, name FROM users
UNION ALL
SELECT id, name FROM admins;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func setupTestFixtures(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
SELECT users.id, users.name
FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	return tmpDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func TestRunWithOrderByAndLimit(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListRecentUsers :many
SELECT id, name FROM users ORDER BY created_at DESC LIMIT 10;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithGroupByAndHaving(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    total REAL,
    status TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetHighValueUsers :many
SELECT user_id, SUM(total) FROM orders GROUP BY user_id HAVING SUM(total) > 1000;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "orders.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithDistinct(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    status TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUniqueStatuses :many
SELECT DISTINCT status FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithWindowFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE sales (
    id INTEGER PRIMARY KEY,
    region TEXT,
    amount REAL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetSalesRanking :many
SELECT 
    id, 
    region, 
    amount,
    RANK() OVER (PARTITION BY region ORDER BY amount DESC) as rank
FROM sales;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "sales.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithCTE(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE employees (
    id INTEGER PRIMARY KEY,
    name TEXT,
    manager_id INTEGER
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetEmployeeHierarchy :many
WITH RECURSIVE hierarchy AS (
    SELECT id, name, manager_id FROM employees WHERE manager_id IS NULL
    UNION ALL
    SELECT e.id, e.name, e.manager_id 
    FROM employees e
    JOIN hierarchy h ON e.manager_id = h.id
)
SELECT * FROM hierarchy;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "employees.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithCommentsInQuery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
-- This query retrieves all users
SELECT id, name FROM users; -- simple select
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithMultipleSchemaFiles(t *testing.T) {
	tmpDir := t.TempDir()

	schema1 := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	schema2 := `CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    title TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema1.sql"), schema1)
	writeFile(t, filepath.Join(tmpDir, "schema2.sql"), schema2)

	queries := `-- name: ListUsersWithPosts :many
SELECT u.id, u.name, p.title
FROM users u
JOIN posts p ON p.user_id = u.id;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema1.sql", "schema2.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithCoalesce(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    email TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserDisplay :one
SELECT id, COALESCE(name, email, 'Unknown') FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithNullIf(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    status TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetActiveStatus :one
SELECT id, NULLIF(status, 'inactive') FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithDateFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    name TEXT,
    event_date DATETIME
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetEventInfo :many
SELECT 
    id, 
    name,
    DATE(event_date),
    STRFTIME('%Y', event_date)
FROM events;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "events.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithStringFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    email TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetFormattedUsers :many
SELECT 
    id,
    UPPER(name),
    LOWER(email),
    LENGTH(name)
FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithLeftJoin(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE profiles (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    bio TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsersWithProfiles :many
SELECT u.id, u.name, p.bio
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithCrossJoin(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetAllCombinations :many
SELECT c.name as category, p.name as product
FROM categories c
CROSS JOIN products p;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "combinations.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithInClause(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    status TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListActiveOrPendingUsers :many
SELECT id, name FROM users WHERE status IN ('active', 'pending');
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithBetweenClause(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name TEXT,
    price REAL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListMidRangeProducts :many
SELECT id, name, price FROM products WHERE price BETWEEN 10 AND 100;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "products.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithExistsSubquery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsersWithOrders :many
SELECT id, name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id);
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithNotNullConstraint(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
SELECT * FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "no changes") {
		t.Errorf("expected output to mention no changes, got %q", output)
	}
}

func TestRunWithForeignKey(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListPostsWithAuthors :many
SELECT p.id, p.title, u.name as author_name
FROM posts p
JOIN users u ON u.id = p.user_id;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "posts.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithIndexedColumn(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT
);

CREATE INDEX idx_users_email ON users(email);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: FindByEmail :one
SELECT id, email FROM users WHERE email = ?;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithUniqueConstraint(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT UNIQUE
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserByEmail :one
SELECT id, email FROM users WHERE email = ?;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithDefaultValue(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    status TEXT DEFAULT 'active'
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsers :many
SELECT * FROM users;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithCheckConstraint(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name TEXT,
    price REAL CHECK (price > 0)
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListProducts :many
SELECT id, name, price FROM products WHERE price < 100;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "products.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithLikeClause(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: SearchUsers :many
SELECT id, name FROM users WHERE name LIKE '%john%';
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithIsNullClause(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    deleted_at DATETIME
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListActiveUsers :many
SELECT id, name FROM users WHERE deleted_at IS NULL;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithIsNotNullClause(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    deleted_at DATETIME
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListDeletedUsers :many
SELECT id, name FROM users WHERE deleted_at IS NOT NULL;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithOffset(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: ListUsersPaged :many
SELECT id, name FROM users ORDER BY id LIMIT 10 OFFSET 20;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithCastExpression(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    price REAL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetFormattedPrice :one
SELECT id, CAST(price AS TEXT) FROM products WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "products.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithArithmeticExpression(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    quantity INTEGER,
    unit_price REAL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetOrderTotal :one
SELECT id, quantity * unit_price FROM orders WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "orders.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithConcatenation(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    first_name TEXT,
    last_name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetFullName :one
SELECT id, first_name || ' ' || last_name FROM users WHERE id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithMultipleJoins(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER
);

CREATE TABLE order_items (
    id INTEGER PRIMARY KEY,
    order_id INTEGER,
    product_name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserOrderItems :many
SELECT u.name, o.id as order_id, oi.product_name
FROM users u
JOIN orders o ON o.user_id = u.id
JOIN order_items oi ON oi.order_id = o.id;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "orders.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithSelfJoin(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE employees (
    id INTEGER PRIMARY KEY,
    name TEXT,
    manager_id INTEGER
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetEmployeeManager :one
SELECT e.name as employee, m.name as manager
FROM employees e
LEFT JOIN employees m ON m.id = e.manager_id
WHERE e.id = 1;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "employees.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithNestedSubquery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    category_id INTEGER,
    price REAL
);

CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetExpensiveProductsInCategory :many
SELECT p.id, p.price FROM products p
WHERE p.category_id = (SELECT id FROM categories WHERE name = 'Electronics')
AND p.price > (SELECT AVG(price) FROM products);
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "products.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithAliasInSubquery(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetUserStats :many
SELECT u.id, u.name, (SELECT COUNT(*) FROM users) as total_users
FROM users u;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithComplexAggregate(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE sales (
    id INTEGER PRIMARY KEY,
    region TEXT,
    amount REAL,
    sale_date DATE
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetRegionalStats :many
SELECT 
    region,
    COUNT(*) as sale_count,
    SUM(amount) as total_sales,
    AVG(amount) as avg_sale,
    MIN(amount) as min_sale,
    MAX(amount) as max_sale
FROM sales
GROUP BY region;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "sales.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithConditionalAggregation(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    status TEXT,
    amount REAL
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetOrderSummary :one
SELECT 
    COUNT(*) as total_orders,
    SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END) as completed_amount,
    SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) as pending_amount
FROM orders;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "orders.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithTableAliasStar(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    email TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: GetAllUserFields :many
SELECT u.* FROM users u;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "expanded") {
		t.Errorf("expected output to mention star expansion, got %q", output)
	}
}

func TestRunWithReturningClause(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: CreateUser :one
INSERT INTO users (name) VALUES (?) RETURNING id, name;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}

func TestRunWithOnConflict(t *testing.T) {
	tmpDir := t.TempDir()

	schema := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT UNIQUE,
    name TEXT
);
`
	writeFile(t, filepath.Join(tmpDir, "schema.sql"), schema)

	queries := `-- name: UpsertUser :one
INSERT INTO users (email, name) VALUES (?, ?)
ON CONFLICT(email) DO UPDATE SET name = excluded.name
RETURNING id, email, name;
`
	queriesDir := filepath.Join(tmpDir, "queries")
	if err := os.MkdirAll(queriesDir, 0750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	writeFile(t, filepath.Join(queriesDir, "users.sql"), queries)

	config := `package = "app"
out = "gen"
schemas = ["schema.sql"]
queries = ["queries/*.sql"]
`
	writeFile(t, filepath.Join(tmpDir, "db-catalyst.toml"), config)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")

	exitCode := run(context.Background(), configPath, true, false, nil, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", exitCode, stderr.String())
	}
}
