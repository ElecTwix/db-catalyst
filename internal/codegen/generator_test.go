package codegen

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestGeneratorProducesDeterministicOutput(t *testing.T) {
	catalog, analyses := sampleCatalogAndAnalyses()

	g := New(Options{Package: "store", EmitJSONTags: true})

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
		want, err := os.ReadFile(goldenPath)
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

	g := New(Options{Package: "store", Prepared: PreparedOptions{Enabled: true, EmitMetrics: true, ThreadSafe: true}})

	ctx := context.Background()
	files, err := g.Generate(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var prepared *File
	for i := range files {
		if files[i].Path == "prepared.go" {
			prepared = &files[i]
			break
		}
	}
	if prepared == nil {
		t.Fatalf("prepared.go not emitted")
	}

	goldenPath := filepath.Join("testdata", "golden", "prepared.go.golden")
	want, err := os.ReadFile(goldenPath)
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
