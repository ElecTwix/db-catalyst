package parser

import (
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/query/block"
)

func TestParseSelectSuccess(t *testing.T) {
	blk := block.Block{
		Path:   "query/users.sql",
		Line:   10,
		Column: 1,
		SQL: `SELECT u.id, u.name AS full_name, COUNT(o.id) total_orders
FROM users u
JOIN orders o ON o.user_id = u.id
WHERE u.status = :status AND u.score > ? AND u.id = ?1;`,
	}

	q, diags := Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if q.Verb != VerbSelect {
		t.Fatalf("expected VerbSelect, got %v", q.Verb)
	}
	if len(q.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(q.Columns))
	}

	first := q.Columns[0]
	if first.Expr != "u.id" {
		t.Errorf("unexpected first column expr %q", first.Expr)
	}
	if first.Alias != "id" {
		t.Errorf("expected alias id, got %q", first.Alias)
	}
	if first.Table != "u" {
		t.Errorf("expected table u, got %q", first.Table)
	}

	third := q.Columns[2]
	if third.Expr != "COUNT(o.id)" {
		t.Errorf("unexpected third column expr %q", third.Expr)
	}
	if third.Alias != "total_orders" {
		t.Errorf("expected alias total_orders, got %q", third.Alias)
	}
	if third.Table != "" {
		t.Errorf("expected empty table for aggregate, got %q", third.Table)
	}

	if len(q.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(q.Params))
	}
	if q.Params[0].Name != "status" || q.Params[0].Style != ParamStyleNamed {
		t.Errorf("unexpected first param: %+v", q.Params[0])
	}
	if q.Params[1].Name != "arg1" || q.Params[1].Order != 1 || q.Params[1].Style != ParamStylePositional {
		t.Errorf("unexpected second param: %+v", q.Params[1])
	}
}

func TestParseSelectMissingAlias(t *testing.T) {
	blk := block.Block{
		Path:   "query/users.sql",
		Line:   5,
		Column: 1,
		SQL:    "SELECT COUNT(*) FROM users;",
	}

	q, diags := Parse(blk)
	if len(q.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(q.Columns))
	}
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Severity != SeverityError {
		t.Fatalf("expected error severity, got %v", diags[0].Severity)
	}
	if want := "requires alias"; !strings.Contains(diags[0].Message, want) {
		t.Fatalf("expected diagnostic message to contain %q, got %q", want, diags[0].Message)
	}
}

func TestParseParametersNumbered(t *testing.T) {
	blk := block.Block{
		Path:   "query/books.sql",
		Line:   3,
		Column: 1,
		SQL:    "SELECT b.id FROM books b WHERE b.author_id = ?2 OR b.author_id = ?1 OR b.author_id = ?2;",
	}

	q, diags := Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if len(q.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(q.Params))
	}
	if q.Params[0].Name != "arg2" || q.Params[0].Order != 2 {
		t.Errorf("unexpected first param %+v", q.Params[0])
	}
	if q.Params[1].Name != "arg1" || q.Params[1].Order != 1 {
		t.Errorf("unexpected second param %+v", q.Params[1])
	}
}

func TestParseVerbUnsupported(t *testing.T) {
	blk := block.Block{
		Path:   "query/cte.sql",
		Line:   1,
		Column: 1,
		SQL: `WITH cte AS (
    SELECT 1 AS value
)
SELECT value FROM cte;`,
	}

	_, diags := Parse(blk)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if want := "unsupported query verb"; !strings.Contains(diags[0].Message, want) {
		t.Fatalf("expected diagnostic to contain %q, got %q", want, diags[0].Message)
	}
}

func TestParseInsertVerb(t *testing.T) {
	blk := block.Block{
		Path:   "query/users.sql",
		Line:   20,
		Column: 1,
		SQL:    "INSERT INTO users(id, name, type) VALUES(:id, :name, :type);",
	}

	q, diags := Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if q.Verb != VerbInsert {
		t.Fatalf("expected VerbInsert, got %v", q.Verb)
	}
	if len(q.Columns) != 0 {
		t.Fatalf("expected no columns for INSERT, got %d", len(q.Columns))
	}
	if len(q.Params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(q.Params))
	}
	if q.Params[2].Name != "type_" {
		t.Errorf("expected keyword param to be suffixed, got %q", q.Params[2].Name)
	}
}

func BenchmarkParseSelect(b *testing.B) {
	blk := block.Block{
		Path:   "query/bench.sql",
		Line:   1,
		Column: 1,
		SQL: `SELECT u.id, u.email, COALESCE(p.phone, '') phone, COUNT(o.id) total_orders
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.status = :status AND u.signup_after >= :signup_after AND u.score > ? AND u.tier = ?2;`,
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, diags := Parse(blk); len(diags) != 0 {
			b.Fatalf("unexpected diagnostics: %+v", diags)
		}
	}
}
