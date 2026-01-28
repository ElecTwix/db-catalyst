package codegen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// mockGenerator is a test double for Generator.
type mockGenerator struct {
	files []File
	err   error
}

func (m *mockGenerator) Generate(_ context.Context, _ *model.Catalog, _ []analyzer.Result) ([]File, error) {
	return m.files, m.err
}

//nolint:revive // Test uses t for interface compliance check
func TestGenerator_ImplementsInterface(t *testing.T) {
	// This test verifies at compile time that the concrete type implements Generator interface.
	_ = New(Options{})
}

func TestGenerator_WithMock(t *testing.T) {
	// Create a mock generator for testing
	mock := &mockGenerator{
		files: []File{
			{Path: "test.go", Content: []byte("package test")},
		},
	}

	ctx := context.Background()
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {Name: "users"},
		},
	}

	files, err := mock.Generate(ctx, catalog, nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("len(files) = %d, want 1", len(files))
	}
}

func TestGenerator_MockError(t *testing.T) {
	// Create a mock generator that returns an error
	mock := &mockGenerator{
		err: errors.New("generation failed"),
	}

	ctx := context.Background()
	catalog := &model.Catalog{}

	_, err := mock.Generate(ctx, catalog, nil)
	if err == nil {
		t.Fatal("expected error from mock generator")
	}

	if err.Error() != "generation failed" {
		t.Errorf("error message = %q, want 'generation failed'", err.Error())
	}
}

func TestGeneratorProducesDeterministicOutput(t *testing.T) {
	catalog, analyses := sampleCatalogAndAnalyses()
	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	g := New(Options{Package: "store", EmitJSONTags: true, EmitEmptySlices: true})

	ctx := context.Background()
	first, err := g.Generate(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	second, err := g.Generate(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("generate second: %v", err)
	}

	if diff := cmp.Diff(fileList(first), fileList(second)); diff != "" {
		t.Fatalf("file order not deterministic (-want +got):\n%s", diff)
	}

	for _, file := range first {
		goldenPath := filepath.Join("testdata", "golden", file.Path+".golden")
		if updateGolden {
			if err := os.WriteFile(goldenPath, file.Content, 0o600); err != nil {
				t.Fatalf("write golden %s: %v", goldenPath, err)
			}
			continue
		}
		want, err := os.ReadFile(filepath.Clean(goldenPath))
		if err != nil {
			t.Fatalf("read golden %s: %v", goldenPath, err)
		}
		if diff := cmp.Diff(string(want), string(file.Content)); diff != "" {
			t.Errorf("mismatch for %s (-want +got):\n%s", file.Path, diff)
		}
	}
}

func TestGeneratorPreparedQueries(t *testing.T) {
	catalog, analyses := sampleCatalogAndAnalyses()
	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	g := New(Options{Package: "store", EmitEmptySlices: true, Prepared: PreparedOptions{Enabled: true, EmitMetrics: true, ThreadSafe: true}})

	ctx := context.Background()
	files, err := g.Generate(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var prepared *File
	for i := range files {
		if files[i].Path == "prepared.gen.go" {
			prepared = &files[i]
			break
		}
	}
	if prepared == nil {
		t.Fatalf("prepared.gen.go not emitted")
	}

	goldenPath := filepath.Join("testdata", "golden", "prepared.gen.go.golden")
	if updateGolden {
		if err := os.WriteFile(goldenPath, prepared.Content, 0o600); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}
		return
	}
	want, err := os.ReadFile(filepath.Clean(goldenPath))
	if err != nil {
		t.Fatalf("read golden %s: %v\n%s", goldenPath, err, prepared.Content)
	}
	if diff := cmp.Diff(string(want), string(prepared.Content)); diff != "" {
		t.Errorf("mismatch for prepared.go (-want +got):\n%s", diff)
	}
}

func fileList(files []File) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}

func sampleCatalogAndAnalyses() (*model.Catalog, []analyzer.Result) {
	catalog := model.NewCatalog()
	catalog.Tables["users"] = &model.Table{
		Name: "users",
		Columns: []*model.Column{
			{Name: "id", Type: "INTEGER", NotNull: true},
			{Name: "email", Type: "TEXT"},
			{Name: "credits", Type: "REAL"},
		},
	}

	analyses := []analyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "GetUser",
					Command: block.CommandOne,
					SQL:     "SELECT id, email, credits FROM users WHERE id = ?",
					Doc:     "GetUser fetches a single user by identifier.",
				},
				Verb: parser.VerbSelect,
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", Table: "users", GoType: "int64", Nullable: false},
				{Name: "email", Table: "users", GoType: "string", Nullable: true},
				{Name: "credits", Table: "users", GoType: "float64", Nullable: true},
			},
			Params: []analyzer.ResultParam{
				{Name: "id", Style: parser.ParamStylePositional, GoType: "int64", Nullable: false},
			},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "ListUsers",
					Command: block.CommandMany,
					SQL:     "SELECT id, email FROM users ORDER BY email",
				},
				Verb: parser.VerbSelect,
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", Table: "users", GoType: "int64", Nullable: false},
				{Name: "email", Table: "users", GoType: "string", Nullable: true},
			},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "ListUsersByIDs",
					Command: block.CommandMany,
					SQL:     "SELECT id, email FROM users WHERE id IN (?1, ?2, ?3)",
				},
				Verb: parser.VerbSelect,
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", Table: "users", GoType: "int64", Nullable: false},
				{Name: "email", Table: "users", GoType: "string", Nullable: true},
			},
			Params: []analyzer.ResultParam{
				{
					Name:          "ids",
					Style:         parser.ParamStylePositional,
					GoType:        "int64",
					Nullable:      false,
					IsVariadic:    true,
					VariadicCount: 3,
				},
			},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "CreateUser",
					Command: block.CommandExec,
					SQL:     "INSERT INTO users (email, credits) VALUES (?, ?)",
				},
				Verb: parser.VerbInsert,
			},
			Params: []analyzer.ResultParam{
				{Name: "email", Style: parser.ParamStylePositional, GoType: "string", Nullable: true},
				{Name: "credits", Style: parser.ParamStylePositional, GoType: "float64", Nullable: true},
			},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "UpdateUserCredits",
					Command: block.CommandExec,
					SQL:     "UPDATE users SET credits = ? WHERE id = ?",
				},
				Verb: parser.VerbUpdate,
			},
			Params: []analyzer.ResultParam{
				{Name: "credits", Style: parser.ParamStylePositional, GoType: "float64", Nullable: true},
				{Name: "id", Style: parser.ParamStylePositional, GoType: "int64", Nullable: false},
			},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "DeleteUser",
					Command: block.CommandExecResult,
					SQL:     "DELETE FROM users WHERE id = ?",
				},
				Verb: parser.VerbDelete,
			},
			Params: []analyzer.ResultParam{
				{Name: "id", Style: parser.ParamStylePositional, GoType: "int64", Nullable: false},
			},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "SummarizeCredits",
					Command: block.CommandOne,
					SQL: `WITH RECURSIVE credit_totals AS (
    SELECT id, credits FROM users
    UNION ALL
    SELECT u.id, u.credits FROM users u JOIN credit_totals c ON u.id > c.id
)
SELECT COUNT(*) AS total_users,
       SUM(credit_totals.credits) AS sum_credits,
       AVG(credit_totals.credits) AS avg_credit
FROM credit_totals;`,
					Doc: "SummarizeCredits aggregates user credits across a recursive rollup.",
				},
				Verb: parser.VerbSelect,
			},
			Columns: []analyzer.ResultColumn{
				{Name: "total_users", GoType: "int64", Nullable: false},
				{Name: "sum_credits", GoType: "float64", Nullable: true},
				{Name: "avg_credit", GoType: "float64", Nullable: true},
			},
		},
	}

	return catalog, analyses
}
