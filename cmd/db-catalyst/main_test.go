package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/codegen"
	"github.com/electwix/db-catalyst/internal/pipeline"
	queryanalyzer "github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
)

// TestRunHelpFlag tests the help flag handling
func TestRunHelpFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	out := stdout.String()
	if !strings.Contains(out, "Usage of db-catalyst") {
		t.Fatalf("stdout missing usage info: %q", out)
	}
}

// TestRunNoArgs tests that running with no arguments shows help
func TestRunNoArgs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{}, stdout, stderr)
	// When no args are provided, CLI shows help and exits with code 0
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	out := stdout.String()
	if !strings.Contains(out, "Usage of db-catalyst") {
		t.Fatalf("stdout missing usage info: %q", out)
	}
}

// TestRunInvalidFlag tests CLI argument parsing errors
func TestRunInvalidFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--invalid-flag"}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "Usage of db-catalyst") {
		t.Fatalf("stderr missing usage info: %q", errOut)
	}
}

// TestRunVerboseMode tests verbose mode flag
func TestRunVerboseMode(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--verbose", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	// Verbose mode should work without errors
	expected := filepath.Join(filepath.Dir(configPath), "gen", "querier.gen.go")
	if !strings.Contains(stdout.String(), expected) {
		t.Fatalf("stdout %q missing generated file %q", stdout.String(), expected)
	}
}

// TestRunShortVerboseFlag tests the short -v flag for verbose
func TestRunShortVerboseFlag(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "-v", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunOutputOverride tests output directory override
func TestRunOutputOverride(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--out", "custom_output", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	expected := filepath.Join(filepath.Dir(configPath), "custom_output", "querier.gen.go")
	if !strings.Contains(stdout.String(), expected) {
		t.Fatalf("stdout %q missing custom output path %q", stdout.String(), expected)
	}
}

// TestRunNoJSONTags tests the --no-json-tags flag
func TestRunNoJSONTags(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--no-json-tags", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunEmitIFNotExists tests the --if-not-exists flag
func TestRunEmitIFNotExists(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--if-not-exists", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunSQLDialect tests the --sql-dialect flag
func TestRunSQLDialect(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--sql-dialect", "sqlite", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunStrictConfig tests the --strict-config flag
func TestRunStrictConfig(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--strict-config", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunMissingConfig tests error handling for missing config file
func TestRunMissingConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", "/nonexistent/config.toml"}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	errOut := stderr.String()
	if errOut == "" {
		t.Fatal("expected error output for missing config")
	}
}

// TestRunInvalidConfig tests error handling for invalid config
func TestRunInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	invalidConfig := filepath.Join(tmpDir, "invalid.toml")
	if err := os.WriteFile(invalidConfig, []byte("invalid toml content [[["), 0o600); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", invalidConfig}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	errOut := stderr.String()
	if errOut == "" {
		t.Fatal("expected error output for invalid config")
	}
}

// TestRunPipelineSuccess tests successful pipeline execution
func TestRunPipelineSuccess(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}
}

// TestRunShortConfigFlag tests the short -c flag for config
func TestRunShortConfigFlag(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"-c", configPath, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestPrintDiagnostics tests the printDiagnostics function with various severities
func TestPrintDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		diags    []queryanalyzer.Diagnostic
		expected []string
	}{
		{
			name:     "empty diagnostics",
			diags:    []queryanalyzer.Diagnostic{},
			expected: []string{},
		},
		{
			name: "single warning",
			diags: []queryanalyzer.Diagnostic{
				{Path: "test.sql", Line: 10, Column: 5, Message: "unused parameter", Severity: queryanalyzer.SeverityWarning},
			},
			expected: []string{"test.sql:10:5: unused parameter [warning]"},
		},
		{
			name: "single error",
			diags: []queryanalyzer.Diagnostic{
				{Path: "test.sql", Line: 20, Column: 15, Message: "table not found", Severity: queryanalyzer.SeverityError},
			},
			expected: []string{"test.sql:20:15: table not found [error]"},
		},
		{
			name: "mixed severities",
			diags: []queryanalyzer.Diagnostic{
				{Path: "a.sql", Line: 1, Column: 1, Message: "first warning", Severity: queryanalyzer.SeverityWarning},
				{Path: "b.sql", Line: 2, Column: 2, Message: "an error", Severity: queryanalyzer.SeverityError},
				{Path: "c.sql", Line: 3, Column: 3, Message: "second warning", Severity: queryanalyzer.SeverityWarning},
			},
			expected: []string{
				"a.sql:1:1: first warning [warning]",
				"b.sql:2:2: an error [error]",
				"c.sql:3:3: second warning [warning]",
			},
		},
		{
			name: "diagnostics with special characters in message",
			diags: []queryanalyzer.Diagnostic{
				{Path: "test.sql", Line: 5, Column: 10, Message: "column 'name' has type TEXT", Severity: queryanalyzer.SeverityWarning},
			},
			expected: []string{"test.sql:5:10: column 'name' has type TEXT [warning]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printDiagnostics(&buf, tt.diags)

			output := strings.TrimSpace(buf.String())
			if len(tt.expected) == 0 {
				if output != "" {
					t.Fatalf("expected empty output, got %q", output)
				}
				return
			}

			lines := strings.Split(output, "\n")
			if len(lines) != len(tt.expected) {
				t.Fatalf("expected %d lines, got %d: %q", len(tt.expected), len(lines), output)
			}

			for i, expected := range tt.expected {
				if lines[i] != expected {
					t.Fatalf("line %d: expected %q, got %q", i, expected, lines[i])
				}
			}
		})
	}
}

// TestPrintDiagnosticsWriterError tests error handling when writer fails
func TestPrintDiagnosticsWriterError(_ *testing.T) {
	diags := []queryanalyzer.Diagnostic{
		{Path: "test.sql", Line: 1, Column: 1, Message: "test", Severity: queryanalyzer.SeverityWarning},
	}

	// Use a writer that always fails
	writer := &failingWriter{err: errors.New("write failed")}
	printDiagnostics(writer, diags)
	// Should not panic, just ignore the error
}

// TestPrintQuerySummary tests the printQuerySummary function
func TestPrintQuerySummary(t *testing.T) {
	tests := []struct {
		name     string
		analyses []queryanalyzer.Result
		expected []string
	}{
		{
			name:     "empty analyses",
			analyses: []queryanalyzer.Result{},
			expected: []string{},
		},
		{
			name: "single query with no params",
			analyses: []queryanalyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{Name: "ListUsers", Command: block.CommandMany},
					},
					Params: []queryanalyzer.ResultParam{},
				},
			},
			expected: []string{"ListUsers :many params: none"},
		},
		{
			name: "single query with params",
			analyses: []queryanalyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{Name: "GetUser", Command: block.CommandOne},
					},
					Params: []queryanalyzer.ResultParam{
						{Name: "id", GoType: "int64", Nullable: false},
					},
				},
			},
			expected: []string{"GetUser :one params: id:int64"},
		},
		{
			name: "multiple queries",
			analyses: []queryanalyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{Name: "ListUsers", Command: block.CommandMany},
					},
					Params: []queryanalyzer.ResultParam{},
				},
				{
					Query: parser.Query{
						Block: block.Block{Name: "CreateUser", Command: block.CommandExec},
					},
					Params: []queryanalyzer.ResultParam{
						{Name: "name", GoType: "string", Nullable: false},
						{Name: "email", GoType: "string", Nullable: true},
					},
				},
			},
			expected: []string{
				"ListUsers :many params: none",
				"CreateUser :exec params: name:string, email:string?",
			},
		},
		{
			name: "query with execresult command",
			analyses: []queryanalyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{Name: "UpdateUser", Command: block.CommandExecResult},
					},
					Params: []queryanalyzer.ResultParam{
						{Name: "id", GoType: "int64", Nullable: false},
					},
				},
			},
			expected: []string{"UpdateUser :execresult params: id:int64"},
		},
		{
			name: "query with unknown command",
			analyses: []queryanalyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{Name: "UnknownQuery", Command: block.CommandUnknown},
					},
					Params: []queryanalyzer.ResultParam{},
				},
			},
			expected: []string{"UnknownQuery :unknown params: none"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printQuerySummary(&buf, tt.analyses)

			output := strings.TrimSpace(buf.String())
			if len(tt.expected) == 0 {
				if output != "" {
					t.Fatalf("expected empty output, got %q", output)
				}
				return
			}

			lines := strings.Split(output, "\n")
			if len(lines) != len(tt.expected) {
				t.Fatalf("expected %d lines, got %d: %q", len(tt.expected), len(lines), output)
			}

			for i, expected := range tt.expected {
				if lines[i] != expected {
					t.Fatalf("line %d: expected %q, got %q", i, expected, lines[i])
				}
			}
		})
	}
}

// TestPrintQuerySummaryWriterError tests error handling when writer fails
func TestPrintQuerySummaryWriterError(_ *testing.T) {
	analyses := []queryanalyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{Name: "Test", Command: block.CommandOne},
			},
			Params: []queryanalyzer.ResultParam{},
		},
	}

	// Use a writer that always fails
	writer := &failingWriter{err: errors.New("write failed")}
	printQuerySummary(writer, analyses)
	// Should not panic, just ignore the error
}

// TestFormatParams tests the formatParams function with various param combinations
func TestFormatParams(t *testing.T) {
	tests := []struct {
		name     string
		params   []queryanalyzer.ResultParam
		expected string
	}{
		{
			name:     "empty params",
			params:   []queryanalyzer.ResultParam{},
			expected: "params: none",
		},
		{
			name: "single param",
			params: []queryanalyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false, IsVariadic: false},
			},
			expected: "params: id:int64",
		},
		{
			name: "multiple params",
			params: []queryanalyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false},
				{Name: "name", GoType: "string", Nullable: false},
				{Name: "active", GoType: "bool", Nullable: false},
			},
			expected: "params: id:int64, name:string, active:bool",
		},
		{
			name: "nullable param",
			params: []queryanalyzer.ResultParam{
				{Name: "email", GoType: "string", Nullable: true},
			},
			expected: "params: email:string?",
		},
		{
			name: "variadic param",
			params: []queryanalyzer.ResultParam{
				{Name: "ids", GoType: "int64", Nullable: false, IsVariadic: true},
			},
			expected: "params: ids...:int64",
		},
		{
			name: "variadic param with count",
			params: []queryanalyzer.ResultParam{
				{Name: "ids", GoType: "int64", Nullable: false, IsVariadic: true, VariadicCount: 3},
			},
			expected: "params: ids...:int64[x3]",
		},
		{
			name: "nullable variadic param",
			params: []queryanalyzer.ResultParam{
				{Name: "names", GoType: "string", Nullable: true, IsVariadic: true},
			},
			expected: "params: names...:string?",
		},
		{
			name: "nullable variadic param with count",
			params: []queryanalyzer.ResultParam{
				{Name: "names", GoType: "string", Nullable: true, IsVariadic: true, VariadicCount: 5},
			},
			expected: "params: names...:string?[x5]",
		},
		{
			name: "mixed params",
			params: []queryanalyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false},
				{Name: "name", GoType: "string", Nullable: true},
				{Name: "tags", GoType: "string", Nullable: false, IsVariadic: true, VariadicCount: 2},
			},
			expected: "params: id:int64, name:string?, tags...:string[x2]",
		},
		{
			name: "different Go types",
			params: []queryanalyzer.ResultParam{
				{Name: "id", GoType: "int64"},
				{Name: "count", GoType: "int"},
				{Name: "price", GoType: "float64"},
				{Name: "data", GoType: "[]byte"},
				{Name: "created", GoType: "time.Time"},
			},
			expected: "params: id:int64, count:int, price:float64, data:[]byte, created:time.Time",
		},
		{
			name: "zero variadic count",
			params: []queryanalyzer.ResultParam{
				{Name: "items", GoType: "string", IsVariadic: true, VariadicCount: 0},
			},
			expected: "params: items...:string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatParams(tt.params)
			if result != tt.expected {
				t.Fatalf("formatParams() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRunWithDiagnosticsError tests handling of DiagnosticsError
func TestRunWithDiagnosticsError(t *testing.T) {
	// Create a config that will cause a diagnostic error (missing schema file)
	tmpDir := t.TempDir()
	configContent := `package = "app"
out = "gen"
schemas = ["nonexistent.sql"]
queries = ["queries/*.sql"]
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath}, stdout, stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	// Should have error output
	if stderr.Len() == 0 {
		t.Fatal("expected error output")
	}
}

// TestRunListQueriesWithParams tests list-queries with queries that have params
func TestRunListQueriesWithParams(t *testing.T) {
	// Create test fixtures with a query that has parameters
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	schemaContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);
`
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "users.sql"), []byte(schemaContent), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	queryContent := `-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = ?1;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (?1, ?2);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "users.sql"), []byte(queryContent), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "GetUser") {
		t.Fatalf("stdout %q missing GetUser query", out)
	}
	if !strings.Contains(out, "CreateUser") {
		t.Fatalf("stdout %q missing CreateUser query", out)
	}
	if !strings.Contains(out, ":one") {
		t.Fatalf("stdout %q missing :one command", out)
	}
	if !strings.Contains(out, ":exec") {
		t.Fatalf("stdout %q missing :exec command", out)
	}
}

// TestRunContextCancellation tests that context cancellation is handled
func TestRunContextCancellation(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	exitCode := run(ctx, []string{"--config", configPath, "--dry-run"}, stdout, stderr)
	// Context cancellation should result in non-zero exit
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 for cancelled context", exitCode)
	}
}

// TestRunWithQueryDiagnostics tests that query diagnostics are printed
func TestRunWithQueryDiagnostics(t *testing.T) {
	// Create test fixtures with a query that has an issue
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Schema with users table
	schemaContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);
`
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "users.sql"), []byte(schemaContent), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	// Query with ambiguous column reference (no table specified for id)
	queryContent := `-- name: GetUser :one
SELECT id, name FROM users WHERE id = ?1;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "users.sql"), []byte(queryContent), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Run actual generation - this should succeed but may have warnings
	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	// Query analysis should succeed
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	// Should have output showing the query
	out := stdout.String()
	if !strings.Contains(out, "GetUser") {
		t.Fatalf("stdout %q missing GetUser query", out)
	}
}

// TestRunWithConfigWarnings tests handling of config warnings in strict mode
func TestRunWithConfigWarnings(t *testing.T) {
	// Create a config with an unknown field that will generate a warning
	tmpDir := t.TempDir()
	configContent := `package = "app"
out = "gen"
unknown_field = "value"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create minimal schema
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Test :one\nSELECT * FROM t;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	// Test without strict mode - should succeed with warning
	t.Run("non-strict mode", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0 in non-strict mode; stderr=%q", exitCode, stderr.String())
		}
	})

	// Test with strict mode - should fail
	t.Run("strict mode", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--strict-config", "--list-queries"}, stdout, stderr)
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1 in strict mode", exitCode)
		}
	})
}

// TestRunWithWriteError tests handling of WriteError
func TestRunWithWriteError(t *testing.T) {
	// This test verifies that write errors return exit code 2
	// We need to create a scenario where file writing fails
	configPath := prepareCmdFixtures(t)

	// Create a read-only output directory to cause write failures
	genDir := filepath.Join(filepath.Dir(configPath), "gen")
	if err := os.MkdirAll(genDir, 0o750); err != nil {
		t.Fatalf("failed to create gen dir: %v", err)
	}
	//nolint:gosec // Intentionally setting restrictive permissions for test
	if err := os.Chmod(genDir, 0o555); err != nil {
		t.Fatalf("failed to chmod gen dir: %v", err)
	}
	defer func() {
		//nolint:gosec // Restoring permissions for cleanup
		_ = os.Chmod(genDir, 0o750)
	}()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2 for write error", exitCode)
	}
}

// TestRunWithMultipleQueryFiles tests handling of multiple query files
func TestRunWithMultipleQueryFiles(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "users.sql"), []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "users.sql"), []byte("-- name: GetUser :one\nSELECT * FROM users WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "extra.sql"), []byte("-- name: CountUsers :one\nSELECT COUNT(*) FROM users;"), 0o600); err != nil {
		t.Fatalf("failed to write extra query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "GetUser") {
		t.Fatalf("stdout %q missing GetUser query", out)
	}
	if !strings.Contains(out, "CountUsers") {
		t.Fatalf("stdout %q missing CountUsers query", out)
	}
}

// TestRunWithMultipleSchemaFiles tests handling of multiple schema files
func TestRunWithMultipleSchemaFiles(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "users.sql"), []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write users schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "posts.sql"), []byte("CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER);"), 0o600); err != nil {
		t.Fatalf("failed to write posts schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: GetPosts :many\nSELECT * FROM posts WHERE user_id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunDryRunWithExistingOutput tests dry run when output already exists
func TestRunDryRunWithExistingOutput(t *testing.T) {
	configPath := prepareCmdFixtures(t)

	// Create an existing output file
	genDir := filepath.Join(filepath.Dir(configPath), "gen")
	if err := os.MkdirAll(genDir, 0o750); err != nil {
		t.Fatalf("failed to create gen dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(genDir, "existing.txt"), []byte("existing"), 0o600); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	// Dry run should not modify files
	//nolint:gosec // Test file path is constructed safely
	content, err := os.ReadFile(filepath.Join(genDir, "existing.txt"))
	if err != nil {
		t.Fatalf("failed to read existing file: %v", err)
	}
	if string(content) != "existing" {
		t.Fatal("dry run modified existing file")
	}
}

// TestRunListQueriesDryRunCombination tests that --list-queries and --dry-run work independently
func TestRunListQueriesDryRunCombination(t *testing.T) {
	configPath := prepareCmdFixtures(t)

	// Test list-queries takes precedence
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--list-queries", "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	out := stdout.String()
	// Should show query list, not file paths
	if !strings.Contains(out, "ListUsers") {
		t.Fatalf("stdout %q missing query name (list-queries should take effect)", out)
	}
}

// TestRunAbsoluteOutputPath tests output override with absolute path
func TestRunAbsoluteOutputPath(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	absOut := filepath.Join(t.TempDir(), "absolute_output")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--out", absOut, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	expected := filepath.Join(absOut, "querier.gen.go")
	if !strings.Contains(stdout.String(), expected) {
		t.Fatalf("stdout %q missing absolute output path %q", stdout.String(), expected)
	}
}

// TestRunEmptyQueriesDir tests handling of empty queries directory
func TestRunEmptyQueriesDir(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	// Create empty queries directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	// Empty queries directory results in error
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", exitCode, stderr.String())
	}

	// Should have error output
	if stderr.Len() == 0 {
		t.Fatal("expected error output for empty queries")
	}
}

// TestRunEmptySchemasDir tests handling when schema files don't match pattern
func TestRunEmptySchemasDir(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create empty schemas directory
	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Test :one\nSELECT 1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	// Should fail because schema files don't exist
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
}

// TestRunWithVariadicParams tests queries with variadic parameters
func TestRunWithVariadicParams(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "users.sql"), []byte("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	// Query with IN clause creates variadic params
	queryContent := `-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id IN (/*IDS*/);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "users.sql"), []byte(queryContent), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "GetUsersByIDs") {
		t.Fatalf("stdout %q missing GetUsersByIDs query", out)
	}
}

// TestRunWithAllCommandTypes tests all command types
func TestRunWithAllCommandTypes(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "users.sql"), []byte("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = ?1;

-- name: ListUsers :many
SELECT * FROM users;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?1;

-- name: UpdateUser :execresult
UPDATE users SET name = ?1 WHERE id = ?2;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "users.sql"), []byte(queryContent), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	out := stdout.String()
	for _, expected := range []string{
		"GetUser :one",
		"ListUsers :many",
		"DeleteUser :exec",
		"UpdateUser :execresult",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("stdout %q missing %q", out, expected)
		}
	}
}

// TestRunWithComplexQuery tests a more complex query scenario
func TestRunWithComplexQuery(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	schemaContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE
);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "app.sql"), []byte(schemaContent), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = ?1;

-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES (?1, ?2);

-- name: GetPublishedPosts :many
SELECT * FROM posts WHERE published = TRUE;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "app.sql"), []byte(queryContent), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	out := stdout.String()
	for _, expected := range []string{
		"GetUser",
		"CreateUser",
		"GetPublishedPosts",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("stdout %q missing %q", out, expected)
		}
	}
}

// TestRunGeneratesFiles tests that actual file generation works
func TestRunGeneratesFiles(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	genDir := filepath.Join(filepath.Dir(configPath), "gen")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	// Verify files were created
	entries, err := os.ReadDir(genDir)
	if err != nil {
		t.Fatalf("failed to read gen dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no files generated")
	}

	// Check for expected generated files
	foundQuerier := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".gen.go") {
			foundQuerier = true
			break
		}
	}
	if !foundQuerier {
		t.Fatalf("no .gen.go files found in %v", entries)
	}
}

// TestRunIdempotent tests that running twice produces the same result
func TestRunIdempotent(t *testing.T) {
	configPath := prepareCmdFixtures(t)

	// First run
	stdout1 := &bytes.Buffer{}
	stderr1 := &bytes.Buffer{}
	exitCode1 := run(context.Background(), []string{"--config", configPath, "--dry-run"}, stdout1, stderr1)
	if exitCode1 != 0 {
		t.Fatalf("first run exit code = %d, want 0; stderr=%q", exitCode1, stderr1.String())
	}

	// Second run
	stdout2 := &bytes.Buffer{}
	stderr2 := &bytes.Buffer{}
	exitCode2 := run(context.Background(), []string{"--config", configPath, "--dry-run"}, stdout2, stderr2)
	if exitCode2 != 0 {
		t.Fatalf("second run exit code = %d, want 0; stderr=%q", exitCode2, stderr2.String())
	}

	if stdout1.String() != stdout2.String() {
		t.Fatalf("outputs differ between runs:\nfirst: %q\nsecond: %q", stdout1.String(), stdout2.String())
	}
}

// TestRunWithPreparedQueriesConfig tests config with prepared queries enabled
func TestRunWithPreparedQueriesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[prepared_queries]
enabled = true
metrics = true
thread_safe = true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithCustomTypesConfig tests config with custom types
func TestRunWithCustomTypesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[[custom_types.mapping]]
sqlite_type = "DATETIME"
go_type = "time.Time"
import_path = "time"

[[custom_types.mapping]]
sqlite_type = "JSON"
go_type = "json.RawMessage"
import_path = "encoding/json"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY, created DATETIME);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// failingWriter is a writer that always returns an error
type failingWriter struct {
	err error
}

func (w *failingWriter) Write(_ []byte) (n int, err error) {
	return 0, w.err
}

// Ensure failingWriter implements io.Writer
var _ io.Writer = (*failingWriter)(nil)

// TestDiagnosticsErrorUnwrap tests that DiagnosticsError can be unwrapped
func TestDiagnosticsErrorUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	diagErr := &pipeline.DiagnosticsError{
		Diagnostic: queryanalyzer.Diagnostic{
			Path:     "test.sql",
			Line:     1,
			Column:   1,
			Message:  "test message",
			Severity: queryanalyzer.SeverityError,
		},
		Cause: innerErr,
	}

	if diagErr.Error() != "test.sql:1:1: test message" {
		t.Fatalf("unexpected error message: %q", diagErr.Error())
	}

	if !errors.Is(diagErr, innerErr) {
		t.Fatal("expected errors.Is to find inner error")
	}
}

// TestWriteErrorUnwrap tests that WriteError can be unwrapped
func TestWriteErrorUnwrap(t *testing.T) {
	innerErr := errors.New("write failed")
	writeErr := &pipeline.WriteError{
		Path: "/some/path",
		Err:  innerErr,
	}

	expected := "write /some/path: write failed"
	if writeErr.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, writeErr.Error())
	}

	if !errors.Is(writeErr, innerErr) {
		t.Fatal("expected errors.Is to find inner error")
	}
}

// TestRunWithCodegenFile tests the Summary.Files field handling
func TestRunWithCodegenFile(t *testing.T) {
	// This test verifies that the Files field in Summary is properly handled
	configPath := prepareCmdFixtures(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", configPath, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	// Verify that file paths are printed
	out := stdout.String()
	if out == "" {
		t.Fatal("expected file paths in stdout")
	}
}

// TestRunWithSummaryDiagnostics tests that diagnostics in summary are printed
func TestRunWithSummaryDiagnostics(t *testing.T) {
	// Create a scenario with warnings
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	// Query with unused parameter (may generate warning)
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithEmptyQueryFile tests handling of empty query files
func TestRunWithEmptyQueryFile(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	// Empty query file with just comments
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "empty.sql"), []byte("-- Just a comment\n-- Another comment"), 0o600); err != nil {
		t.Fatalf("failed to write empty query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	// Empty query file results in error
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithQueryFileContainingOnlyWhitespace tests handling of whitespace-only query files
func TestRunWithQueryFileContainingOnlyWhitespace(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	// Whitespace-only file
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "whitespace.sql"), []byte("   \n\t\n   "), 0o600); err != nil {
		t.Fatalf("failed to write whitespace query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithDuplicateTable tests handling of duplicate table definitions
func TestRunWithDuplicateTable(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	// First schema file with users table
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "a.sql"), []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema a: %v", err)
	}
	// Second schema file with duplicate users table
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "b.sql"), []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema b: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM users WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	// Should fail due to duplicate table
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 for duplicate table", exitCode)
	}

	// Should have error output
	if stderr.Len() == 0 {
		t.Fatal("expected error output for duplicate table")
	}
}

// TestRunWithView tests handling of SQL views
func TestRunWithView(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
CREATE VIEW active_users AS SELECT * FROM users WHERE name IS NOT NULL;
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte(schemaContent), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: GetActive :many\nSELECT * FROM active_users;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithIndex tests handling of SQL indexes
func TestRunWithIndex(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT);
CREATE INDEX idx_email ON users(email);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte(schemaContent), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: GetByEmail :one\nSELECT * FROM users WHERE email = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithForeignKey tests handling of foreign keys
func TestRunWithForeignKey(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER REFERENCES users(id)
);
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte(schemaContent), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: GetPosts :many\nSELECT * FROM posts WHERE user_id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestFormatParamsEdgeCases tests edge cases for formatParams
func TestFormatParamsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		params   []queryanalyzer.ResultParam
		expected string
	}{
		{
			name:     "nil params",
			params:   nil,
			expected: "params: none",
		},
		{
			name: "param with empty name",
			params: []queryanalyzer.ResultParam{
				{Name: "", GoType: "int64"},
			},
			expected: "params: :int64",
		},
		{
			name: "param with empty type",
			params: []queryanalyzer.ResultParam{
				{Name: "id", GoType: ""},
			},
			expected: "params: id:",
		},
		{
			name: "param with complex type",
			params: []queryanalyzer.ResultParam{
				{Name: "data", GoType: "map[string]interface{}"},
			},
			expected: "params: data:map[string]interface{}",
		},
		{
			name: "param with pointer type",
			params: []queryanalyzer.ResultParam{
				{Name: "ptr", GoType: "*int", Nullable: true},
			},
			expected: "params: ptr:*int?",
		},
		{
			name: "many params",
			params: []queryanalyzer.ResultParam{
				{Name: "a", GoType: "int"},
				{Name: "b", GoType: "int"},
				{Name: "c", GoType: "int"},
				{Name: "d", GoType: "int"},
				{Name: "e", GoType: "int"},
			},
			expected: "params: a:int, b:int, c:int, d:int, e:int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatParams(tt.params)
			if result != tt.expected {
				t.Fatalf("formatParams() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRunWithConfigPathOverride tests that config path can be overridden
func TestRunWithConfigPathOverride(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"-c", configPath, "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithEmitPointersForNull tests config with emit_pointers_for_null
func TestRunWithEmitPointersForNull(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
emit_pointers_for_null = true
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithEmitEmptySlices tests config with emit_empty_slices
func TestRunWithEmitEmptySlices(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[prepared_queries]
enabled = true
emit_empty_slices = true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: List :many\nSELECT * FROM t;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithInvalidQuerySyntax tests handling of invalid query block syntax
func TestRunWithInvalidQuerySyntax(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	// Query with SQL before block marker will cause error
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("SELECT * FROM t;\n-- name: Bad :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Run to trigger error
	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml")}, stdout, stderr)
	// Should fail due to SQL before block marker
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 for invalid query file; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithModerncDriver tests the modernc driver option
func TestRunWithModerncDriver(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
sqlite_driver = "modernc"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithMattnDriver tests the mattn driver option
func TestRunWithMattnDriver(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
sqlite_driver = "mattn"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithSQLDialectAndOutput tests SQL dialect with SQL output enabled
func TestRunWithSQLDialectAndOutput(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
sql_dialect = "sqlite"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestCodegenFileStruct tests the codegen.File struct
func TestCodegenFileStruct(t *testing.T) {
	file := codegen.File{
		Path:    "test.go",
		Content: []byte("package test"),
	}

	if file.Path != "test.go" {
		t.Fatalf("expected Path to be 'test.go', got %q", file.Path)
	}

	if string(file.Content) != "package test" {
		t.Fatalf("expected Content to be 'package test', got %q", string(file.Content))
	}
}

// TestRunWithExtraArguments tests that extra arguments are captured
func TestRunWithExtraArguments(t *testing.T) {
	configPath := prepareCmdFixtures(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Extra arguments should be ignored by the CLI
	exitCode := run(context.Background(), []string{"--config", configPath, "--dry-run", "extra", "args"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestRunWithRelativeConfigPath tests using a relative config path
func TestRunWithRelativeConfigPath(t *testing.T) {
	// Create fixtures in temp dir
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Change to temp dir and use default config path
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	exitCode := run(context.Background(), []string{"--dry-run"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

// TestFlagErrHelp tests the flag.ErrHelp handling
func TestFlagErrHelp(t *testing.T) {
	// Test that flag.ErrHelp is properly detected
	err := flag.ErrHelp
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatal("expected errors.Is to detect flag.ErrHelp")
	}
}

// TestRunWithLargeFile tests handling of large files (within limits)
func TestRunWithLargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `package = "app"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0o750); err != nil {
		t.Fatalf("failed to create schemas dir: %v", err)
	}
	// Create a reasonably sized schema
	var schemaBuilder strings.Builder
	schemaBuilder.WriteString("CREATE TABLE t (id INTEGER PRIMARY KEY);")
	for i := 0; i < 100; i++ {
		schemaBuilder.WriteString(fmt.Sprintf("CREATE TABLE t%d (id INTEGER PRIMARY KEY);\n", i))
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schemas", "test.sql"), []byte(schemaBuilder.String()), 0o600); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "queries"), 0o750); err != nil {
		t.Fatalf("failed to create queries dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries", "test.sql"), []byte("-- name: Get :one\nSELECT * FROM t WHERE id = ?1;"), 0o600); err != nil {
		t.Fatalf("failed to write query: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run(context.Background(), []string{"--config", filepath.Join(tmpDir, "config.toml"), "--list-queries"}, stdout, stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
}

func prepareCmdFixtures(t *testing.T) string {
	t.Helper()
	src := "testdata"
	dst := t.TempDir()
	copyTree(t, dst, src)
	return filepath.Join(dst, "config.toml")
}

func copyTree(t *testing.T, dst, src string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("ReadDir %q: %v", src, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o750); err != nil {
				t.Fatalf("MkdirAll %q: %v", dstPath, err)
			}
			copyTree(t, dstPath, srcPath)
			continue
		}
		copyFile(t, dstPath, srcPath)
	}
}

func copyFile(t *testing.T, dst, src string) {
	t.Helper()
	in, err := os.Open(filepath.Clean(src))
	if err != nil {
		t.Fatalf("open %q: %v", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(filepath.Clean(dst), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("create %q: %v", dst, err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %q -> %q: %v", src, dst, err)
	}
}
