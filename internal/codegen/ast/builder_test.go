package ast

import (
	"context"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestBuilder_Build(t *testing.T) {
	ctx := context.Background()

	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER"},
					{Name: "name", Type: "TEXT"},
				},
			},
		},
	}

	analyses := []analyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "GetUser",
					SQL:     "SELECT id, name FROM users WHERE id = :id",
					Command: block.CommandOne,
				},
				Columns: []parser.Column{
					{Expr: "id", Table: "users"},
					{Expr: "name", Table: "users"},
				},
				Params: []parser.Param{
					{Name: "id", Line: 1, Column: 35},
				},
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64", Nullable: false, Table: "users"},
				{Name: "name", GoType: "string", Nullable: true, Table: "users"},
			},
			Params: []analyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false},
			},
		},
	}

	builder := New(Options{
		Package:      "test",
		EmitJSONTags: true,
	})

	files, err := builder.Build(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("expected files to be generated")
	}

	// Check that models file was generated (since columns reference "users" table)
	hasModels := false
	for _, f := range files {
		if f.Path == "models.gen.go" {
			hasModels = true
			break
		}
	}
	if !hasModels {
		t.Error("expected models.gen.go file to be generated")
	}

	// Check that querier file was generated
	hasQuerier := false
	for _, f := range files {
		if f.Path == "querier.gen.go" {
			hasQuerier = true
			break
		}
	}
	if !hasQuerier {
		t.Error("expected querier.gen.go file to be generated")
	}
}

func TestBuilder_Build_NoTableRefs(t *testing.T) {
	ctx := context.Background()

	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER"},
					{Name: "name", Type: "TEXT"},
				},
			},
		},
	}

	// No table references in columns - models file should not be generated
	analyses := []analyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "GetUser",
					SQL:     "SELECT 1 as num",
					Command: block.CommandOne,
				},
				Columns: []parser.Column{
					{Expr: "1", Alias: "num"},
				},
			},
			Columns: []analyzer.ResultColumn{
				{Name: "num", GoType: "interface{}", Nullable: true},
			},
		},
	}

	builder := New(Options{
		Package:      "test",
		EmitJSONTags: true,
	})

	files, err := builder.Build(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Models file should not be generated when no table references
	hasModels := false
	for _, f := range files {
		if f.Path == "models.gen.go" {
			hasModels = true
			break
		}
	}
	if hasModels {
		t.Error("expected models.gen.go file NOT to be generated when no table references")
	}
}

func TestTypeResolver_ResolveType(t *testing.T) {
	tr := NewTypeResolver(nil)

	tests := []struct {
		name     string
		sqlite   string
		nullable bool
		want     string
	}{
		{"integer not null", "INTEGER", false, "int32"},
		{"integer nullable", "INTEGER", true, "*int32"},
		{"text not null", "TEXT", false, "string"},
		{"text nullable", "TEXT", true, "sql.NullString"},
		{"real not null", "REAL", false, "float64"},
		{"real nullable", "REAL", true, "sql.NullFloat64"},
		{"blob not null", "BLOB", false, "[]byte"},
		{"blob nullable", "BLOB", true, "*[]byte"},
		{"bigint not null", "BIGINT", false, "int64"},
		{"smallint not null", "SMALLINT", false, "int16"},
		{"tinyint not null", "TINYINT", false, "int8"},
		{"varchar not null", "VARCHAR", false, "string"},
		{"char not null", "CHAR", false, "string"},
		{"float not null", "FLOAT", false, "float64"},
		{"double not null", "DOUBLE", false, "float64"},
		{"boolean not null", "BOOLEAN", false, "bool"},
		{"bool not null", "BOOL", false, "bool"},
		{"numeric not null", "NUMERIC", false, "float64"},
		{"decimal not null", "DECIMAL", false, "float64"},
		{"unknown type", "UNKNOWN", false, "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.ResolveType(tt.sqlite, tt.nullable)
			if got.GoType != tt.want {
				t.Errorf("ResolveType(%q, %v) = %q, want %q", tt.sqlite, tt.nullable, got.GoType, tt.want)
			}
		})
	}
}

func TestPreparedOptions(t *testing.T) {
	opts := PreparedOptions{
		Enabled:     true,
		EmitMetrics: true,
		ThreadSafe:  true,
	}

	if !opts.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if !opts.EmitMetrics {
		t.Error("Expected EmitMetrics to be true")
	}
	if !opts.ThreadSafe {
		t.Error("Expected ThreadSafe to be true")
	}
}

func TestBuilder_Options(t *testing.T) {
	opts := Options{
		Package:             "mypackage",
		EmitJSONTags:        true,
		EmitEmptySlices:     true,
		EmitPointersForNull: true,
		Prepared: PreparedOptions{
			Enabled:     true,
			EmitMetrics: true,
			ThreadSafe:  true,
		},
	}

	builder := New(opts)
	if builder.opts.Package != "mypackage" {
		t.Errorf("Package = %q, want mypackage", builder.opts.Package)
	}
	if !builder.opts.EmitJSONTags {
		t.Error("EmitJSONTags should be true")
	}
	if !builder.opts.EmitEmptySlices {
		t.Error("EmitEmptySlices should be true")
	}
	if !builder.opts.EmitPointersForNull {
		t.Error("EmitPointersForNull should be true")
	}
	if !builder.opts.Prepared.Enabled {
		t.Error("Prepared.Enabled should be true")
	}
}

func TestBuilder_EmptyAnalyses(t *testing.T) {
	ctx := context.Background()
	catalog := model.NewCatalog()

	builder := New(Options{Package: "test"})
	files, err := builder.Build(ctx, catalog, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should still generate querier file even with no analyses
	if len(files) == 0 {
		t.Error("expected at least querier.gen.go to be generated")
	}
}

func TestBuilder_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	catalog := model.NewCatalog()
	builder := New(Options{Package: "test"})

	_, err := builder.Build(ctx, catalog, nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestTypeInfo(t *testing.T) {
	info := TypeInfo{
		GoType:      "string",
		UsesSQLNull: true,
		Import:      "database/sql",
		Package:     "sql",
	}

	if info.GoType != "string" {
		t.Errorf("GoType = %q, want string", info.GoType)
	}
	if !info.UsesSQLNull {
		t.Error("UsesSQLNull should be true")
	}
	if info.Import != "database/sql" {
		t.Errorf("Import = %q, want database/sql", info.Import)
	}
}

func TestCollectImports(t *testing.T) {
	infos := []TypeInfo{
		{GoType: "string"},
		{GoType: "sql.NullInt64", Import: "database/sql", Package: "sql", UsesSQLNull: true},
	}

	imports := CollectImports(infos)
	if len(imports) != 1 {
		t.Errorf("len(imports) = %d, want 1", len(imports))
	}
	if pkg, ok := imports["database/sql"]; !ok || pkg != "sql" {
		t.Errorf("imports[database/sql] = %q, want sql", pkg)
	}
}

func TestTypeResolver_WithPointersForNull(t *testing.T) {
	tr := NewTypeResolverWithOptions(nil, true)

	// When emitPointersForNull is true, nullable types should use pointers
	got := tr.ResolveType("INTEGER", true)
	if !strings.HasPrefix(got.GoType, "*") {
		t.Errorf("ResolveType with emitPointersForNull = true should return pointer, got %q", got.GoType)
	}
}
