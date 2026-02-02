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
			//nolint:thelper // Anonymous function in test table
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
			name:    "cte passthrough",
			catalog: catalog,
			sql: `WITH latest AS (
    SELECT u.id, u.email FROM users u WHERE u.status = :status
)
SELECT latest.id, latest.email FROM latest;`,
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(res.Columns))
				}
				if res.Columns[0].GoType != "int64" || res.Columns[0].Nullable {
					t.Errorf("unexpected first column %+v", res.Columns[0])
				}
				if res.Columns[1].GoType != "string" || !res.Columns[1].Nullable {
					t.Errorf("unexpected second column %+v", res.Columns[1])
				}
				if len(res.Params) != 1 {
					t.Fatalf("expected 1 param, got %d", len(res.Params))
				}
				if res.Params[0].Name != "status" || res.Params[0].GoType != "string" {
					t.Errorf("unexpected param %+v", res.Params[0])
				}
			},
		},
		{
			name:    "recursive cte counter",
			catalog: catalog,
			sql: `WITH RECURSIVE numbers(id, depth) AS (
    SELECT u.id, 0 FROM users u WHERE u.id = :user_id
    UNION ALL
    SELECT u.id, numbers.depth + 1 FROM users u JOIN numbers ON u.id = numbers.id
)
SELECT numbers.id, numbers.depth FROM numbers;`,
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Columns) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(res.Columns))
				}
				if res.Columns[0].GoType != "int64" || res.Columns[0].Nullable {
					t.Errorf("unexpected id column %+v", res.Columns[0])
				}
				if res.Columns[1].GoType != "any" || !res.Columns[1].Nullable {
					t.Errorf("unexpected depth column %+v", res.Columns[1])
				}
				if len(res.Params) != 1 {
					t.Fatalf("expected 1 param, got %d", len(res.Params))
				}
				if res.Params[0].Name != "userId" || res.Params[0].GoType != "int64" {
					t.Errorf("unexpected param %+v", res.Params[0])
				}
				foundWarning := false
				for _, d := range res.Diagnostics {
					if d.Severity == analyzer.SeverityWarning && strings.Contains(d.Message, "defaulting to interface{}") {
						foundWarning = true
						break
					}
				}
				if !foundWarning {
					t.Fatalf("expected warning about interface{} fallback, got %+v", res.Diagnostics)
				}
			},
		},
		{
			name:    "recursive cte column mismatch",
			catalog: catalog,
			sql: `WITH RECURSIVE bad_cte(id, depth) AS (
    SELECT u.id, 0 FROM users u
    UNION ALL
    SELECT u.id, bad_cte.depth, u.email FROM users u JOIN bad_cte ON bad_cte.id = u.id
)
SELECT id, depth FROM bad_cte;`,
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) == 0 {
					t.Fatalf("expected diagnostics, got none")
				}
				found := false
				for _, d := range res.Diagnostics {
					if d.Severity == analyzer.SeverityError && strings.Contains(d.Message, "projects") {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected column count diagnostic, got %+v", res.Diagnostics)
				}
			},
		},
		{
			name:    "cte join with aliases",
			catalog: catalog,
			sql: `WITH flagged AS (
    SELECT u.id, u.email FROM users u WHERE u.status = :status
)
SELECT u.id, flagged.email
FROM users u
JOIN flagged ON flagged.id = u.id;`,
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(res.Columns))
				}
				if res.Columns[0].GoType != "int64" || res.Columns[0].Nullable {
					t.Errorf("unexpected first column %+v", res.Columns[0])
				}
				if res.Columns[1].GoType != "string" || !res.Columns[1].Nullable {
					t.Errorf("unexpected second column %+v", res.Columns[1])
				}
				if len(res.Params) != 1 {
					t.Fatalf("expected 1 param, got %d", len(res.Params))
				}
				if res.Params[0].Name != "status" || res.Params[0].GoType != "string" {
					t.Errorf("unexpected param %+v", res.Params[0])
				}
			},
		},
		{
			name:    "aggregate count star alias",
			catalog: catalog,
			sql:     "SELECT COUNT(*) AS total FROM users;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				col := res.Columns[0]
				if col.GoType != "int64" || col.Nullable {
					t.Errorf("unexpected aggregate column %+v", col)
				}
			},
		},
		{
			name:    "aggregate count column not nullable",
			catalog: catalog,
			sql:     "SELECT COUNT(users.email) AS email_count FROM users;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				col := res.Columns[0]
				if col.GoType != "int64" || col.Nullable {
					t.Errorf("unexpected aggregate column %+v", col)
				}
			},
		},
		{
			name:    "aggregate sum nullable integer",
			catalog: catalog,
			sql:     "SELECT SUM(users.credits) AS total_credits FROM users;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				col := res.Columns[0]
				if col.GoType != "int64" || !col.Nullable {
					t.Errorf("unexpected aggregate column %+v", col)
				}
			},
		},
		{
			name:    "aggregate max text column",
			catalog: catalog,
			sql:     "SELECT MAX(users.email) AS max_email FROM users;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				col := res.Columns[0]
				if col.GoType != "string" || !col.Nullable {
					t.Errorf("unexpected aggregate column %+v", col)
				}
			},
		},
		{
			name:    "cte aggregate propagation",
			catalog: catalog,
			sql: `WITH totals(count_users) AS (
    SELECT COUNT(*) AS count_users FROM users
)
SELECT totals.count_users FROM totals;`,
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				col := res.Columns[0]
				if col.GoType != "int64" || col.Nullable {
					t.Errorf("unexpected aggregate column %+v", col)
				}
			},
		},
		{
			name:    "unknown table diagnostic",
			catalog: catalog,
			sql:     "SELECT orders.id FROM orders;",
			//nolint:thelper // Anonymous function in test table
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
				if res.Columns[0].GoType != "any" || !res.Columns[0].Nullable {
					t.Errorf("expected fallback column typing, got %+v", res.Columns[0])
				}
			},
		},
		{
			name:    "nil catalog fallback",
			catalog: nil,
			sql: `SELECT users.id FROM users
WHERE users.id = :id;`,
			//nolint:thelper // Anonymous function in test table
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
				if res.Columns[0].GoType != "any" || !res.Columns[0].Nullable {
					t.Errorf("expected fallback column typing, got %+v", res.Columns[0])
				}
				if len(res.Params) != 1 {
					t.Fatalf("expected 1 param, got %d", len(res.Params))
				}
				if res.Params[0].GoType != "any" || !res.Params[0].Nullable {
					t.Errorf("expected fallback param typing, got %+v", res.Params[0])
				}
			},
		},
		{
			name:    "parameter equality patterns",
			catalog: catalog,
			sql: `SELECT users.email FROM users
WHERE users.email = ? AND ? = users.id;`,
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Params) != 2 {
					t.Fatalf("expected 2 params, got %d", len(res.Params))
				}
				// First param inferred from users.email = ?
				if res.Params[0].Name != "email" || res.Params[0].GoType != "string" || !res.Params[0].Nullable {
					t.Errorf("unexpected first param %+v", res.Params[0])
				}
				// Second param - reversed pattern ? = users.id is harder to infer
				// because we need to look ahead, not backward
				if res.Params[1].Name != "arg2" || res.Params[1].GoType != "int64" || res.Params[1].Nullable {
					t.Errorf("unexpected second param %+v", res.Params[1])
				}
			},
		},
		{
			name:    "insert parameter mapping",
			catalog: catalog,
			sql:     "INSERT INTO users (id, email) VALUES (:id, :email);",
			//nolint:thelper // Anonymous function in test table
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
		{
			name:    "sqlc slice parameter",
			catalog: catalog,
			sql:     "SELECT id FROM users WHERE id IN (sqlc.slice('ids'));",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				if len(res.Params) != 1 {
					t.Fatalf("expected 1 param, got %d", len(res.Params))
				}
				p := res.Params[0]
				if p.Name != "ids" {
					t.Errorf("expected param name ids, got %s", p.Name)
				}
				if !p.IsVariadic {
					t.Errorf("expected param to be variadic")
				}
				if p.GoType != "[]int64" {
					t.Errorf("expected param type []int64, got %s", p.GoType)
				}
			},
		},
		{
			name:    "aggregate count star implicit alias",
			catalog: catalog,
			sql:     "SELECT COUNT(*) FROM users;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Columns) != 1 {
					t.Fatalf("expected 1 column, got %d", len(res.Columns))
				}
				col := res.Columns[0]
				if col.Name != "count" || col.GoType != "int64" {
					t.Errorf("unexpected aggregate column %+v", col)
				}
				foundWarning := false
				for _, d := range res.Diagnostics {
					if d.Severity == analyzer.SeverityWarning && strings.Contains(d.Message, "requires an alias; defaulting to \"count\"") {
						foundWarning = true
						break
					}
				}
				if !foundWarning {
					t.Errorf("expected warning about missing alias, got %+v", res.Diagnostics)
				}
			},
		},
		{
			name:    "star expansion",
			catalog: catalog,
			sql:     "SELECT * FROM users;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
				}
				// 4 columns in buildTestCatalog
				if len(res.Columns) != 4 {
					t.Fatalf("expected 4 columns, got %d", len(res.Columns))
				}
				names := make(map[string]bool)
				for _, c := range res.Columns {
					names[c.Name] = true
				}
				expected := []string{"id", "email", "status", "credits"}
				for _, name := range expected {
					if !names[name] {
						t.Errorf("missing expanded column %s", name)
					}
				}
			},
		},
		{
			name:    "insert returning star",
			catalog: catalog,
			sql:     "INSERT INTO users (id, email) VALUES (?, ?) RETURNING *;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				if len(res.Columns) != 4 {
					t.Fatalf("expected 4 columns from RETURNING *, got %d", len(res.Columns))
				}
				if res.Columns[0].Table != "users" {
					t.Errorf("expected table users, got %s", res.Columns[0].Table)
				}
			},
		},
		{
			name:    "where clause unknown column",
			catalog: catalog,
			sql:     "SELECT id FROM users WHERE users.unknown_col = ?;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				found := false
				for _, d := range res.Diagnostics {
					if strings.Contains(d.Message, "unknown column \"unknown_col\"") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected diagnostic for unknown column in WHERE clause, got %+v", res.Diagnostics)
				}
			},
		},
		{
			name:    "order by unknown column",
			catalog: catalog,
			sql:     "SELECT id FROM users ORDER BY users.invalid_col;",
			//nolint:thelper // Anonymous function in test table
			assert: func(t *testing.T, res analyzer.Result) {
				found := false
				for _, d := range res.Diagnostics {
					if strings.Contains(d.Message, "unknown column \"invalid_col\"") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected diagnostic for unknown column in ORDER BY clause, got %+v", res.Diagnostics)
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

			// We now allow missing aliases as warnings in parser too
			hasError := false
			for _, d := range diags {
				if d.Severity == parser.SeverityError {
					hasError = true
				}
			}
			if hasError {
				t.Fatalf("unexpected parser errors: %+v", diags)
			}
			res := analyzer.New(tc.catalog).Analyze(q)
			tc.assert(t, res)
		})
	}
}

func TestAnalyzerAggregateRequiresAlias(t *testing.T) {
	catalog := buildTestCatalog()
	blk := block.Block{
		Path:   "query/agg_missing_alias.sql",
		Line:   1,
		Column: 1,
		SQL:    "SELECT COUNT(*) FROM users;",
	}
	q, diags := parser.Parse(blk)

	// Parser now emits warning
	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "requires alias") && d.Severity == parser.SeverityWarning {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected requires alias warning in parser, got %+v", diags)
	}

	res := analyzer.New(catalog).Analyze(q)
	if len(res.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(res.Columns))
	}
	col := res.Columns[0]
	if col.GoType != "int64" || col.Nullable {
		t.Errorf("expected COUNT(*) column to be int64 and not nullable, got %+v", col)
	}
	if col.Name != "count" {
		t.Errorf("expected default name 'count', got %q", col.Name)
	}

	found = false
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "requires an alias; defaulting to \"count\"") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected analyzer to surface alias warning, got %+v", res.Diagnostics)
	}
}

func TestAnalyzer_JoinTwoTables(t *testing.T) {
	catalog := buildTestCatalog()
	sql := `SELECT u.id, u.email, p.title 
FROM users u 
JOIN posts p ON p.user_id = u.id`

	blk := block.Block{
		Path:   "query/join.sql",
		Line:   1,
		Column: 1,
		SQL:    sql,
	}

	q, diags := parser.Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected parser diagnostics: %+v", diags)
	}

	res := analyzer.New(catalog).Analyze(q)

	t.Logf("Diagnostics: %+v", res.Diagnostics)
	for _, d := range res.Diagnostics {
		t.Logf("  - Line %d:%d: %s", d.Line, d.Column, d.Message)
	}

	if len(res.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}

	if len(res.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(res.Columns))
	}

	if res.Columns[0].Name != "id" || res.Columns[0].GoType != "int64" {
		t.Errorf("unexpected first column %+v", res.Columns[0])
	}
	if res.Columns[1].Name != "email" || res.Columns[1].GoType != "string" {
		t.Errorf("unexpected second column %+v", res.Columns[1])
	}
	if res.Columns[2].Name != "title" || res.Columns[2].GoType != "string" {
		t.Errorf("unexpected third column %+v", res.Columns[2])
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
					{Name: "credits", Type: "INTEGER", NotNull: false},
				},
			},
			"posts": {
				Name: "posts",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "user_id", Type: "INTEGER", NotNull: true},
					{Name: "title", Type: "TEXT", NotNull: false},
				},
			},
		},
	}
}
