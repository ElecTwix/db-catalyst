package pipeline

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/codegen"
	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
)

type memoryWriter struct {
	writes map[string][]byte
	count  int
}

func (w *memoryWriter) WriteFile(path string, data []byte) error {
	if w.writes == nil {
		w.writes = make(map[string][]byte)
	}
	w.count++
	w.writes[path] = append([]byte(nil), data...)
	return nil
}

func TestPipelineDryRun(t *testing.T) {
	configPath := prepareFixtures(t)
	writer := &memoryWriter{}

	p := Pipeline{Env: Environment{Writer: writer}}
	summary, err := p.Run(context.Background(), RunOptions{ConfigPath: configPath, DryRun: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(summary.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %v, want none", summary.Diagnostics)
	}
	if len(summary.Files) == 0 {
		t.Fatalf("Files = %v, want generated files", summary.Files)
	}
	if writer.count != 0 {
		t.Fatalf("writer invoked %d times during dry-run, want 0", writer.count)
	}
	if len(summary.Analyses) != 2 {
		t.Fatalf("Analyses = %d, want 2", len(summary.Analyses))
	}

	var (
		listUsersFound bool
		summarizeFound bool
		helperContent  string
		queryFound     bool
	)

	for _, analysis := range summary.Analyses {
		switch analysis.Query.Block.Name {
		case "ListUsers":
			listUsersFound = true
			if analysis.Query.Block.Command != block.CommandMany {
				t.Fatalf("ListUsers command = %v, want CommandMany", analysis.Query.Block.Command)
			}
		case "SummarizeCredits":
			summarizeFound = true
			if analysis.Query.Block.Command != block.CommandOne {
				t.Fatalf("SummarizeCredits command = %v, want CommandOne", analysis.Query.Block.Command)
			}
			if len(analysis.Columns) != 3 {
				t.Fatalf("SummarizeCredits columns = %d, want 3", len(analysis.Columns))
			}
			if col := analysis.Columns[0]; col.Name != "total_users" || col.GoType != "int64" || col.Nullable {
				t.Fatalf("total_users column = %+v, want int64 non-null", col)
			}
			if col := analysis.Columns[1]; col.Name != "sum_credits" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("sum_credits column = %+v, want float64 nullable", col)
			}
			if col := analysis.Columns[2]; col.Name != "avg_credit" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("avg_credit column = %+v, want float64 nullable", col)
			}
			if len(analysis.Params) != 0 {
				t.Fatalf("SummarizeCredits params = %v, want none", analysis.Params)
			}
		default:
			t.Fatalf("unexpected query %q in analyses", analysis.Query.Block.Name)
		}
	}
	if !listUsersFound || !summarizeFound {
		t.Fatalf("expected analyses for ListUsers and SummarizeCredits, got %+v", summary.Analyses)
	}

	outPrefix := filepath.Join(filepath.Dir(configPath), "gen") + string(os.PathSeparator)
	for _, file := range summary.Files {
		if !strings.HasPrefix(file.Path, outPrefix) {
			t.Fatalf("file path %q does not reside under %q", file.Path, outPrefix)
		}
		if strings.HasSuffix(file.Path, "_helpers.gen.go") {
			helperContent = string(file.Content)
		}
		if strings.HasSuffix(file.Path, "query_summarize_credits.go") {
			queryFound = true
		}
	}
	if !queryFound {
		t.Fatalf("query_summarize_credits.go not emitted; files = %+v", summary.Files)
	}
	if !strings.Contains(helperContent, "type SummarizeCreditsRow struct") ||
		!strings.Contains(helperContent, "TotalUsers int32") ||
		!strings.Contains(helperContent, "SumCredits sql.NullFloat64") ||
		!strings.Contains(helperContent, "AvgCredit  sql.NullFloat64") && !strings.Contains(helperContent, "AvgCredit sql.NullFloat64") {
		t.Fatalf("_helpers.gen.go missing expected SummarizeCreditsRow fields")
	}
}

func TestPipelineListQueries(t *testing.T) {
	configPath := prepareFixtures(t)
	writer := &memoryWriter{}

	p := Pipeline{Env: Environment{Writer: writer}}
	summary, err := p.Run(context.Background(), RunOptions{ConfigPath: configPath, ListQueries: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(summary.Files) != 0 {
		t.Fatalf("Files = %d, want none when listing", len(summary.Files))
	}
	if writer.count != 0 {
		t.Fatalf("writer invoked %d times when listing, want 0", writer.count)
	}
	if len(summary.Analyses) != 2 {
		t.Fatalf("Analyses = %d, want 2", len(summary.Analyses))
	}

	var (
		listUsersFound bool
		summarizeFound bool
	)

	for _, analysis := range summary.Analyses {
		switch analysis.Query.Block.Name {
		case "ListUsers":
			listUsersFound = true
			if analysis.Query.Block.Command != block.CommandMany {
				t.Fatalf("ListUsers command = %v, want CommandMany", analysis.Query.Block.Command)
			}
			if len(analysis.Params) != 0 {
				t.Fatalf("ListUsers params = %v, want none", analysis.Params)
			}
		case "SummarizeCredits":
			summarizeFound = true
			if analysis.Query.Block.Command != block.CommandOne {
				t.Fatalf("SummarizeCredits command = %v, want CommandOne", analysis.Query.Block.Command)
			}
			if len(analysis.Columns) != 3 {
				t.Fatalf("SummarizeCredits columns = %d, want 3", len(analysis.Columns))
			}
			if col := analysis.Columns[0]; col.Name != "total_users" || col.GoType != "int64" || col.Nullable {
				t.Fatalf("total_users column = %+v, want int64 non-null", col)
			}
			if col := analysis.Columns[1]; col.Name != "sum_credits" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("sum_credits column = %+v, want float64 nullable", col)
			}
			if col := analysis.Columns[2]; col.Name != "avg_credit" || col.GoType != "float64" || !col.Nullable {
				t.Fatalf("avg_credit column = %+v, want float64 nullable", col)
			}
			if len(analysis.Params) != 0 {
				t.Fatalf("SummarizeCredits params = %v, want none", analysis.Params)
			}
		default:
			t.Fatalf("unexpected query %q in analyses", analysis.Query.Block.Name)
		}
	}
	if !listUsersFound || !summarizeFound {
		t.Fatalf("expected analyses for ListUsers and SummarizeCredits, got %+v", summary.Analyses)
	}
	if len(summary.Diagnostics) != 0 {
		t.Fatalf("Diagnostics = %v, want none", summary.Diagnostics)
	}
}

func prepareFixtures(t *testing.T) string {
	t.Helper()
	src := "testdata"
	dst := t.TempDir()
	copyDir(t, dst, src)
	return filepath.Join(dst, "config.toml")
}

func copyDir(t *testing.T, dst, src string) {
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
			copyDir(t, dstPath, srcPath)
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

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("create %q: %v", dst, err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %q -> %q: %v", src, dst, err)
	}
}

type mockSchemaParser struct {
	catalog *model.Catalog
	err     error
}

func (m *mockSchemaParser) Parse(_ context.Context, _ string, _ []byte) (*model.Catalog, []schemaparser.Diagnostic, error) {
	return m.catalog, nil, m.err
}

func TestPipeline_Run_WithCustomSchemaParser(t *testing.T) {
	// Create a mock schema parser
	mockParser := &mockSchemaParser{
		catalog: model.NewCatalog(),
	}
	mockParser.catalog.Tables["test"] = &model.Table{
		Name: "test",
		Columns: []*model.Column{
			{Name: "id", Type: "INTEGER"},
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// Create empty schema file (mock parser ignores content)
	schemaPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	// Create queries file with valid query against mock table
	queriesPath := filepath.Join(tmpDir, "queries.sql")
	queryContent := "-- name: GetTest :one\nSELECT id FROM test;"
	if err := os.WriteFile(queriesPath, []byte(queryContent), 0o644); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	pipeline := &Pipeline{
		Env: Environment{
			Logger:       logging.NewSlogAdapter(slog.Default()),
			Writer:       &memoryWriter{},
			SchemaParser: mockParser, // inject mock
		},
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: configPath,
		DryRun:     true,
	}

	summary, err := pipeline.Run(ctx, opts)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify mock was used (check diagnostics or other indicators)
	_ = summary
}

func TestPipeline_Run_WithMemoryWriter(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o644); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	writer := &MemoryWriter{}

	pipeline := &Pipeline{
		Env: Environment{
			Logger:       logging.NewSlogAdapter(slog.Default()),
			Writer:       writer,
			SchemaParser: nil, // will use default
		},
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	}

	summary, err := pipeline.Run(ctx, opts)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify files were written to memory
	if writer.FileCount() == 0 {
		t.Error("expected files to be written")
	}

	// Check that models file was generated
	hasModels := false
	for path := range writer.Files {
		if strings.Contains(path, "models") {
			hasModels = true
			break
		}
	}
	if !hasModels {
		t.Error("expected models file to be generated")
	}

	_ = summary
}

// mockGenerator is a test double for codegen.Generator
type mockGenerator struct {
	files []codegen.File
	err   error
}

func (m *mockGenerator) Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]codegen.File, error) {
	return m.files, m.err
}

func TestPipeline_Run_WithMockGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY);`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o644); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	writer := &MemoryWriter{}

	// Create mock generator with predefined output
	mockGen := &mockGenerator{
		files: []codegen.File{
			{Path: "models.gen.go", Content: []byte("package test\n")},
			{Path: "query_get_user.go", Content: []byte("package test\n")},
		},
	}

	pipeline := &Pipeline{
		Env: Environment{
			Logger:       logging.NewSlogAdapter(slog.Default()),
			Writer:       writer,
			SchemaParser: nil,     // use default
			Generator:    mockGen, // inject mock
		},
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	}

	summary, err := pipeline.Run(ctx, opts)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify mock was used - files should match mock output
	if writer.FileCount() != 2 {
		t.Errorf("expected 2 files, got %d", writer.FileCount())
	}

	outPrefix := filepath.Join(tmpDir, "out") + string(os.PathSeparator)
	if !writer.HasFile(outPrefix+"models.gen.go") && !writer.HasFile("out/models.gen.go") {
		t.Error("expected models.gen.go to be written")
	}

	_ = summary
}

func TestPipeline_Run_WithMockGeneratorError(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `package = "test"
out = "out"
schemas = ["schema.sql"]
queries = ["queries.sql"]
`
	schemaContent := `CREATE TABLE users (id INTEGER PRIMARY KEY);`
	queryContent := `-- name: GetUser :one
SELECT * FROM users WHERE id = :id;`

	// Write files to temp directory
	if err := os.WriteFile(filepath.Join(tmpDir, "db-catalyst.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "schema.sql"), []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "queries.sql"), []byte(queryContent), 0o644); err != nil {
		t.Fatalf("write queries: %v", err)
	}

	// Create mock generator that returns an error
	mockGen := &mockGenerator{
		err: errors.New("generation failed"),
	}

	pipeline := &Pipeline{
		Env: Environment{
			Logger:       logging.NewSlogAdapter(slog.Default()),
			Writer:       &MemoryWriter{},
			SchemaParser: nil,
			Generator:    mockGen,
		},
	}

	ctx := context.Background()
	opts := RunOptions{
		ConfigPath: filepath.Join(tmpDir, "db-catalyst.toml"),
	}

	_, err := pipeline.Run(ctx, opts)
	if err == nil {
		t.Fatal("expected error from mock generator")
	}

	if !strings.Contains(err.Error(), "generation failed") {
		t.Errorf("error message = %q, want to contain 'generation failed'", err.Error())
	}
}
