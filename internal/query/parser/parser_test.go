package parser

import (
	"fmt"
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
	if len(q.CTEs) != 0 {
		t.Fatalf("expected no CTEs, got %d", len(q.CTEs))
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
	if diags[0].Severity != SeverityWarning {
		t.Fatalf("expected warning severity, got %v", diags[0].Severity)
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
	// Parameters are now inferred from context (authorId from b.author_id = ?)
	if q.Params[0].Order != 2 {
		t.Errorf("unexpected first param order %+v", q.Params[0])
	}
	if q.Params[1].Order != 1 {
		t.Errorf("unexpected second param order %+v", q.Params[1])
	}
}

func TestParseSelectWithCTE(t *testing.T) {
	blk := block.Block{
		Path:   "query/cte.sql",
		Line:   1,
		Column: 1,
		SQL: `WITH cte AS (
    SELECT 1 AS value
)
SELECT value FROM cte;`,
	}

	q, diags := Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if q.Verb != VerbSelect {
		t.Fatalf("expected VerbSelect, got %v", q.Verb)
	}
	if len(q.CTEs) != 1 {
		t.Fatalf("expected 1 CTE, got %d", len(q.CTEs))
	}
	cte := q.CTEs[0]
	if cte.Name != "cte" {
		t.Fatalf("expected CTE name cte, got %q", cte.Name)
	}
	if len(cte.Columns) != 0 {
		t.Fatalf("expected no CTE columns, got %d", len(cte.Columns))
	}
	if got := cte.SelectSQL; got != "SELECT 1 AS value" {
		t.Fatalf("unexpected CTE SQL %q", got)
	}
	if len(q.Columns) != 1 {
		t.Fatalf("expected 1 result column, got %d", len(q.Columns))
	}
	if q.Columns[0].Expr != "value" {
		t.Fatalf("expected first column expr value, got %q", q.Columns[0].Expr)
	}
}

func TestParseWithRecursiveCTE(t *testing.T) {
	blk := block.Block{
		Path:   "query/recursive.sql",
		Line:   1,
		Column: 1,
		SQL: `WITH RECURSIVE ancestors(id, depth) AS (
    SELECT id, 0 FROM users WHERE id = :target_id
    UNION ALL
    SELECT p.parent_id, a.depth + 1
    FROM parents p
    JOIN ancestors a ON a.id = p.child_id
)
SELECT id, depth FROM ancestors;`,
	}

	q, diags := Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if q.Verb != VerbSelect {
		t.Fatalf("expected VerbSelect, got %v", q.Verb)
	}
	if len(q.CTEs) != 1 {
		t.Fatalf("expected 1 CTE, got %d", len(q.CTEs))
	}
	cte := q.CTEs[0]
	if cte.Name != "ancestors" {
		t.Fatalf("expected CTE name ancestors, got %q", cte.Name)
	}
	if len(cte.Columns) != 2 || cte.Columns[0] != "id" || cte.Columns[1] != "depth" {
		t.Fatalf("unexpected CTE columns: %+v", cte.Columns)
	}
	if !strings.Contains(cte.SelectSQL, "UNION ALL") {
		t.Fatalf("expected recursive CTE SQL to contain UNION ALL, got %q", cte.SelectSQL)
	}
	if !strings.Contains(cte.SelectSQL, ":target_id") {
		t.Fatalf("expected recursive CTE SQL to reference :target_id, got %q", cte.SelectSQL)
	}
	if len(q.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(q.Params))
	}
	if q.Params[0].Name != "targetId" {
		t.Fatalf("unexpected param name %q", q.Params[0].Name)
	}
	if len(q.Columns) != 2 {
		t.Fatalf("expected 2 result columns, got %d", len(q.Columns))
	}
}

func TestParseInsertWithCTE(t *testing.T) {
	blk := block.Block{
		Path:   "query/insert_cte.sql",
		Line:   1,
		Column: 1,
		SQL: `WITH active_users AS (
    SELECT id FROM users WHERE status = :status
)
INSERT INTO snapshots(user_id)
SELECT id FROM active_users;`,
	}

	q, diags := Parse(blk)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if q.Verb != VerbInsert {
		t.Fatalf("expected VerbInsert, got %v", q.Verb)
	}
	if len(q.CTEs) != 1 {
		t.Fatalf("expected 1 CTE, got %d", len(q.CTEs))
	}
	if got := q.CTEs[0].SelectSQL; got != "SELECT id FROM users WHERE status = :status" {
		t.Fatalf("unexpected CTE SQL %q", got)
	}
	if len(q.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(q.Params))
	}
	if q.Params[0].Name != "status" {
		t.Fatalf("unexpected param name %q", q.Params[0].Name)
	}
	if q.Params[0].Style != ParamStyleNamed {
		t.Fatalf("unexpected param style %+v", q.Params[0])
	}
	if len(q.Columns) != 0 {
		t.Fatalf("expected no result columns for INSERT, got %d", len(q.Columns))
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

func TestParseVariadicInClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantCount int
	}{
		{
			name:      "UnnamedPlaceholders",
			sql:       "SELECT id FROM users WHERE id IN (?, ?, ?)",
			wantCount: 3,
		},
		{
			name:      "NumberedPlaceholders",
			sql:       "SELECT id FROM users WHERE id IN (?1, ?2, ?3, ?4)",
			wantCount: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blk := block.Block{
				Path:   "query/variadic.sql",
				Line:   1,
				Column: 1,
				SQL:    tc.sql,
			}
			q, diags := Parse(blk)
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", diags)
			}
			if len(q.Params) != 1 {
				t.Fatalf("expected 1 param, got %d", len(q.Params))
			}
			param := q.Params[0]
			if !param.IsVariadic {
				t.Fatalf("expected variadic parameter")
			}
			if param.VariadicCount != tc.wantCount {
				t.Fatalf("expected variadic count %d, got %d", tc.wantCount, param.VariadicCount)
			}
			if param.Style != ParamStylePositional {
				t.Fatalf("expected positional style, got %v", param.Style)
			}
			if param.Name != "id" {
				t.Fatalf("expected param name id, got %q", param.Name)
			}
		})
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

// TestInferParamName tests parameter name inference for various SQL patterns.
func TestInferParamName(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		paramIdx      int
		wantName      string
		wantAmbiguous bool // if true, we expect default "argN" name
	}{
		// Basic WHERE clause - already works
		{
			name:     "WHERE equals",
			sql:      "SELECT * FROM users WHERE id = ?",
			paramIdx: 0,
			wantName: "id",
		},
		{
			name:     "WHERE LIKE",
			sql:      "SELECT * FROM users WHERE name LIKE ?",
			paramIdx: 0,
			wantName: "name",
		},
		{
			name:     "WHERE IN",
			sql:      "SELECT * FROM users WHERE status IN (?)",
			paramIdx: 0,
			wantName: "status",
		},
		// JOIN conditions - new
		{
			name:     "JOIN ON clause",
			sql:      "SELECT * FROM users u JOIN orders o ON o.user_id = ?",
			paramIdx: 0,
			wantName: "userId",
		},
		{
			name:     "LEFT JOIN ON clause",
			sql:      "SELECT * FROM users u LEFT JOIN profiles p ON p.id = ?",
			paramIdx: 0,
			wantName: "id",
		},
		// UPDATE SET - new
		{
			name:     "UPDATE SET",
			sql:      "UPDATE users SET name = ? WHERE id = 1",
			paramIdx: 0,
			wantName: "name",
		},
		{
			name:     "UPDATE SET multiple",
			sql:      "UPDATE users SET name = ?, email = ? WHERE id = 1",
			paramIdx: 0,
			wantName: "name",
		},
		{
			name:     "UPDATE_SET_multiple_second_param",
			sql:      "UPDATE users SET name = ?, email = ? WHERE id = 1",
			paramIdx: 1,
			wantName: "email", // UPDATE SET params are now all inferred
		},
		// BETWEEN - new
		{
			name:     "BETWEEN first param",
			sql:      "SELECT * FROM users WHERE age BETWEEN ? AND ?",
			paramIdx: 0,
			wantName: "age",
		},
		{
			name:     "BETWEEN second param (now gets Upper suffix)",
			sql:      "SELECT * FROM users WHERE age BETWEEN ? AND ?",
			paramIdx: 1,
			wantName: "ageUpper",
		},
		// Forward equality pattern (reversed)
		{
			name:     "forward equality ? = column",
			sql:      "SELECT * FROM users WHERE ? = id",
			paramIdx: 0,
			wantName: "id",
		},
		{
			name:     "forward equality with table prefix",
			sql:      "SELECT * FROM users u WHERE ? = u.email",
			paramIdx: 0,
			wantName: "email",
		},
		{
			name:     "mixed equality forward then backward",
			sql:      "SELECT * FROM users WHERE ? = id AND name = ?",
			paramIdx: 0,
			wantName: "id",
		},
		{
			name:     "mixed equality backward then forward",
			sql:      "SELECT * FROM users WHERE email = ? AND ? = user_id",
			paramIdx: 1,
			wantName: "userId",
		},
		// HAVING clause
		{
			name:     "HAVING clause with COUNT",
			sql:      "SELECT user_id, COUNT(*) as cnt FROM orders GROUP BY user_id HAVING cnt > ?",
			paramIdx: 0,
			wantName: "cnt",
		},
		{
			name:     "HAVING clause with SUM",
			sql:      "SELECT category, SUM(amount) as total FROM sales GROUP BY category HAVING total < ?",
			paramIdx: 0,
			wantName: "total",
		},
		// Multiple params - different columns
		{
			name:     "multiple WHERE conditions first",
			sql:      "SELECT * FROM users WHERE id = ? AND name = ?",
			paramIdx: 0,
			wantName: "id",
		},
		{
			name:     "multiple WHERE conditions second",
			sql:      "SELECT * FROM users WHERE id = ? AND name = ?",
			paramIdx: 1,
			wantName: "name",
		},
		// Table-qualified columns
		{
			name:     "table qualified column",
			sql:      "SELECT * FROM users u WHERE u.email = ?",
			paramIdx: 0,
			wantName: "email",
		},
		// INSERT statements
		{
			name:     "INSERT with column list first param",
			sql:      "INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
			paramIdx: 0,
			wantName: "id",
		},
		{
			name:     "INSERT with column list second param",
			sql:      "INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
			paramIdx: 1,
			wantName: "name",
		},
		{
			name:     "INSERT with column list third param",
			sql:      "INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
			paramIdx: 2,
			wantName: "email",
		},
		{
			name:     "INSERT single column",
			sql:      "INSERT INTO users (name) VALUES (?)",
			paramIdx: 0,
			wantName: "name",
		},
		{
			name:     "INSERT without column list falls back to argN",
			sql:      "INSERT INTO users VALUES (?)",
			paramIdx: 0,
			wantName: "arg1",
		},
		// LIMIT and OFFSET
		{
			name:     "LIMIT parameter",
			sql:      "SELECT * FROM users LIMIT ?",
			paramIdx: 0,
			wantName: "limit",
		},
		{
			name:     "LIMIT with OFFSET",
			sql:      "SELECT * FROM users LIMIT ? OFFSET ?",
			paramIdx: 0,
			wantName: "limit",
		},
		{
			name:     "OFFSET parameter",
			sql:      "SELECT * FROM users LIMIT ? OFFSET ?",
			paramIdx: 1,
			wantName: "offset",
		},
		// UPDATE WHERE clause
		{
			name:     "UPDATE WHERE clause",
			sql:      "UPDATE users SET name = ? WHERE id = ?",
			paramIdx: 1,
			wantName: "id",
		},
		{
			name:     "UPDATE WHERE with multiple SET",
			sql:      "UPDATE users SET name = ?, email = ? WHERE id = ?",
			paramIdx: 2,
			wantName: "id",
		},
		// Complex LIKE patterns with OR
		{
			name:     "LIKE pattern with OR first param",
			sql:      "SELECT * FROM posts WHERE title LIKE '%' || ? || '%' OR content LIKE '%' || ? || '%'",
			paramIdx: 0,
			wantName: "title",
		},
		{
			name:     "LIKE pattern with OR second param",
			sql:      "SELECT * FROM posts WHERE title LIKE '%' || ? || '%' OR content LIKE '%' || ? || '%'",
			paramIdx: 1,
			wantName: "content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blk := block.Block{
				Path:   "query/test.sql",
				Line:   1,
				Column: 1,
				SQL:    tc.sql,
			}

			q, diags := Parse(blk)
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", diags)
			}

			if tc.paramIdx >= len(q.Params) {
				t.Fatalf("paramIdx %d out of range, only %d params found", tc.paramIdx, len(q.Params))
			}

			param := q.Params[tc.paramIdx]
			expectedName := tc.wantName
			if tc.wantAmbiguous {
				expectedName = fmt.Sprintf("arg%d", tc.paramIdx+1)
			}

			if param.Name != expectedName {
				t.Errorf("param[%d].Name = %q, want %q", tc.paramIdx, param.Name, expectedName)
			}
		})
	}
}

func TestParseSQLiteEdgeCases(t *testing.T) {
	t.Run("RepeatedNamedParameter", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT e1.expires_at FROM entities AS e1
WHERE e1.key = sqlc.arg('key')
AND e1.last_modified_at_block <= sqlc.arg('block')
AND NOT EXISTS (
  SELECT 1 FROM entities AS e2
  WHERE e2.key = e1.key
  AND e2.last_modified_at_block > e1.last_modified_at_block
  AND e2.last_modified_at_block <= sqlc.arg('block')
)`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			for _, d := range diags {
				t.Logf("diagnostic: %v", d)
			}
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params (key, block), got %d: %+v", len(q.Params), q.Params)
		}
		if q.Params[0].Name != "key" {
			t.Errorf("expected first param 'key', got %q", q.Params[0].Name)
		}
		if q.Params[1].Name != "block" {
			t.Errorf("expected second param 'block', got %q", q.Params[1].Name)
		}
	})

	t.Run("SliceWithNamedArgMixed", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM mytable WHERE typ IN (sqlc.slice('types')) AND (sqlc.arg('allnames') OR (name IN (sqlc.slice('names'))))`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 3 {
			t.Fatalf("expected 3 params (types, allnames, names), got %d: %+v", len(q.Params), q.Params)
		}
		if !q.Params[0].IsVariadic {
			t.Errorf("expected 'types' to be variadic")
		}
		if q.Params[1].IsVariadic {
			t.Errorf("expected 'allnames' to not be variadic")
		}
		if !q.Params[2].IsVariadic {
			t.Errorf("expected 'names' to be variadic")
		}
	})

	t.Run("OrderByExpressionParam", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT ID FROM Sequence WHERE SeriesID = ? ORDER BY Name = ? DESC, ID LIMIT 1`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
		if q.Params[0].Name != "seriesid" {
			t.Errorf("expected first param 'seriesid', got %q", q.Params[0].Name)
		}
		if q.Params[1].Name != "name" {
			t.Errorf("expected second param 'name', got %q", q.Params[1].Name)
		}
	})

	t.Run("OnConflictWithRepeatedParam", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `INSERT INTO x (user_id, x)
VALUES ((SELECT user_id FROM users WHERE email = ?), ?)
ON CONFLICT(x) 
DO UPDATE SET user_id = (SELECT user_id FROM users WHERE email = ?)`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 3 {
			t.Fatalf("expected 3 params (email twice, x once), got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("JSONGroupArrayWithOrderBy", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT item.*,
  COALESCE(
    JSON_GROUP_ARRAY(document.id ORDER BY document.created_at ASC) FILTER (WHERE document.id IS NOT NULL),
    '[]'
  ) as document_ids
FROM item
LEFT JOIN document ON document.item_id = item.id
GROUP BY item.id`,
		}
		q, diags := Parse(blk)
		for _, d := range diags {
			t.Logf("diagnostic: %v", d)
		}
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
	})

	t.Run("FilterClauseWithAggregate", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT COUNT(*) FILTER (WHERE status = 'active') AS active_count FROM users`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 1 {
			t.Fatalf("expected 1 column, got %d", len(q.Columns))
		}
		if q.Columns[0].Expr != "COUNT(*) FILTER (WHERE status = 'active')" {
			t.Errorf("unexpected column expr: %q", q.Columns[0].Expr)
		}
	})

	t.Run("CorrelatedSubqueryAlias", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT * FROM locations l
WHERE l.public_id = ?
AND EXISTS (
    SELECT 1 FROM projects p
    JOIN organization_members om ON p.organization_id = om.organization_id
    WHERE p.id = l.project_id AND om.account_id = ?
)`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("NotExistsInSubquery", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT * FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM blocked b
    WHERE b.user_id = u.id AND b.blocked_by = ?
)`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
		if q.Params[0].Name != "blockedBy" {
			t.Errorf("expected param 'blockedBy', got %q", q.Params[0].Name)
		}
	})

	t.Run("ChineseCommentsUTF8", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `-- 当新建MasterItem时，需要为所有App创建相应的AppItem
INSERT INTO app_items (app_id, master_item_id)
SELECT a.id, ? FROM apps a
WHERE NOT EXISTS (
    SELECT 1 FROM app_items ai 
    WHERE ai.app_id = a.id AND ai.master_item_id = ?
)`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbInsert {
			t.Errorf("expected VerbInsert, got %v", q.Verb)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("UpsertReturning", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `INSERT INTO users (email, name) VALUES (?, ?)
ON CONFLICT(email) DO UPDATE SET name = excluded.name
RETURNING id, email`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbInsert {
			t.Errorf("expected VerbInsert, got %v", q.Verb)
		}
		// Note: RETURNING columns are discovered by analyzer, not parser
	})

	t.Run("MultipleSlicesInQuery", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM items WHERE category_id IN (sqlc.slice('cat_ids')) AND tag_id IN (sqlc.slice('tag_ids'))`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
		if !q.Params[0].IsVariadic || !q.Params[1].IsVariadic {
			t.Errorf("expected both params to be variadic")
		}
	})

	t.Run("ExcludedTableReference", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `INSERT INTO users (id, name, email) VALUES (?, ?, ?)
ON CONFLICT(id) DO UPDATE SET 
  name = excluded.name,
  email = excluded.email
RETURNING id`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbInsert {
			t.Errorf("expected VerbInsert, got %v", q.Verb)
		}
		// Note: RETURNING columns are discovered by analyzer, not parser
	})

	t.Run("CaseExpressionInSelect", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT 
  id,
  CASE 
    WHEN status = 'active' THEN 1
    WHEN status = 'inactive' THEN 0
    ELSE -1
  END AS status_code
FROM users
WHERE id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(q.Columns))
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("WindowFunction", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT 
  id,
  ROW_NUMBER() OVER (PARTITION BY category ORDER BY created_at DESC) AS row_num
FROM items
WHERE user_id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(q.Columns))
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("SubqueryInSelect", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT 
  u.id,
  (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) AS order_count
FROM users u
WHERE u.status = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(q.Columns))
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("ValuesClauseWithMultipleRows", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `INSERT INTO items (name, value) VALUES (?, ?), (?, ?), (?, ?)`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 6 {
			t.Fatalf("expected 6 params, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("NullLiteralComparison", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users WHERE deleted_at IS NULL AND status = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("BooleanLiteralComparison", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users WHERE deleted = FALSE AND role = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("GlobOperator", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM files WHERE path GLOB ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("CastExpression", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT CAST(score AS REAL) AS score_real FROM items WHERE id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 1 {
			t.Fatalf("expected 1 column, got %d", len(q.Columns))
		}
	})

	t.Run("CoalesceExpression", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT COALESCE(nickname, email, 'unknown') AS display_name FROM users WHERE id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 1 {
			t.Fatalf("expected 1 column, got %d", len(q.Columns))
		}
	})

	t.Run("NullIfExpression", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT NULLIF(status, 'deleted') AS effective_status FROM items WHERE id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 1 {
			t.Fatalf("expected 1 column, got %d", len(q.Columns))
		}
	})

	t.Run("IifFunction", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT IIF(active = 1, 'yes', 'no') AS is_active FROM users WHERE id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 1 {
			t.Fatalf("expected 1 column, got %d", len(q.Columns))
		}
	})

	t.Run("DollarSignParameter", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `UPDATE handles SET name = $2, updated_at = $1 WHERE id = $1`,
		}
		q, diags := Parse(blk)
		for _, d := range diags {
			t.Logf("diagnostic: %v", d)
		}
		// Note: $ params may or may not be fully supported depending on implementation
		if q.Verb != VerbUpdate {
			t.Errorf("expected VerbUpdate, got %v", q.Verb)
		}
	})

	t.Run("BetweenAfterAnd", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT id FROM test WHERE name = ? AND id BETWEEN ? AND ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 3 {
			t.Fatalf("expected 3 params (name, id lower, id upper), got %d: %+v", len(q.Params), q.Params)
		}
		if q.Params[0].Name != "name" {
			t.Errorf("expected first param 'name', got %q", q.Params[0].Name)
		}
	})

	t.Run("BetweenNamedAfterAnd", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT id FROM test WHERE name = :name AND id BETWEEN :min AND :max`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 3 {
			t.Fatalf("expected 3 params, got %d: %+v", len(q.Params), q.Params)
		}
		if q.Params[0].Name != "name" {
			t.Errorf("expected first param 'name', got %q", q.Params[0].Name)
		}
		if q.Params[1].Name != "min" {
			t.Errorf("expected second param 'min', got %q", q.Params[1].Name)
		}
		if q.Params[2].Name != "max" {
			t.Errorf("expected third param 'max', got %q", q.Params[2].Name)
		}
	})

	t.Run("SelectWithoutFromWhereNotExists", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT sqlc.arg('id'), sqlc.arg('name')
WHERE NOT EXISTS (
    SELECT 1 FROM rels
    WHERE rels.left = sqlc.arg('left') AND rels.right = sqlc.arg('right')
)`,
		}
		q, diags := Parse(blk)
		for _, d := range diags {
			t.Logf("diagnostic: %v", d)
		}
		// SQLite allows SELECT without FROM
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
		if len(q.Params) != 4 {
			t.Fatalf("expected 4 params (id, name, left, right), got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("AtSignParameter", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users WHERE email = @email AND status = @status`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		// Note: @ params may not be fully supported - check implementation
		// SQLite supports @VVV syntax but parser may not recognize them
		t.Logf("params found: %+v", q.Params)
	})

	t.Run("IsDistinctFrom", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users WHERE status IS DISTINCT FROM 'deleted'`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
	})

	t.Run("IsNotDistinctFrom", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users WHERE status IS NOT DISTINCT FROM ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("FullOuterJoin", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT u.id, p.title
FROM users u
FULL OUTER JOIN posts p ON p.user_id = u.id
WHERE u.status = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("RightOuterJoin", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT u.id, p.title
FROM users u
RIGHT JOIN posts p ON p.user_id = u.id
WHERE p.status = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("NaturalJoin", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users NATURAL JOIN profiles WHERE users.id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
	})

	t.Run("UsingClause", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users JOIN posts USING (user_id) WHERE users.status = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
	})

	t.Run("CompoundSelectUnion", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT id FROM users WHERE status = ?
UNION
SELECT id FROM admins WHERE role = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("CompoundSelectIntersect", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT id FROM users WHERE status = ?
INTERSECT
SELECT user_id FROM posts WHERE status = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("CompoundSelectExcept", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL: `SELECT id FROM users WHERE status = ?
EXCEPT
SELECT user_id FROM blocked WHERE active = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 2 {
			t.Fatalf("expected 2 params, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("ValuesClause", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `VALUES (1, 'one'), (2, 'two'), (3, ?)`,
		}
		q, diags := Parse(blk)
		// VALUES as standalone statement is SQLite-specific and may not be supported
		if len(diags) > 0 {
			t.Logf("expected diagnostic for VALUES: %v", diags[0].Message)
		}
		if q.Verb != VerbUnknown {
			t.Logf("verb: %v", q.Verb)
		}
	})

	t.Run("NullsFirstLast", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM users ORDER BY status ASC NULLS FIRST, created_at DESC NULLS LAST`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
	})

	t.Run("ExtractOperator", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT data->'name' AS name, data->>'email' AS email FROM users WHERE id = ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Columns) != 2 {
			t.Fatalf("expected 2 columns, got %d", len(q.Columns))
		}
	})

	t.Run("GlobOperator", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM files WHERE path NOT GLOB ?`,
		}
		q, diags := Parse(blk)
		if len(diags) > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if len(q.Params) != 1 {
			t.Fatalf("expected 1 param, got %d: %+v", len(q.Params), q.Params)
		}
	})

	t.Run("TableValuedFunction", func(t *testing.T) {
		blk := block.Block{
			Path: "query/test.sql",
			SQL:  `SELECT * FROM json_each(?) WHERE value > 10`,
		}
		q, diags := Parse(blk)
		for _, d := range diags {
			t.Logf("diagnostic: %v", d)
		}
		if q.Verb != VerbSelect {
			t.Errorf("expected VerbSelect, got %v", q.Verb)
		}
	})
}
