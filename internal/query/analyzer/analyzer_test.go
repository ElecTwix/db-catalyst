package analyzer_test

import (
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestAnalyzerAnalyze(t *testing.T) {
	catalog := buildTestCatalog()
	testCases := []struct {
		name    string
		catalog *model.Catalog
		sql     string
		assert  func(t *testing.T, res analyzer.Result)
	}{
		{
			name:    "select schema columns and params",
			catalog: catalog,
			sql: `SELECT users.id, users.email FROM users
WHERE users.id = :id AND users.email = :email;`,
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}

				if len(res.Columns) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(res.Columns))
				}

				first := res.Columns[0]
				if first.Name != "id" || first.GoType != "int64" || first.Nullable {
					t.Errorf("unexpected first column %+v", first)
				}

				second := res.Columns[1]
				if second.Name != "email" || second.GoType != "string" || !second.Nullable {
					t.Errorf("unexpected second column %+v", second)
				}

				if len(res.Params) != 2 {
					t.Fatalf("expected 2 params, got %d", len(res.Params))
				}

				if res.Params[0].Name != "id" || res.Params[0].GoType != "int64" || res.Params[0].Nullable {
					t.Errorf("unexpected first param %+v", res.Params[0])
				}
				if res.Params[1].Name != "email" || res.Params[1].GoType != "string" || !res.Params[1].Nullable {
					t.Errorf("unexpected second param %+v", res.Params[1])
				}
			},
		},
		{
			name:    "unknown table diagnostic",
			catalog: catalog,
			sql:     "SELECT orders.id FROM orders;",
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) == 0 {
					t.Fatalf("expected diagnostics, got none")
				}
				d := res.Diagnostics[0]
				if d.Severity != analyzer.SeverityError {
					t.Fatalf("expected error severity, got %v", d.Severity)
				}
				if !strings.Contains(d.Message, "unknown table") {
					t.Fatalf("expected unknown table message, got %q", d.Message)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				if res.Columns[0].GoType != "interface{}" || !res.Columns[0].Nullable {
					t.Errorf("expected fallback column typing, got %+v", res.Columns[0])
				}
			},
		},
		{
			name:    "nil catalog fallback",
			catalog: nil,
			sql: `SELECT users.id FROM users
WHERE users.id = :id;`,
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 1 {
					t.Fatalf("expected one warning, got %d", len(res.Diagnostics))
				}
				d := res.Diagnostics[0]
				if d.Severity != analyzer.SeverityWarning {
					t.Fatalf("expected warning severity, got %v", d.Severity)
				}
				if !strings.Contains(d.Message, "catalog unavailable") {
					t.Fatalf("unexpected warning message %q", d.Message)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				if res.Columns[0].GoType != "interface{}" || !res.Columns[0].Nullable {
					t.Errorf("expected fallback column typing, got %+v", res.Columns[0])
				}
				if len(res.Params) != 1 {
					t.Fatalf("expected 1 param, got %d", len(res.Params))
				}
				if res.Params[0].GoType != "interface{}" || !res.Params[0].Nullable {
					t.Errorf("expected fallback param typing, got %+v", res.Params[0])
				}
			},
		},
		{
			name:    "parameter equality patterns",
			catalog: catalog,
			sql: `SELECT users.email FROM users
WHERE users.email = ? AND ? = users.id;`,
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Params) != 2 {
					t.Fatalf("expected 2 params, got %d", len(res.Params))
				}
				if res.Params[0].Name != "arg1" || res.Params[0].GoType != "string" || !res.Params[0].Nullable {
					t.Errorf("unexpected first param %+v", res.Params[0])
				}
				if res.Params[1].Name != "arg2" || res.Params[1].GoType != "int64" || res.Params[1].Nullable {
					t.Errorf("unexpected second param %+v", res.Params[1])
				}
			},
		},
		{
			name:    "insert parameter mapping",
			catalog: catalog,
			sql:     "INSERT INTO users (id, email) VALUES (:id, :email);",
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Columns) != 0 {
					t.Fatalf("expected no result columns, got %d", len(res.Columns))
				}
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Params) != 2 {
					t.Fatalf("expected 2 params, got %d", len(res.Params))
				}
				if res.Params[0].Name != "id" || res.Params[0].GoType != "int64" || res.Params[0].Nullable {
					t.Errorf("unexpected first param %+v", res.Params[0])
				}
				if res.Params[1].Name != "email" || res.Params[1].GoType != "string" || !res.Params[1].Nullable {
					t.Errorf("unexpected second param %+v", res.Params[1])
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			blk := block.Block{
				Path:   "query/test.sql",
				Line:   1,
				Column: 1,
				SQL:    tc.sql,
			}
			q, diags := parser.Parse(blk)
			if len(diags) != 0 {
				t.Fatalf("unexpected parser diagnostics: %+v", diags)
			}
			res := analyzer.New(tc.catalog).Analyze(q)
			tc.assert(t, res)
		})
	}
}

func BenchmarkAnalyzeSelect(b *testing.B) {
	cat := buildTestCatalog()
	blk := block.Block{
		Path:   "query/bench.sql",
		Line:   1,
		Column: 1,
		SQL: `SELECT users.id, users.email FROM users
WHERE users.id = :id AND users.email = :email;`,
	}
	q, diags := parser.Parse(blk)
	if len(diags) != 0 {
		b.Fatalf("unexpected parser diagnostics: %+v", diags)
	}

	an := analyzer.New(cat)
	b.ReportAllocs()
	for b.Loop() {
		an.Analyze(q)
	}
}

func buildTestCatalog() *model.Catalog {
	return &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: false},
					{Name: "status", Type: "NUMERIC", NotNull: false},
				},
			},
		},
	}
}
