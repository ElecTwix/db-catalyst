package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"testing/fstest"

	"github.com/electwix/db-catalyst/internal/fileset"
)

func TestLoadSuccess(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	copyFixtureDir(t, tempDir, "schemas")
	copyFixtureDir(t, tempDir, "queries")

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`)

	result, err := Load(configPath, LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}

	if result.Plan.Package != "demo" {
		t.Fatalf("unexpected package: %q", result.Plan.Package)
	}

	if result.Plan.SQLiteDriver != DriverModernC {
		t.Fatalf("expected default driver %q, got %q", DriverModernC, result.Plan.SQLiteDriver)
	}

	expectedOut := filepath.Join(tempDir, "gen")
	if result.Plan.Out != expectedOut {
		t.Fatalf("expected out %q, got %q", expectedOut, result.Plan.Out)
	}

	expectedSchemas := []string{
		filepath.Join(tempDir, "schemas", "books.sql"),
		filepath.Join(tempDir, "schemas", "users.sql"),
	}
	if !slices.Equal(result.Plan.Schemas, expectedSchemas) {
		t.Fatalf("unexpected schema files: %v", result.Plan.Schemas)
	}

	expectedQueries := []string{
		filepath.Join(tempDir, "queries", "find_user.sql"),
		filepath.Join(tempDir, "queries", "list_users.sql"),
	}
	if !slices.Equal(result.Plan.Queries, expectedQueries) {
		t.Fatalf("unexpected query files: %v", result.Plan.Queries)
	}

	if result.Plan.PreparedQueries != (PreparedQueries{}) {
		t.Fatalf("expected prepared queries defaults to zero values, got %+v", result.Plan.PreparedQueries)
	}
}

func TestLoadPreparedQueriesConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	copyFixtureDir(t, tempDir, "schemas")
	copyFixtureDir(t, tempDir, "queries")

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[prepared_queries]
enabled = true
metrics = true
thread_safe = true
`)

	result, err := Load(configPath, LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	prepared := result.Plan.PreparedQueries
	if !prepared.Enabled {
		t.Fatalf("expected prepared queries enabled")
	}
	if !prepared.Metrics {
		t.Fatalf("expected metrics flag set")
	}
	if !prepared.ThreadSafe {
		t.Fatalf("expected thread_safe flag set")
	}
}

func TestLoadInvalidPackage(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	copyFixtureDir(t, tempDir, "schemas")
	copyFixtureDir(t, tempDir, "queries")

	configPath := writeConfig(t, tempDir, `
package = "123bad"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`)

	_, err := Load(configPath, LoadOptions{})
	if err == nil {
		t.Fatal("expected error for invalid package name")
	}
	if !strings.Contains(err.Error(), "invalid package name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsAbsoluteOut(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	copyFixtureDir(t, tempDir, "schemas")
	copyFixtureDir(t, tempDir, "queries")

	configPath := writeConfig(t, tempDir, fmt.Sprintf(`
package = "demo"
out = %q
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
`, filepath.Join(tempDir, "gen")))

	_, err := Load(configPath, LoadOptions{})
	if err == nil {
		t.Fatal("expected error for absolute out path")
	}
	if !strings.Contains(err.Error(), "out must be a relative path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadMissingSchemaPattern(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
schemas = ["schemas/*.missing"]
queries = ["queries/*.sql"]
`)

	resolver := fileset.NewResolver(fstest.MapFS{
		"queries/find_user.sql":  &fstest.MapFile{},
		"queries/list_users.sql": &fstest.MapFile{},
	})

	_, err := Load(configPath, LoadOptions{Resolver: &resolver})
	if err == nil {
		t.Fatal("expected error for missing schema glob matches")
	}
	if !strings.Contains(err.Error(), "schemas patterns matched no files") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "schemas/*.missing") {
		t.Fatalf("error should mention missing pattern, got: %v", err)
	}
}

func TestLoadPreparedQueriesUnknownKeysStrict(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[prepared_queries]
unknown = true
`)

	_, err := Load(configPath, LoadOptions{Strict: true})
	if err == nil {
		t.Fatal("expected strict mode to reject unknown prepared_queries keys")
	}
	if !strings.Contains(err.Error(), "unknown prepared_queries keys") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error should mention offending key, got: %v", err)
	}
}

func TestLoadStrictUnknownKeys(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
extra = "value"
`)

	_, err := Load(configPath, LoadOptions{Strict: true})
	if err == nil {
		t.Fatal("expected strict mode to reject unknown keys")
	}
	if !strings.Contains(err.Error(), "unknown configuration keys") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "extra") {
		t.Fatalf("error should mention offending key, got: %v", err)
	}
}

func TestLoadPreparedQueriesUnknownKeysWarning(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	copyFixtureDir(t, tempDir, "schemas")
	copyFixtureDir(t, tempDir, "queries")

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[prepared_queries]
unknown = true
`)

	result, err := Load(configPath, LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning, got %v", result.Warnings)
	}
	warning := result.Warnings[0]
	if !strings.Contains(warning, "unknown prepared_queries keys") {
		t.Fatalf("warning missing prepared_queries message: %q", warning)
	}
	if !strings.Contains(warning, "unknown") {
		t.Fatalf("warning should mention offending key, got: %q", warning)
	}
}

func TestLoadNonStrictUnknownKeysWarning(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	copyFixtureDir(t, tempDir, "schemas")
	copyFixtureDir(t, tempDir, "queries")

	configPath := writeConfig(t, tempDir, `
package = "demo"
out = "gen"
sqlite_driver = "mattn"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]
extra = "value"
`)

	result, err := Load(configPath, LoadOptions{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if result.Plan.SQLiteDriver != DriverMattN {
		t.Fatalf("expected mattn driver, got %q", result.Plan.SQLiteDriver)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning, got %v", result.Warnings)
	}
	warning := result.Warnings[0]
	if !strings.Contains(warning, "unknown configuration keys") {
		t.Fatalf("warning missing unknown keys message: %q", warning)
	}
	if !strings.Contains(warning, "extra") {
		t.Fatalf("warning should mention offending key, got: %q", warning)
	}
}

func copyFixtureDir(tb testing.TB, dstRoot, name string) {
	tb.Helper()

	srcDir := filepath.Join("testdata", name)
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		tb.Fatalf("read fixture dir: %v", err)
	}

	dstDir := filepath.Join(dstRoot, name)
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		tb.Fatalf("create fixture dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Clean(filepath.Join(srcDir, entry.Name())))
		if err != nil {
			tb.Fatalf("read fixture file: %v", err)
		}

		if err := os.WriteFile(filepath.Join(dstDir, entry.Name()), data, 0o600); err != nil {
			tb.Fatalf("write fixture file: %v", err)
		}
	}
}

func writeConfig(tb testing.TB, dir, contents string) string {
	tb.Helper()

	path := filepath.Join(dir, "db-catalyst.toml")
	clean := strings.TrimSpace(contents) + "\n"
	if err := os.WriteFile(path, []byte(clean), 0o600); err != nil {
		tb.Fatalf("write config: %v", err)
	}
	return path
}
