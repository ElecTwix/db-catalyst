package ast

import (
	"context"
	"go/ast"
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
				{Name: "num", GoType: "any", Nullable: true},
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
		{"unknown type", "UNKNOWN", false, "any"},
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

func TestBuildPreparedFile(t *testing.T) {
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

	tests := []struct {
		name         string
		preparedOpts PreparedOptions
		wantFile     bool
		wantPath     string
	}{
		{
			name:         "basic prepared file",
			preparedOpts: PreparedOptions{Enabled: true},
			wantFile:     true,
			wantPath:     "prepared.gen.go",
		},
		{
			name:         "prepared with metrics",
			preparedOpts: PreparedOptions{Enabled: true, EmitMetrics: true},
			wantFile:     true,
			wantPath:     "prepared.gen.go",
		},
		{
			name:         "prepared with thread safe",
			preparedOpts: PreparedOptions{Enabled: true, ThreadSafe: true},
			wantFile:     true,
			wantPath:     "prepared.gen.go",
		},
		{
			name:         "prepared with all options",
			preparedOpts: PreparedOptions{Enabled: true, EmitMetrics: true, ThreadSafe: true},
			wantFile:     true,
			wantPath:     "prepared.gen.go",
		},
		{
			name:         "disabled prepared",
			preparedOpts: PreparedOptions{Enabled: false},
			wantFile:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New(Options{
				Package:      "test",
				EmitJSONTags: true,
				Prepared:     tt.preparedOpts,
			})

			files, err := builder.Build(ctx, catalog, analyses)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			var found bool
			for _, f := range files {
				if f.Path == tt.wantPath {
					found = true
					if f.Node == nil {
						t.Error("expected Node to be set")
					}
					if len(f.Raw) == 0 {
						t.Error("expected Raw to be set")
					}
					break
				}
			}

			if tt.wantFile && !found {
				t.Errorf("expected file %q to be generated", tt.wantPath)
			}
			if !tt.wantFile && found {
				t.Errorf("expected file %q NOT to be generated", tt.wantPath)
			}
		})
	}
}

func TestBuildPreparedFile_MultipleCommands(t *testing.T) {
	ctx := context.Background()

	catalog := model.NewCatalog()

	analyses := []analyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "CreateUser",
					SQL:     "INSERT INTO users (name) VALUES (:name)",
					Command: block.CommandExec,
				},
				Columns: []parser.Column{},
				Params:  []parser.Param{{Name: "name", Line: 1, Column: 32}},
			},
			Columns: []analyzer.ResultColumn{},
			Params:  []analyzer.ResultParam{{Name: "name", GoType: "string", Nullable: false}},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "CreateUserResult",
					SQL:     "INSERT INTO users (name) VALUES (:name)",
					Command: block.CommandExecResult,
				},
				Columns: []parser.Column{},
				Params:  []parser.Param{{Name: "name", Line: 1, Column: 32}},
			},
			Columns: []analyzer.ResultColumn{},
			Params:  []analyzer.ResultParam{{Name: "name", GoType: "string", Nullable: false}},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "GetUsers",
					SQL:     "SELECT id, name FROM users",
					Command: block.CommandMany,
				},
				Columns: []parser.Column{
					{Expr: "id"},
					{Expr: "name"},
				},
				Params: []parser.Param{},
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64", Nullable: false},
				{Name: "name", GoType: "string", Nullable: true},
			},
			Params: []analyzer.ResultParam{},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "GetUser",
					SQL:     "SELECT id, name FROM users WHERE id = :id",
					Command: block.CommandOne,
				},
				Columns: []parser.Column{
					{Expr: "id"},
					{Expr: "name"},
				},
				Params: []parser.Param{{Name: "id", Line: 1, Column: 35}},
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64", Nullable: false},
				{Name: "name", GoType: "string", Nullable: true},
			},
			Params: []analyzer.ResultParam{{Name: "id", GoType: "int64", Nullable: false}},
		},
	}

	builder := New(Options{
		Package:      "test",
		EmitJSONTags: true,
		Prepared:     PreparedOptions{Enabled: true, EmitMetrics: true, ThreadSafe: true},
	})

	files, err := builder.Build(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var preparedFile *File
	for i := range files {
		if files[i].Path == "prepared.gen.go" {
			preparedFile = &files[i]
			break
		}
	}

	if preparedFile == nil {
		t.Fatal("expected prepared.gen.go to be generated")
	}

	// Check that the file contains expected content
	content := string(preparedFile.Raw)
	if !strings.Contains(content, "PrepareDB") {
		t.Error("expected PrepareDB interface to be generated")
	}
	if !strings.Contains(content, "PreparedQueries") {
		t.Error("expected PreparedQueries struct to be generated")
	}
	if !strings.Contains(content, "PreparedMetricsRecorder") {
		t.Error("expected PreparedMetricsRecorder interface to be generated")
	}
	if !strings.Contains(content, "sync.Mutex") {
		t.Error("expected sync.Mutex to be generated for thread-safe mode")
	}
	if !strings.Contains(content, "sync.Once") {
		t.Error("expected sync.Once to be generated for thread-safe mode")
	}
}

func TestBuildPreparedFile_VariadicParams(t *testing.T) {
	ctx := context.Background()

	catalog := model.NewCatalog()

	analyses := []analyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "GetUsersByIDs",
					SQL:     "SELECT id, name FROM users WHERE id IN (:ids)",
					Command: block.CommandMany,
				},
				Columns: []parser.Column{
					{Expr: "id"},
					{Expr: "name"},
				},
				Params: []parser.Param{{Name: "ids", Line: 1, Column: 40, IsVariadic: true, VariadicCount: 2}},
			},
			Columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64", Nullable: false},
				{Name: "name", GoType: "string", Nullable: true},
			},
			Params: []analyzer.ResultParam{
				{Name: "ids", GoType: "int64", Nullable: false, IsVariadic: true, VariadicCount: 2},
			},
		},
	}

	builder := New(Options{
		Package:      "test",
		EmitJSONTags: true,
		Prepared:     PreparedOptions{Enabled: true, EmitMetrics: true},
	})

	files, err := builder.Build(ctx, catalog, analyses)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var preparedFile *File
	for i := range files {
		if files[i].Path == "prepared.gen.go" {
			preparedFile = &files[i]
			break
		}
	}

	if preparedFile == nil {
		t.Fatal("expected prepared.gen.go to be generated")
	}

	// Check that variadic params are handled
	content := string(preparedFile.Raw)
	if !strings.Contains(content, "GetUsersByIDs") {
		t.Error("expected GetUsersByIDs method to be generated")
	}
}

func TestBuildPreparedFile_EmptyQueries(t *testing.T) {
	ctx := context.Background()

	catalog := model.NewCatalog()

	builder := New(Options{
		Package:  "test",
		Prepared: PreparedOptions{Enabled: true},
	})

	files, err := builder.Build(ctx, catalog, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var found bool
	for _, f := range files {
		if f.Path == "prepared.gen.go" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected prepared.gen.go to be generated even with no queries")
	}
}

func TestBuildParams_EdgeCases(t *testing.T) {
	b := New(Options{
		Package:             "test",
		EmitPointersForNull: true,
	})

	tests := []struct {
		name    string
		params  []analyzer.ResultParam
		wantErr bool
	}{
		{
			name:    "empty params",
			params:  []analyzer.ResultParam{},
			wantErr: false,
		},
		{
			name: "single param",
			params: []analyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false},
			},
			wantErr: false,
		},
		{
			name: "duplicate param names",
			params: []analyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false},
				{Name: "id", GoType: "string", Nullable: false},
			},
			wantErr: false,
		},
		{
			name: "variadic param",
			params: []analyzer.ResultParam{
				{Name: "ids", GoType: "int64", Nullable: false, IsVariadic: true, VariadicCount: 3},
			},
			wantErr: false,
		},
		{
			name: "dynamic slice param",
			params: []analyzer.ResultParam{
				{Name: "ids", GoType: "int64", Nullable: false, IsVariadic: true, VariadicCount: 0},
			},
			wantErr: false,
		},
		{
			name: "param named ctx",
			params: []analyzer.ResultParam{
				{Name: "ctx", GoType: "string", Nullable: false},
			},
			wantErr: false,
		},
		{
			name: "empty param name",
			params: []analyzer.ResultParam{
				{Name: "", GoType: "int64", Nullable: false},
			},
			wantErr: false,
		},
		{
			name: "multiple variadic params with same base name",
			params: []analyzer.ResultParam{
				{Name: "id", GoType: "int64", Nullable: false, IsVariadic: true, VariadicCount: 2},
				{Name: "id", GoType: "int64", Nullable: false, IsVariadic: true, VariadicCount: 2},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(tt.params) > 0 && len(got) != len(tt.params) {
				t.Errorf("buildParams() returned %d specs, want %d", len(got), len(tt.params))
			}
		})
	}
}

func TestBuildParams_WithCustomTypeResolver(t *testing.T) {
	tr := NewTypeResolver(nil)
	b := New(Options{
		Package:      "test",
		TypeResolver: tr,
	})

	params := []analyzer.ResultParam{
		{Name: "id", GoType: "int64", Nullable: false},
	}

	got, err := b.buildParams(params)
	if err != nil {
		t.Fatalf("buildParams() error = %v", err)
	}

	if len(got) != 1 {
		t.Errorf("buildParams() returned %d specs, want 1", len(got))
	}

	// When using a TypeResolver, the GoType "int64" is already a Go primitive,
	// so it should remain as int64
	if got[0].goType != "int64" {
		t.Errorf("buildParams() goType = %q, want int64", got[0].goType)
	}
}

func TestUniqueName(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		used     map[string]int
		want     string
		wantErr  bool
		wantUsed map[string]int
	}{
		{
			name:     "unused name",
			base:     "foo",
			used:     map[string]int{},
			want:     "foo",
			wantUsed: map[string]int{"foo": 1},
		},
		{
			name:     "already used once",
			base:     "foo",
			used:     map[string]int{"foo": 1},
			want:     "foo2",
			wantUsed: map[string]int{"foo": 2, "foo2": 1},
		},
		{
			name:     "already used twice",
			base:     "foo",
			used:     map[string]int{"foo": 2, "foo2": 1},
			want:     "foo3",
			wantUsed: map[string]int{"foo": 3, "foo2": 1, "foo3": 1},
		},
		{
			name:     "empty base",
			base:     "",
			used:     map[string]int{},
			want:     "value",
			wantUsed: map[string]int{"value": 1},
		},
		{
			name:    "nil map",
			base:    "foo",
			used:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UniqueName(tt.base, tt.used)
			if (err != nil) != tt.wantErr {
				t.Errorf("UniqueName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UniqueName() = %q, want %q", got, tt.want)
			}
			if !tt.wantErr {
				for k, v := range tt.wantUsed {
					if tt.used[k] != v {
						t.Errorf("used[%q] = %d, want %d", k, tt.used[k], v)
					}
				}
			}
		})
	}
}

func TestBuildQueries_EdgeCases(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name     string
		analyses []analyzer.Result
		wantErr  bool
	}{
		{
			name:     "empty analyses",
			analyses: []analyzer.Result{},
			wantErr:  false,
		},
		{
			name: "query with empty name",
			analyses: []analyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{
							Name:    "",
							SQL:     "SELECT 1",
							Command: block.CommandOne,
						},
						Columns: []parser.Column{{Expr: "1"}},
						Params:  []parser.Param{},
					},
					Columns: []analyzer.ResultColumn{{Name: "column_1", GoType: "int64"}},
					Params:  []analyzer.ResultParam{},
				},
			},
			wantErr: false,
		},
		{
			name: "query with doc comment",
			analyses: []analyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{
							Name:    "GetUser",
							SQL:     "SELECT id FROM users",
							Command: block.CommandOne,
							Doc:     "GetUser retrieves a user by ID",
						},
						Columns: []parser.Column{{Expr: "id"}},
						Params:  []parser.Param{},
					},
					Columns: []analyzer.ResultColumn{{Name: "id", GoType: "int64"}},
					Params:  []analyzer.ResultParam{},
				},
			},
			wantErr: false,
		},
		{
			name: "dynamic slice with invalid offsets",
			analyses: []analyzer.Result{
				{
					Query: parser.Query{
						Block: block.Block{
							Name:    "GetUsers",
							SQL:     "SELECT id FROM users WHERE id IN (:ids)",
							Command: block.CommandMany,
						},
						Columns: []parser.Column{{Expr: "id"}},
						Params:  []parser.Param{{Name: "ids", Line: 1, Column: 35, StartOffset: -1, EndOffset: 10}},
					},
					Columns: []analyzer.ResultColumn{{Name: "id", GoType: "int64"}},
					Params:  []analyzer.ResultParam{{Name: "ids", GoType: "int64", IsVariadic: true, VariadicCount: 0}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildQueries(tt.analyses)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQueries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != len(tt.analyses) {
				t.Errorf("buildQueries() returned %d queries, want %d", len(got), len(tt.analyses))
			}
		})
	}
}

func TestBuildQueries_AllCommandTypes(t *testing.T) {
	b := New(Options{Package: "test"})

	analyses := []analyzer.Result{
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "ExecQuery",
					SQL:     "DELETE FROM users WHERE id = :id",
					Command: block.CommandExec,
				},
				Columns: []parser.Column{},
				Params:  []parser.Param{{Name: "id", Line: 1, Column: 32}},
			},
			Columns: []analyzer.ResultColumn{},
			Params:  []analyzer.ResultParam{{Name: "id", GoType: "int64"}},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "ExecResultQuery",
					SQL:     "INSERT INTO users (name) VALUES (:name)",
					Command: block.CommandExecResult,
				},
				Columns: []parser.Column{},
				Params:  []parser.Param{{Name: "name", Line: 1, Column: 37}},
			},
			Columns: []analyzer.ResultColumn{},
			Params:  []analyzer.ResultParam{{Name: "name", GoType: "string"}},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "OneQuery",
					SQL:     "SELECT id FROM users WHERE id = :id",
					Command: block.CommandOne,
				},
				Columns: []parser.Column{{Expr: "id"}},
				Params:  []parser.Param{{Name: "id", Line: 1, Column: 35}},
			},
			Columns: []analyzer.ResultColumn{{Name: "id", GoType: "int64"}},
			Params:  []analyzer.ResultParam{{Name: "id", GoType: "int64"}},
		},
		{
			Query: parser.Query{
				Block: block.Block{
					Name:    "ManyQuery",
					SQL:     "SELECT id FROM users",
					Command: block.CommandMany,
				},
				Columns: []parser.Column{{Expr: "id"}},
				Params:  []parser.Param{},
			},
			Columns: []analyzer.ResultColumn{{Name: "id", GoType: "int64"}},
			Params:  []analyzer.ResultParam{},
		},
	}

	queries, err := b.buildQueries(analyses)
	if err != nil {
		t.Fatalf("buildQueries() error = %v", err)
	}

	if len(queries) != 4 {
		t.Errorf("buildQueries() returned %d queries, want 4", len(queries))
	}

	// Check return types
	expectedReturnTypes := map[string]string{
		"ExecQuery":       "sql.Result",
		"ExecResultQuery": "QueryResult",
		"OneQuery":        "OneQueryRow",
		"ManyQuery":       "[]ManyQueryRow",
	}

	for _, q := range queries {
		want, ok := expectedReturnTypes[q.methodName]
		if !ok {
			t.Errorf("unexpected query: %s", q.methodName)
			continue
		}
		if q.returnType != want {
			t.Errorf("%s returnType = %q, want %q", q.methodName, q.returnType, want)
		}
	}
}

func TestBuildQueryFunc(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name    string
		query   queryInfo
		wantErr bool
	}{
		{
			name: "simple exec query",
			query: queryInfo{
				methodName: "CreateUser",
				constName:  "queryCreateUser",
				sqlLiteral: "INSERT INTO users (name) VALUES (?)",
				command:    block.CommandExec,
				returnType: "sql.Result",
				returnZero: "nil",
				params: []paramSpec{
					{name: "name", goType: "string", argExpr: "name"},
				},
				args: []string{"name"},
			},
			wantErr: false,
		},
		{
			name: "query with dynamic slice",
			query: queryInfo{
				methodName: "GetUsersByIDs",
				constName:  "queryGetUsersByIDs",
				sqlLiteral: "SELECT id FROM users WHERE id IN (/*SLICE:ids*/)",
				command:    block.CommandMany,
				returnType: "[]GetUsersByIDsRow",
				returnZero: "nil",
				rowType:    "GetUsersByIDsRow",
				helper: &helperSpec{
					rowTypeName: "GetUsersByIDsRow",
					funcName:    "scanGetUsersByIDsRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
				params: []paramSpec{
					{name: "ids", goType: "int64", isDynamicSlice: true, marker: "/*SLICE:ids*/", argExpr: "ids"},
				},
				args: []string{"ids"},
			},
			wantErr: false,
		},
		{
			name: "query with variadic params",
			query: queryInfo{
				methodName: "GetUsersByIDs",
				constName:  "queryGetUsersByIDs",
				sqlLiteral: "SELECT id FROM users WHERE id IN (?)",
				command:    block.CommandMany,
				returnType: "[]GetUsersByIDsRow",
				returnZero: "nil",
				rowType:    "GetUsersByIDsRow",
				helper: &helperSpec{
					rowTypeName: "GetUsersByIDsRow",
					funcName:    "scanGetUsersByIDsRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
				params: []paramSpec{
					{name: "ids", goType: "int64", variadic: true, sliceName: "idsArgs", argExpr: "idsArgs..."},
				},
				args: []string{"idsArgs..."},
			},
			wantErr: false,
		},
		{
			name: "exec result query",
			query: queryInfo{
				methodName: "CreateUser",
				constName:  "queryCreateUser",
				sqlLiteral: "INSERT INTO users (name) VALUES (?)",
				command:    block.CommandExecResult,
				returnType: "QueryResult",
				returnZero: "QueryResult{}",
				params: []paramSpec{
					{name: "name", goType: "string", argExpr: "name"},
				},
				args: []string{"name"},
			},
			wantErr: false,
		},
		{
			name: "one query",
			query: queryInfo{
				methodName: "GetUser",
				constName:  "queryGetUser",
				sqlLiteral: "SELECT id FROM users WHERE id = ?",
				command:    block.CommandOne,
				returnType: "GetUserRow",
				returnZero: "GetUserRow{}",
				rowType:    "GetUserRow",
				helper: &helperSpec{
					rowTypeName: "GetUserRow",
					funcName:    "scanGetUserRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
				params: []paramSpec{
					{name: "id", goType: "int64", argExpr: "id"},
				},
				args: []string{"id"},
			},
			wantErr: false,
		},
		{
			name: "many query with empty slices",
			query: queryInfo{
				methodName: "GetUsers",
				constName:  "queryGetUsers",
				sqlLiteral: "SELECT id FROM users",
				command:    block.CommandMany,
				returnType: "[]GetUsersRow",
				returnZero: "nil",
				rowType:    "GetUsersRow",
				helper: &helperSpec{
					rowTypeName: "GetUsersRow",
					funcName:    "scanGetUsersRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
				params: []paramSpec{},
				args:   []string{},
			},
			wantErr: false,
		},
		{
			name: "query with doc comment",
			query: queryInfo{
				methodName: "GetUser",
				constName:  "queryGetUser",
				sqlLiteral: "SELECT id FROM users",
				command:    block.CommandOne,
				returnType: "GetUserRow",
				returnZero: "GetUserRow{}",
				rowType:    "GetUserRow",
				docComment: "GetUser retrieves a user by ID",
				helper: &helperSpec{
					rowTypeName: "GetUserRow",
					funcName:    "scanGetUserRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
				params: []paramSpec{},
				args:   []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildQueryFunc(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQueryFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("buildQueryFunc() returned nil")
			}
		})
	}
}

func TestBuildQueryFunc_WithEmitEmptySlices(t *testing.T) {
	b := New(Options{
		Package:         "test",
		EmitEmptySlices: true,
	})

	query := queryInfo{
		methodName: "GetUsers",
		constName:  "queryGetUsers",
		sqlLiteral: "SELECT id FROM users",
		command:    block.CommandMany,
		returnType: "[]GetUsersRow",
		returnZero: "nil",
		rowType:    "GetUsersRow",
		helper: &helperSpec{
			rowTypeName: "GetUsersRow",
			funcName:    "scanGetUsersRow",
			fields:      []helperField{{name: "ID", goType: "int64"}},
		},
		params: []paramSpec{},
		args:   []string{},
	}

	got, err := b.buildQueryFunc(query)
	if err != nil {
		t.Fatalf("buildQueryFunc() error = %v", err)
	}
	if got == nil {
		t.Error("buildQueryFunc() returned nil")
	}
}

func TestBuildHelper(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name    string
		method  string
		columns []analyzer.ResultColumn
		wantErr bool
	}{
		{
			name:    "empty columns",
			method:  "GetUser",
			columns: []analyzer.ResultColumn{},
			wantErr: false,
		},
		{
			name:   "single column",
			method: "GetUser",
			columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64", Nullable: false},
			},
			wantErr: false,
		},
		{
			name:   "duplicate column names",
			method: "GetUser",
			columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64"},
				{Name: "id", GoType: "string"},
			},
			wantErr: false,
		},
		{
			name:   "empty column name",
			method: "GetUser",
			columns: []analyzer.ResultColumn{
				{Name: "", GoType: "int64"},
			},
			wantErr: false,
		},
		{
			name:   "multiple columns",
			method: "GetUser",
			columns: []analyzer.ResultColumn{
				{Name: "id", GoType: "int64"},
				{Name: "name", GoType: "string"},
				{Name: "email", GoType: "string", Nullable: true},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildHelper(tt.method, tt.columns)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildHelper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("buildHelper() returned nil")
					return
				}
				if got.rowTypeName != tt.method+"Row" {
					t.Errorf("rowTypeName = %q, want %q", got.rowTypeName, tt.method+"Row")
				}
				if len(got.fields) != len(tt.columns) {
					t.Errorf("fields length = %d, want %d", len(got.fields), len(tt.columns))
				}
			}
		})
	}
}

func TestBuildHelper_WithCustomTypeResolver(t *testing.T) {
	tr := NewTypeResolver(nil)
	b := New(Options{
		Package:      "test",
		TypeResolver: tr,
	})

	columns := []analyzer.ResultColumn{
		{Name: "id", GoType: "int64", Nullable: false},
	}

	got, err := b.buildHelper("GetUser", columns)
	if err != nil {
		t.Fatalf("buildHelper() error = %v", err)
	}
	if got == nil {
		t.Fatal("buildHelper() returned nil")
	}
	if len(got.fields) != 1 {
		t.Errorf("fields length = %d, want 1", len(got.fields))
	}
}

func TestBuildTableModel(t *testing.T) {
	b := New(Options{
		Package:      "test",
		EmitJSONTags: true,
	})

	tests := []struct {
		name    string
		table   *model.Table
		wantErr bool
	}{
		{
			name: "simple table",
			table: &model.Table{
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "name", Type: "TEXT"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty table name",
			table: &model.Table{
				Name: "",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER"},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate column names",
			table: &model.Table{
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER"},
					{Name: "id", Type: "TEXT"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty column name",
			table: &model.Table{
				Name: "users",
				Columns: []*model.Column{
					{Name: "", Type: "INTEGER"},
				},
			},
			wantErr: false,
		},
		{
			name: "all sqlite types",
			table: &model.Table{
				Name: "test",
				Columns: []*model.Column{
					{Name: "int_col", Type: "INTEGER"},
					{Name: "bigint_col", Type: "BIGINT"},
					{Name: "smallint_col", Type: "SMALLINT"},
					{Name: "tinyint_col", Type: "TINYINT"},
					{Name: "text_col", Type: "TEXT"},
					{Name: "varchar_col", Type: "VARCHAR"},
					{Name: "char_col", Type: "CHAR"},
					{Name: "blob_col", Type: "BLOB"},
					{Name: "real_col", Type: "REAL"},
					{Name: "float_col", Type: "FLOAT"},
					{Name: "double_col", Type: "DOUBLE"},
					{Name: "bool_col", Type: "BOOLEAN"},
					{Name: "numeric_col", Type: "NUMERIC"},
					{Name: "decimal_col", Type: "DECIMAL"},
					{Name: "unknown_col", Type: "UNKNOWN"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildTableModel(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildTableModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("buildTableModel() returned nil")
					return
				}
				if len(got.fields) != len(tt.table.Columns) {
					t.Errorf("fields length = %d, want %d", len(got.fields), len(tt.table.Columns))
				}
			}
		})
	}
}

func TestBuildTableModel_WithCustomTypeResolver(t *testing.T) {
	tr := NewTypeResolver(nil)
	b := New(Options{
		Package:      "test",
		TypeResolver: tr,
	})

	table := &model.Table{
		Name: "users",
		Columns: []*model.Column{
			{Name: "id", Type: "INTEGER", NotNull: true},
		},
	}

	got, err := b.buildTableModel(table)
	if err != nil {
		t.Fatalf("buildTableModel() error = %v", err)
	}
	if got == nil {
		t.Fatal("buildTableModel() returned nil")
	}
	if len(got.fields) != 1 {
		t.Errorf("fields length = %d, want 1", len(got.fields))
	}
}

func TestCollectTableModels(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name     string
		catalog  *model.Catalog
		analyses []analyzer.Result
		wantLen  int
	}{
		{
			name:     "nil catalog",
			catalog:  nil,
			analyses: []analyzer.Result{},
			wantLen:  0,
		},
		{
			name: "no table references",
			catalog: &model.Catalog{
				Tables: map[string]*model.Table{
					"users": {Name: "users", Columns: []*model.Column{{Name: "id", Type: "INTEGER"}}},
				},
			},
			analyses: []analyzer.Result{
				{
					Columns: []analyzer.ResultColumn{
						{Name: "num", GoType: "int64"},
					},
				},
			},
			wantLen: 0,
		},
		{
			name: "case insensitive table match",
			catalog: &model.Catalog{
				Tables: map[string]*model.Table{
					"Users": {Name: "Users", Columns: []*model.Column{{Name: "id", Type: "INTEGER"}}},
				},
			},
			analyses: []analyzer.Result{
				{
					Columns: []analyzer.ResultColumn{
						{Name: "id", GoType: "int64", Table: "users"},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "table not in catalog",
			catalog: &model.Catalog{
				Tables: map[string]*model.Table{
					"users": {Name: "users", Columns: []*model.Column{{Name: "id", Type: "INTEGER"}}},
				},
			},
			analyses: []analyzer.Result{
				{
					Columns: []analyzer.ResultColumn{
						{Name: "id", GoType: "int64", Table: "nonexistent"},
					},
				},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.collectTableModels(tt.catalog, tt.analyses)
			if err != nil {
				t.Errorf("collectTableModels() error = %v", err)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("collectTableModels() returned %d models, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestBuildHelpersFile(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name    string
		helpers []*helperSpec
		wantErr bool
	}{
		{
			name:    "empty helpers",
			helpers: []*helperSpec{},
			wantErr: false,
		},
		{
			name: "single helper",
			helpers: []*helperSpec{
				{
					rowTypeName: "GetUserRow",
					funcName:    "scanGetUserRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple helpers",
			helpers: []*helperSpec{
				{
					rowTypeName: "GetUserRow",
					funcName:    "scanGetUserRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
				{
					rowTypeName: "GetUsersRow",
					funcName:    "scanGetUsersRow",
					fields:      []helperField{{name: "ID", goType: "int64"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildHelpersFile("test", tt.helpers)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildHelpersFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("buildHelpersFile() returned nil")
			}
		})
	}
}

func TestBuildQueryFiles(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name    string
		queries []queryInfo
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty queries",
			queries: []queryInfo{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "single exec query",
			queries: []queryInfo{
				{
					methodName: "DeleteUser",
					constName:  "queryDeleteUser",
					fileName:   "query_delete_user.go",
					sqlLiteral: "DELETE FROM users WHERE id = ?",
					command:    block.CommandExec,
					returnType: "sql.Result",
					params:     []paramSpec{{name: "id", goType: "int64", argExpr: "id"}},
					args:       []string{"id"},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "multiple queries",
			queries: []queryInfo{
				{
					methodName: "DeleteUser",
					constName:  "queryDeleteUser",
					fileName:   "query_delete_user.go",
					sqlLiteral: "DELETE FROM users WHERE id = ?",
					command:    block.CommandExec,
					returnType: "sql.Result",
					params:     []paramSpec{{name: "id", goType: "int64", argExpr: "id"}},
					args:       []string{"id"},
				},
				{
					methodName: "CreateUser",
					constName:  "queryCreateUser",
					fileName:   "query_create_user.go",
					sqlLiteral: "INSERT INTO users (name) VALUES (?)",
					command:    block.CommandExec,
					returnType: "sql.Result",
					params:     []paramSpec{{name: "name", goType: "string", argExpr: "name"}},
					args:       []string{"name"},
				},
			},
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildQueryFiles("test", tt.queries)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQueryFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("buildQueryFiles() returned %d files, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestBuildModelsFile(t *testing.T) {
	b := New(Options{
		Package:      "test",
		EmitJSONTags: true,
	})

	tests := []struct {
		name    string
		models  []*tableModel
		wantErr bool
	}{
		{
			name:    "empty models",
			models:  []*tableModel{},
			wantErr: false,
		},
		{
			name: "single model",
			models: []*tableModel{
				{
					tableName: "users",
					typeName:  "User",
					fields: []modelField{
						{columnName: "id", fieldName: "ID", goType: "int64", jsonTag: "`json:\"id\"`"},
					},
					needsSQL: false,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple models",
			models: []*tableModel{
				{
					tableName: "users",
					typeName:  "User",
					fields: []modelField{
						{columnName: "id", fieldName: "ID", goType: "int64"},
					},
				},
				{
					tableName: "posts",
					typeName:  "Post",
					fields: []modelField{
						{columnName: "id", fieldName: "ID", goType: "int64"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildModelsFile("test", tt.models)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildModelsFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("buildModelsFile() returned nil")
			}
		})
	}
}

func TestBuildModelsFile_WithCustomTypeResolver(t *testing.T) {
	tr := NewTypeResolver(nil)
	b := New(Options{
		Package:      "test",
		TypeResolver: tr,
	})

	models := []*tableModel{
		{
			tableName: "users",
			typeName:  "User",
			fields: []modelField{
				{columnName: "id", fieldName: "ID", goType: "int64"},
			},
		},
	}

	got, err := b.buildModelsFile("test", models)
	if err != nil {
		t.Fatalf("buildModelsFile() error = %v", err)
	}
	if got == nil {
		t.Fatal("buildModelsFile() returned nil")
	}
}

func TestStringLiteral(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "simple string",
			value: "SELECT * FROM users",
			want:  "`SELECT * FROM users`",
		},
		{
			name:  "string with backticks",
			value: "SELECT `name` FROM users",
			want:  "\"SELECT `name` FROM users\"",
		},
		{
			name:  "empty string",
			value: "",
			want:  "``",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringLiteral(tt.value)
			if got == nil {
				t.Fatal("stringLiteral() returned nil")
			}
			// The value is a BasicLit, check its Value field
			lit, ok := got.(*ast.BasicLit)
			if !ok {
				t.Fatal("stringLiteral() did not return *ast.BasicLit")
			}
			if lit.Value != tt.want {
				t.Errorf("stringLiteral() = %q, want %q", lit.Value, tt.want)
			}
		})
	}
}

func TestParseStmt(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "simple statement",
			code:    "x := 1",
			wantErr: false,
		},
		{
			name:    "if statement",
			code:    "if x > 0 { return x }",
			wantErr: false,
		},
		{
			name:    "for loop",
			code:    "for i := 0; i < 10; i++ { fmt.Println(i) }",
			wantErr: false,
		},
		{
			name:    "invalid code",
			code:    "this is not valid go code @#$%",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStmt(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStmt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("parseStmt() returned nil")
			}
		})
	}
}

func TestMustParseStmt(t *testing.T) {
	t.Run("valid statement", func(t *testing.T) {
		got := mustParseStmt("x := 1")
		if got == nil {
			t.Error("mustParseStmt() returned nil")
		}
	})

	t.Run("panic on invalid", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("mustParseStmt() did not panic on invalid code")
			}
		}()
		mustParseStmt("this is not valid go code @#$%")
	})
}

func TestBuildQuerierFile(t *testing.T) {
	b := New(Options{Package: "test"})

	tests := []struct {
		name    string
		queries []queryInfo
		wantErr bool
	}{
		{
			name:    "empty queries",
			queries: []queryInfo{},
			wantErr: false,
		},
		{
			name: "query with variadic params",
			queries: []queryInfo{
				{
					methodName: "GetUsersByIDs",
					params: []paramSpec{
						{name: "ids", goType: "int64", variadic: true},
					},
					returnType: "[]GetUsersByIDsRow",
				},
			},
			wantErr: false,
		},
		{
			name: "query without return type",
			queries: []queryInfo{
				{
					methodName: "DeleteUser",
					params:     []paramSpec{},
					returnType: "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.buildQuerierFile("test", tt.queries)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildQuerierFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("buildQuerierFile() returned nil")
			}
		})
	}
}

func TestResolveType_Legacy(t *testing.T) {
	tests := []struct {
		name     string
		goType   string
		nullable bool
		want     string
	}{
		{
			name:     "int64 not null",
			goType:   "int64",
			nullable: false,
			want:     "int64",
		},
		{
			name:     "int64 nullable",
			goType:   "int64",
			nullable: true,
			want:     "sql.NullInt64",
		},
		{
			name:     "float64 nullable",
			goType:   "float64",
			nullable: true,
			want:     "sql.NullFloat64",
		},
		{
			name:     "string nullable",
			goType:   "string",
			nullable: true,
			want:     "sql.NullString",
		},
		{
			name:     "bool nullable",
			goType:   "bool",
			nullable: true,
			want:     "sql.NullBool",
		},
		{
			name:     "custom type nullable",
			goType:   "CustomType",
			nullable: true,
			want:     "*CustomType",
		},
		{
			name:     "empty type",
			goType:   "",
			nullable: false,
			want:     "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveType(tt.goType, tt.nullable)
			if got.GoType != tt.want {
				t.Errorf("resolveType() = %q, want %q", got.GoType, tt.want)
			}
		})
	}
}

func TestTypeResolver_ResolveType_GoTypes(t *testing.T) {
	tr := NewTypeResolver(nil)

	tests := []struct {
		name     string
		input    string
		nullable bool
		want     string
	}{
		{
			name:     "package qualified type",
			input:    "example.IDWrap",
			nullable: false,
			want:     "example.IDWrap",
		},
		{
			name:     "package qualified type nullable",
			input:    "example.IDWrap",
			nullable: true,
			want:     "*example.IDWrap",
		},
		{
			name:     "pointer type",
			input:    "*CustomType",
			nullable: false,
			want:     "*CustomType",
		},
		{
			name:     "pointer type nullable",
			input:    "*CustomType",
			nullable: true,
			want:     "*CustomType",
		},
		{
			name:     "sql.NullInt64 not null",
			input:    "sql.NullInt64",
			nullable: false,
			want:     "sql.NullInt64",
		},
		{
			name:     "sql.NullInt64 nullable",
			input:    "sql.NullInt64",
			nullable: true,
			want:     "*sql.NullInt64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.ResolveType(tt.input, tt.nullable)
			if got.GoType != tt.want {
				t.Errorf("ResolveType() = %q, want %q", got.GoType, tt.want)
			}
		})
	}
}

func TestTypeResolver_FindCustomMappingBySQLiteType(t *testing.T) {
	// Test with nil transformer
	tr := NewTypeResolver(nil)
	got := tr.findCustomMappingBySQLType("INTEGER")
	if got != nil {
		t.Errorf("findCustomMappingBySQLType() = %v, want nil", got)
	}
}

func TestTypeResolver_ResolveType_SQLiteTypes(t *testing.T) {
	tr := NewTypeResolver(nil)

	tests := []struct {
		name     string
		sqlType  string
		nullable bool
		wantType string
		wantSQL  bool
	}{
		{"INTEGER not null", "INTEGER", false, "int32", false},
		{"INTEGER nullable", "INTEGER", true, "*int32", false}, // int32 uses pointer for nullable
		{"BIGINT not null", "BIGINT", false, "int64", false},
		{"BIGINT nullable", "BIGINT", true, "sql.NullInt64", true},
		{"SMALLINT not null", "SMALLINT", false, "int16", false},
		{"SMALLINT nullable", "SMALLINT", true, "*int16", false}, // int16 uses pointer for nullable
		{"TINYINT not null", "TINYINT", false, "int8", false},
		{"TINYINT nullable", "TINYINT", true, "*int8", false}, // int8 uses pointer for nullable
		{"TEXT not null", "TEXT", false, "string", false},
		{"TEXT nullable", "TEXT", true, "sql.NullString", true},
		{"VARCHAR not null", "VARCHAR", false, "string", false},
		{"VARCHAR nullable", "VARCHAR", true, "sql.NullString", true},
		{"CHAR not null", "CHAR", false, "string", false},
		{"CHAR nullable", "CHAR", true, "sql.NullString", true},
		{"BLOB not null", "BLOB", false, "[]byte", false},
		{"BLOB nullable", "BLOB", true, "*[]byte", false},
		{"REAL not null", "REAL", false, "float64", false},
		{"REAL nullable", "REAL", true, "sql.NullFloat64", true},
		{"FLOAT not null", "FLOAT", false, "float64", false},
		{"FLOAT nullable", "FLOAT", true, "sql.NullFloat64", true},
		{"DOUBLE not null", "DOUBLE", false, "float64", false},
		{"DOUBLE nullable", "DOUBLE", true, "sql.NullFloat64", true},
		{"BOOLEAN not null", "BOOLEAN", false, "bool", false},
		{"BOOLEAN nullable", "BOOLEAN", true, "sql.NullBool", true},
		{"BOOL not null", "BOOL", false, "bool", false},
		{"BOOL nullable", "BOOL", true, "sql.NullBool", true},
		{"NUMERIC not null", "NUMERIC", false, "float64", false},
		{"NUMERIC nullable", "NUMERIC", true, "sql.NullFloat64", true},
		{"DECIMAL not null", "DECIMAL", false, "float64", false},
		{"DECIMAL nullable", "DECIMAL", true, "sql.NullFloat64", true},
		{"UNKNOWN not null", "UNKNOWN", false, "any", false},
		{"UNKNOWN nullable", "UNKNOWN", true, "*any", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.ResolveType(tt.sqlType, tt.nullable)
			if got.GoType != tt.wantType {
				t.Errorf("ResolveType() GoType = %q, want %q", got.GoType, tt.wantType)
			}
			if got.UsesSQLNull != tt.wantSQL {
				t.Errorf("ResolveType() UsesSQLNull = %v, want %v", got.UsesSQLNull, tt.wantSQL)
			}
		})
	}
}

func TestTypeResolver_ResolveType_WithPointersForNull(t *testing.T) {
	tr := NewTypeResolverWithOptions(nil, true)

	tests := []struct {
		name     string
		sqlType  string
		nullable bool
		want     string
	}{
		{"INTEGER not null", "INTEGER", false, "int32"},
		{"INTEGER nullable with pointer", "INTEGER", true, "*int32"},
		{"TEXT nullable with pointer", "TEXT", true, "*string"},
		{"REAL nullable with pointer", "REAL", true, "*float64"},
		{"BOOLEAN nullable with pointer", "BOOLEAN", true, "*bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.ResolveType(tt.sqlType, tt.nullable)
			if got.GoType != tt.want {
				t.Errorf("ResolveType() = %q, want %q", got.GoType, tt.want)
			}
		})
	}
}
