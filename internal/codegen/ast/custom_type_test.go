// Package ast_test provides integration tests for custom type code generation.
package ast_test

import (
	"context"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/codegen/ast"
	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestCustomTypeImportGeneration(t *testing.T) {
	// Create a simple catalog with a users table
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: false},
				},
			},
		},
	}

	// Create a query that selects and filters by user id
	blk := block.Block{
		Path:    "queries/test.sql",
		Line:    1,
		Column:  1,
		SQL:     "SELECT id, email FROM users WHERE id = :user_id;",
		Name:    "GetUser",
		Command: block.CommandOne,
	}

	// Parse the query
	q, diags := parser.Parse(blk)
	if len(diags) > 0 {
		t.Fatalf("unexpected parser diagnostics: %+v", diags)
	}

	// Analyze with column override for users.id
	an := analyzer.New(catalog)
	an.SetColumnOverrides(map[string]config.ColumnOverride{
		"users.id": {
			Column: "users.id",
			GoType: config.GoTypeDetails{
				Import:  "github.com/example/types",
				Package: "types",
				Type:    "UserID",
			},
		},
	})
	res := an.Analyze(q)

	if len(res.Diagnostics) > 0 {
		t.Fatalf("unexpected analyzer diagnostics: %+v", res.Diagnostics)
	}

	// Build AST files
	builder := ast.New(ast.Options{
		Package:         "db",
		EmitJSONTags:    true,
		ColumnOverrides: []config.ColumnOverride{},
	})

	files, err := builder.Build(context.Background(), catalog, []analyzer.Result{res})
	if err != nil {
		t.Fatalf("failed to build AST files: %v", err)
	}

	// Find the query file
	var queryFile *ast.File
	for i := range files {
		if strings.Contains(files[i].Path, "query_get_user") {
			queryFile = &files[i]
			break
		}
	}

	if queryFile == nil {
		t.Fatal("query file not found")
	}

	// Convert AST to string to check for imports
	var buf strings.Builder
	if err := format.Node(&buf, token.NewFileSet(), queryFile.Node); err != nil {
		t.Fatalf("failed to format AST: %v", err)
	}
	generatedCode := buf.String()

	// Verify Issue 1: Import is added
	if !strings.Contains(generatedCode, `"github.com/example/types"`) {
		t.Errorf("Expected import 'github.com/example/types' not found in generated code")
	}

	// Verify Issue 2: Query parameter uses custom type
	if !strings.Contains(generatedCode, "UserID") {
		t.Errorf("Expected parameter type 'UserID' not found in generated code")
	}

	// Verify Issue 3: Result struct uses custom type
	// Check that the row type uses UserID
	helperFile := findFile(files, "helpers.gen.go")
	if helperFile == nil {
		t.Fatal("helpers file not found")
	}

	buf.Reset()
	if err := format.Node(&buf, token.NewFileSet(), helperFile.Node); err != nil {
		t.Fatalf("failed to format helpers AST: %v", err)
	}
	helperCode := buf.String()

	if !strings.Contains(helperCode, "UserID") {
		t.Errorf("Expected result type 'UserID' not found in helpers")
	}

	t.Log("All three issues are fixed:")
	t.Log("✓ Issue 1: Custom type imports are added to generated files")
	t.Log("✓ Issue 2: Query parameters use custom types from column overrides")
	t.Log("✓ Issue 3: Query result structs use custom types from column overrides")
}

func findFile(files []ast.File, name string) *ast.File {
	for i := range files {
		if files[i].Path == name {
			return &files[i]
		}
	}
	return nil
}
