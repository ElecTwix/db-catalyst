package parser

import (
	"context"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// TestSqliteParserParseExtended covers the sqliteParser.Parse method comprehensively with additional cases
func TestSqliteParserParseExtended(t *testing.T) {
	p := &sqliteParser{}

	t.Run("simple table", func(t *testing.T) {
		ctx := context.Background()
		input := []byte("CREATE TABLE users (id INTEGER, name TEXT);")
		catalog, diags, err := p.Parse(ctx, "test.sql", input)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if len(diags) > 0 {
			t.Errorf("unexpected diagnostics: %+v", diags)
		}
		if catalog == nil {
			t.Fatal("Parse() returned nil catalog")
		}
		if _, ok := catalog.Tables["users"]; !ok {
			t.Error("expected 'users' table in catalog")
		}
	})

	t.Run("context cancellation before parse", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		input := []byte("CREATE TABLE test (id INTEGER);")
		_, _, err := p.Parse(ctx, "test.sql", input)
		if err == nil {
			t.Error("expected context cancellation error")
		}
		if !strings.Contains(err.Error(), "cancelled") {
			t.Errorf("expected 'cancelled' in error message, got: %v", err)
		}
	})

	t.Run("tokenization failure", func(_ *testing.T) {
		ctx := context.Background()
		// Invalid UTF-8 sequence that might cause tokenizer issues
		input := []byte{0x80, 0x81, 0x82}
		_, _, err := p.Parse(ctx, "test.sql", input)
		// Tokenizer may or may not error on this input
		// Just verify it doesn't panic
		_ = err
	})

	t.Run("empty input", func(t *testing.T) {
		ctx := context.Background()
		input := []byte("")
		catalog, diags, err := p.Parse(ctx, "test.sql", input)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if catalog == nil {
			t.Fatal("Parse() returned nil catalog for empty input")
		}
		if len(diags) > 0 {
			t.Errorf("unexpected diagnostics for empty input: %+v", diags)
		}
	})

	t.Run("multiple statements", func(t *testing.T) {
		ctx := context.Background()
		input := []byte(`
			CREATE TABLE users (id INTEGER PRIMARY KEY);
			CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id));
			CREATE INDEX idx_posts_user ON posts (user_id);
		`)
		catalog, diags, err := p.Parse(ctx, "test.sql", input)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if len(diags) > 0 {
			t.Errorf("unexpected diagnostics: %+v", diags)
		}
		if len(catalog.Tables) != 2 {
			t.Errorf("expected 2 tables, got %d", len(catalog.Tables))
		}
	})
}

// TestParseCreateTable covers parseCreateTable comprehensively
func TestParseCreateTable(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "basic table with all constraints",
			input: "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, email TEXT DEFAULT 'test', age INTEGER CHECK (age > 0));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if table == nil {
					t.Fatal("table 't' not found")
				}
				if len(table.Columns) != 4 {
					t.Errorf("expected 4 columns, got %d", len(table.Columns))
				}
				if table.PrimaryKey == nil {
					t.Error("expected primary key")
				}
				if len(table.UniqueKeys) != 1 {
					t.Errorf("expected 1 unique key, got %d", len(table.UniqueKeys))
				}
			},
		},
		{
			name:  "table with composite primary key",
			input: "CREATE TABLE t (a INTEGER, b INTEGER, PRIMARY KEY (a, b));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if table.PrimaryKey == nil {
					t.Fatal("expected primary key")
				}
				if len(table.PrimaryKey.Columns) != 2 {
					t.Errorf("expected 2 PK columns, got %d", len(table.PrimaryKey.Columns))
				}
			},
		},
		{
			name:  "table with named constraint",
			input: "CREATE TABLE t (id INTEGER, CONSTRAINT pk_id PRIMARY KEY (id), CONSTRAINT uq_name UNIQUE (id));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if table.PrimaryKey == nil || table.PrimaryKey.Name != "pk_id" {
					t.Error("expected named primary key constraint")
				}
				if len(table.UniqueKeys) != 1 || table.UniqueKeys[0].Name != "uq_name" {
					t.Error("expected named unique constraint")
				}
			},
		},
		{
			name:  "table with foreign key constraint",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (id INTEGER, parent_id INTEGER, FOREIGN KEY (parent_id) REFERENCES parent(id));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				child := cat.Tables["child"]
				if child == nil {
					t.Fatal("child table not found")
				}
				if len(child.ForeignKeys) != 1 {
					t.Errorf("expected 1 FK, got %d", len(child.ForeignKeys))
				}
			},
		},
		{
			name:  "table with multiple foreign key actions",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (id INTEGER, parent_id INTEGER, FOREIGN KEY (parent_id) REFERENCES parent(id) ON DELETE CASCADE ON UPDATE SET NULL);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				child := cat.Tables["child"]
				if child == nil {
					t.Fatal("child table not found")
				}
				if len(child.ForeignKeys) != 1 {
					t.Errorf("expected 1 FK, got %d", len(child.ForeignKeys))
				}
			},
		},
		{
			name:  "table with WITHOUT ROWID",
			input: "CREATE TABLE t (id INTEGER PRIMARY KEY) WITHOUT ROWID;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if !table.WithoutRowID {
					t.Error("expected WithoutRowID to be true")
				}
			},
		},
		{
			name:  "table with STRICT",
			input: "CREATE TABLE t (id INTEGER) STRICT;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if !table.Strict {
					t.Error("expected Strict to be true")
				}
			},
		},
		{
			name:  "table with IF NOT EXISTS",
			input: "CREATE TABLE IF NOT EXISTS t (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Tables["t"] == nil {
					t.Error("table 't' not found")
				}
			},
		},
		{
			name:  "table with schema-qualified name",
			input: "CREATE TABLE main.t (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// Should use the last part of the name
				if cat.Tables["t"] == nil {
					t.Error("table 't' not found")
				}
			},
		},
		{
			name:      "duplicate table",
			input:     "CREATE TABLE t (id INTEGER); CREATE TABLE t (name TEXT);",
			wantDiags: true,
			diagMsg:   "duplicate table",
		},
		{
			name:      "missing closing paren",
			input:     "CREATE TABLE t (id INTEGER",
			wantDiags: true,
		},
		{
			name:      "missing table name",
			input:     "CREATE TABLE (id INTEGER);",
			wantDiags: true,
		},
		{
			name:      "double primary key error",
			input:     "CREATE TABLE t (id INTEGER PRIMARY KEY, CONSTRAINT pk PRIMARY KEY (id));",
			wantDiags: true,
			diagMsg:   "already has a primary key",
		},
		{
			name:  "table with CHECK constraint",
			input: "CREATE TABLE t (id INTEGER, CHECK (id > 0));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// CHECK constraints are parsed but not stored in model
				if cat.Tables["t"] == nil {
					t.Error("table 't' not found")
				}
			},
		},
		{
			name:  "table with column check constraint",
			input: "CREATE TABLE t (id INTEGER CHECK (id > 0));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Tables["t"] == nil {
					t.Error("table 't' not found")
				}
			},
		},
		{
			name:  "table with autoincrement",
			input: "CREATE TABLE t (id INTEGER PRIMARY KEY AUTOINCREMENT);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if table.PrimaryKey == nil {
					t.Error("expected primary key")
				}
			},
		},
		{
			name:  "table with various data types",
			input: "CREATE TABLE t (a INTEGER, b TEXT, c REAL, d BLOB, e NUMERIC, f VARCHAR, g DECIMAL);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Columns) != 7 {
					t.Errorf("expected 7 columns, got %d", len(table.Columns))
				}
				// Check that types are preserved
				if table.Columns[5].Type != "VARCHAR" {
					t.Errorf("expected 'VARCHAR', got %q", table.Columns[5].Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseCreateIndex covers parseCreateIndex comprehensively
func TestParseCreateIndex(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "basic index",
			input: "CREATE TABLE t (id INTEGER, name TEXT); CREATE INDEX idx ON t (name);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes) != 1 {
					t.Errorf("expected 1 index, got %d", len(table.Indexes))
				}
				if table.Indexes[0].Name != "idx" {
					t.Errorf("expected index name 'idx', got %q", table.Indexes[0].Name)
				}
			},
		},
		{
			name:  "unique index",
			input: "CREATE TABLE t (id INTEGER, email TEXT); CREATE UNIQUE INDEX idx ON t (email);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes) != 1 {
					t.Fatal("expected 1 index")
				}
				if !table.Indexes[0].Unique {
					t.Error("expected unique index")
				}
			},
		},
		{
			name:  "composite index",
			input: "CREATE TABLE t (a INTEGER, b INTEGER); CREATE INDEX idx ON t (a, b);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes[0].Columns) != 2 {
					t.Errorf("expected 2 columns, got %d", len(table.Indexes[0].Columns))
				}
			},
		},
		{
			name:  "index with WHERE clause",
			input: "CREATE TABLE t (id INTEGER, active INTEGER); CREATE INDEX idx ON t (id) WHERE active = 1;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes) != 1 {
					t.Error("expected 1 index")
				}
			},
		},
		{
			name:  "index with IF NOT EXISTS",
			input: "CREATE TABLE t (id INTEGER); CREATE INDEX IF NOT EXISTS idx ON t (id);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes) != 1 {
					t.Error("expected 1 index")
				}
			},
		},
		{
			name:      "index on unknown table",
			input:     "CREATE INDEX idx ON unknown_table (id);",
			wantDiags: true,
			diagMsg:   "unknown table",
		},
		{
			name:      "index missing ON keyword",
			input:     "CREATE TABLE t (id INTEGER); CREATE INDEX idx t (id);",
			wantDiags: true,
			diagMsg:   "expected ON",
		},
		{
			name:  "index with column ordering",
			input: "CREATE TABLE t (id INTEGER, name TEXT); CREATE INDEX idx ON t (id ASC, name DESC);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes) != 1 {
					t.Error("expected 1 index")
				}
			},
		},
		{
			name:  "index with COLLATE",
			input: "CREATE TABLE t (name TEXT); CREATE INDEX idx ON t (name COLLATE NOCASE);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Indexes) != 1 {
					t.Error("expected 1 index")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseCreateView covers parseCreateView comprehensively
func TestParseCreateView(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "basic view",
			input: "CREATE VIEW v AS SELECT * FROM t;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				view := cat.Views["v"]
				if view == nil {
					t.Fatal("view 'v' not found")
				}
				if view.SQL != "SELECT * FROM t" {
					t.Errorf("expected SQL 'SELECT * FROM t', got %q", view.SQL)
				}
			},
		},
		{
			name:  "view with doc comment",
			input: "-- My view\nCREATE VIEW v AS SELECT 1;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				view := cat.Views["v"]
				if view == nil {
					t.Fatal("view 'v' not found")
				}
				if view.Doc != "My view" {
					t.Errorf("expected doc 'My view', got %q", view.Doc)
				}
			},
		},
		{
			name:  "view with IF NOT EXISTS",
			input: "CREATE VIEW IF NOT EXISTS v AS SELECT 1;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Views["v"] == nil {
					t.Error("view 'v' not found")
				}
			},
		},
		{
			name:      "duplicate view",
			input:     "CREATE VIEW v AS SELECT 1; CREATE VIEW v AS SELECT 2;",
			wantDiags: true,
			diagMsg:   "duplicate view",
		},
		{
			name:      "view missing AS",
			input:     "CREATE VIEW v SELECT 1;",
			wantDiags: true,
			diagMsg:   "expected AS",
		},
		{
			name:  "complex view",
			input: "CREATE VIEW v AS SELECT a.id, b.name FROM a JOIN b ON a.id = b.id WHERE a.active = 1;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				view := cat.Views["v"]
				if view == nil {
					t.Fatal("view 'v' not found")
				}
				if !strings.Contains(view.SQL, "JOIN") {
					t.Error("expected SQL to contain JOIN")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseAlter covers parseAlter comprehensively
func TestParseAlter(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "alter table add column",
			input: "CREATE TABLE t (id INTEGER); ALTER TABLE t ADD COLUMN name TEXT;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Columns) != 2 {
					t.Errorf("expected 2 columns, got %d", len(table.Columns))
				}
				if table.Columns[1].Name != "name" {
					t.Errorf("expected column 'name', got %q", table.Columns[1].Name)
				}
			},
		},
		{
			name:  "alter table add column without COLUMN keyword",
			input: "CREATE TABLE t (id INTEGER); ALTER TABLE t ADD name TEXT;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.Columns) != 2 {
					t.Errorf("expected 2 columns, got %d", len(table.Columns))
				}
			},
		},
		{
			name:  "alter table add column with constraints",
			input: "CREATE TABLE t (id INTEGER); ALTER TABLE t ADD COLUMN email TEXT NOT NULL DEFAULT 'test';",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				col := table.Columns[1]
				if col.Name != "email" {
					t.Errorf("expected 'email', got %q", col.Name)
				}
				if !col.NotNull {
					t.Error("expected NOT NULL")
				}
				if col.Default == nil || col.Default.Text != "'test'" {
					t.Errorf("expected default 'test', got %v", col.Default)
				}
			},
		},
		{
			name:      "alter table on unknown table",
			input:     "ALTER TABLE unknown ADD COLUMN name TEXT;",
			wantDiags: true,
			diagMsg:   "unknown table",
		},
		{
			name:      "alter table duplicate column",
			input:     "CREATE TABLE t (id INTEGER); ALTER TABLE t ADD COLUMN id TEXT;",
			wantDiags: true,
			diagMsg:   "already has column",
		},
		{
			name:      "alter table missing TABLE keyword",
			input:     "ALTER ADD COLUMN name TEXT;",
			wantDiags: true,
			diagMsg:   "expected TABLE",
		},
		{
			name:      "alter table missing ADD keyword",
			input:     "CREATE TABLE t (id INTEGER); ALTER TABLE t COLUMN name TEXT;",
			wantDiags: true,
			diagMsg:   "expected ADD",
		},
		{
			name:  "alter table add column with primary key",
			input: "CREATE TABLE t (id INTEGER); ALTER TABLE t ADD COLUMN pk INTEGER PRIMARY KEY;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if table.PrimaryKey == nil {
					t.Error("expected primary key to be added")
				}
			},
		},
		{
			name:      "alter table add column with duplicate primary key",
			input:     "CREATE TABLE t (id INTEGER PRIMARY KEY); ALTER TABLE t ADD COLUMN pk INTEGER PRIMARY KEY;",
			wantDiags: true,
			diagMsg:   "already has a primary key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseColumnDefinition covers parseColumnDefinition comprehensively
func TestParseColumnDefinition(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "column with all constraints",
			input: "CREATE TABLE t (id INTEGER PRIMARY KEY NOT NULL UNIQUE DEFAULT 1 CHECK (id > 0));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Name != "id" {
					t.Errorf("expected 'id', got %q", col.Name)
				}
				if col.Type != "INTEGER" {
					t.Errorf("expected type 'INTEGER', got %q", col.Type)
				}
				if !col.NotNull {
					t.Error("expected NOT NULL")
				}
				if col.Default == nil || col.Default.Text != "1" {
					t.Errorf("expected default 1, got %v", col.Default)
				}
			},
		},
		{
			name:  "column with string default",
			input: "CREATE TABLE t (name TEXT DEFAULT 'hello world');",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil || col.Default.Text != "'hello world'" {
					t.Errorf("expected default 'hello world', got %v", col.Default)
				}
				if col.Default.Kind != model.ValueKindString {
					t.Errorf("expected string kind, got %v", col.Default.Kind)
				}
			},
		},
		{
			name:  "column with numeric default",
			input: "CREATE TABLE t (count INTEGER DEFAULT 42);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil || col.Default.Text != "42" {
					t.Errorf("expected default 42, got %v", col.Default)
				}
				if col.Default.Kind != model.ValueKindNumber {
					t.Errorf("expected number kind, got %v", col.Default.Kind)
				}
			},
		},
		{
			name:  "column with blob default",
			input: "CREATE TABLE t (data BLOB DEFAULT X'ABCD');",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil {
					t.Fatal("expected default")
				}
				if col.Default.Kind != model.ValueKindBlob {
					t.Errorf("expected blob kind, got %v", col.Default.Kind)
				}
			},
		},
		{
			name:  "column with keyword default",
			input: "CREATE TABLE t (created TEXT DEFAULT CURRENT_TIMESTAMP);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil {
					t.Fatal("expected default")
				}
				if col.Default.Kind != model.ValueKindKeyword {
					t.Errorf("expected keyword kind, got %v", col.Default.Kind)
				}
			},
		},
		{
			name:  "column with expression default",
			input: "CREATE TABLE t (value INTEGER DEFAULT (1 + 2));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil {
					t.Fatal("expected default")
				}
				if !strings.Contains(col.Default.Text, "1") {
					t.Errorf("expected default to contain '1', got %q", col.Default.Text)
				}
			},
		},
		{
			name:  "column with inline foreign key",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (parent_id INTEGER REFERENCES parent(id));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				child := cat.Tables["child"]
				col := child.Columns[0]
				if col.References == nil {
					t.Fatal("expected references")
				}
				if col.References.Table != "parent" {
					t.Errorf("expected ref table 'parent', got %q", col.References.Table)
				}
				if len(child.ForeignKeys) != 1 {
					t.Errorf("expected 1 FK, got %d", len(child.ForeignKeys))
				}
			},
		},
		{
			name:  "column with inline foreign key no columns",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (parent_id INTEGER REFERENCES parent);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				child := cat.Tables["child"]
				if len(child.ForeignKeys) != 1 {
					t.Errorf("expected 1 FK, got %d", len(child.ForeignKeys))
				}
				if len(child.ForeignKeys[0].Ref.Columns) != 0 {
					t.Errorf("expected 0 ref columns, got %d", len(child.ForeignKeys[0].Ref.Columns))
				}
			},
		},
		{
			name:      "column with incomplete NOT NULL",
			input:     "CREATE TABLE t (name TEXT NOT);",
			wantDiags: true,
			diagMsg:   "expected NULL after NOT",
		},
		{
			name:      "column with incomplete PRIMARY KEY",
			input:     "CREATE TABLE t (id INTEGER PRIMARY);",
			wantDiags: true,
			diagMsg:   "expected KEY after PRIMARY",
		},
		{
			name:  "column no type",
			input: "CREATE TABLE t (id);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Type != "" {
					t.Errorf("expected empty type, got %q", col.Type)
				}
			},
		},
		{
			name:  "column with unsupported constraint",
			input: "CREATE TABLE t (id INTEGER GENERATED ALWAYS AS IDENTITY);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// Should parse but with warning diagnostic
				if cat.Tables["t"] == nil {
					t.Error("table not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseTableConstraint covers parseTableConstraint comprehensively
func TestParseTableConstraint(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "named primary key constraint",
			input: "CREATE TABLE t (a INTEGER, b INTEGER, CONSTRAINT pk_ab PRIMARY KEY (a, b));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if table.PrimaryKey == nil {
					t.Fatal("expected primary key")
				}
				if table.PrimaryKey.Name != "pk_ab" {
					t.Errorf("expected name 'pk_ab', got %q", table.PrimaryKey.Name)
				}
				if len(table.PrimaryKey.Columns) != 2 {
					t.Errorf("expected 2 columns, got %d", len(table.PrimaryKey.Columns))
				}
			},
		},
		{
			name:  "named unique constraint",
			input: "CREATE TABLE t (a INTEGER, b INTEGER, CONSTRAINT uq_ab UNIQUE (a, b));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.UniqueKeys) != 1 {
					t.Fatalf("expected 1 unique key, got %d", len(table.UniqueKeys))
				}
				if table.UniqueKeys[0].Name != "uq_ab" {
					t.Errorf("expected name 'uq_ab', got %q", table.UniqueKeys[0].Name)
				}
			},
		},
		{
			name:  "named foreign key constraint",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY, id2 INTEGER); CREATE TABLE child (a INTEGER, b INTEGER, CONSTRAINT fk_parent FOREIGN KEY (a, b) REFERENCES parent(id, id2));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				child := cat.Tables["child"]
				if len(child.ForeignKeys) != 1 {
					t.Fatalf("expected 1 FK, got %d", len(child.ForeignKeys))
				}
				if child.ForeignKeys[0].Name != "fk_parent" {
					t.Errorf("expected name 'fk_parent', got %q", child.ForeignKeys[0].Name)
				}
			},
		},
		{
			name:  "check constraint",
			input: "CREATE TABLE t (id INTEGER, CHECK (id > 0));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// CHECK constraints are parsed but not stored
				if cat.Tables["t"] == nil {
					t.Error("table not found")
				}
			},
		},
		{
			name:      "foreign key missing KEY keyword",
			input:     "CREATE TABLE t (id INTEGER, FOREIGN (id) REFERENCES parent(id));",
			wantDiags: true,
			diagMsg:   "expected KEY after FOREIGN",
		},
		{
			name:      "foreign key missing REFERENCES",
			input:     "CREATE TABLE t (id INTEGER, FOREIGN KEY (id));",
			wantDiags: true,
			diagMsg:   "expected REFERENCES",
		},
		{
			name:      "constraint without keyword",
			input:     "CREATE TABLE t (id INTEGER, CONSTRAINT foo);",
			wantDiags: true,
			diagMsg:   "expected PRIMARY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseCreate covers parseCreate with edge cases
func TestParseCreate(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "create temp table with warning",
			input: "CREATE TEMP TABLE t (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Tables["t"] == nil {
					t.Error("table 't' not found")
				}
			},
		},
		{
			name:  "create temporary table with warning",
			input: "CREATE TEMPORARY TABLE t (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Tables["t"] == nil {
					t.Error("table 't' not found")
				}
			},
		},
		{
			name:      "unique without index",
			input:     "CREATE UNIQUE TABLE t (id INTEGER);",
			wantDiags: true,
			diagMsg:   "UNIQUE modifier only supported for indexes",
		},
		{
			name:      "create unsupported target",
			input:     "CREATE TRIGGER trg AFTER INSERT ON t BEGIN SELECT 1; END;",
			wantDiags: true,
			diagMsg:   "unsupported CREATE",
		},
		{
			name:      "create without keyword",
			input:     "CREATE 123;",
			wantDiags: true,
			diagMsg:   "expected TABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseEdgeCases covers edge cases and error recovery
func TestParseEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "empty tokens",
			input: "",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat == nil {
					t.Error("expected non-nil catalog")
				}
			},
		},
		{
			name:  "only semicolons",
			input: ";;;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat == nil {
					t.Error("expected non-nil catalog")
				}
			},
		},
		{
			name:  "doc comment without following statement",
			input: "-- Orphan comment",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat == nil {
					t.Error("expected non-nil catalog")
				}
			},
		},
		{
			name:      "unsupported statement",
			input:     "DROP TABLE t;",
			wantDiags: true,
			diagMsg:   "unsupported statement",
		},
		{
			name:      "unexpected symbol",
			input:     "@#$;",
			wantDiags: true,
			diagMsg:   "unexpected",
		},
		{
			name:  "multiple doc comments",
			input: "-- First\n-- Second\nCREATE TABLE t (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// Doc comments are accumulated with newlines
				table := cat.Tables["t"]
				if table == nil {
					t.Fatal("table not found")
				}
				// Both doc comments are preserved
				if !strings.Contains(table.Doc, "First") || !strings.Contains(table.Doc, "Second") {
					t.Errorf("expected doc to contain both 'First' and 'Second', got %q", table.Doc)
				}
			},
		},
		{
			name: "complex schema",
			input: `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					email TEXT NOT NULL UNIQUE,
					created_at TEXT DEFAULT CURRENT_TIMESTAMP
				) STRICT;
				
				CREATE TABLE posts (
					id INTEGER PRIMARY KEY,
					user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					title TEXT NOT NULL,
					body TEXT,
					published INTEGER DEFAULT 0,
					CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
				);
				
				CREATE INDEX idx_posts_user ON posts (user_id);
				CREATE INDEX idx_posts_published ON posts (published) WHERE published = 1;
				
				CREATE VIEW active_posts AS 
				SELECT p.id, p.title, u.email 
				FROM posts p 
				JOIN users u ON p.user_id = u.id 
				WHERE p.published = 1;
			`,
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables) != 2 {
					t.Errorf("expected 2 tables, got %d", len(cat.Tables))
				}
				if len(cat.Views) != 1 {
					t.Errorf("expected 1 view, got %d", len(cat.Views))
				}
				posts := cat.Tables["posts"]
				if len(posts.Indexes) != 2 {
					t.Errorf("expected 2 indexes on posts, got %d", len(posts.Indexes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestValidation covers validation logic comprehensively
func TestValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDiags bool
		diagMsg   string
	}{
		{
			name:      "primary key references unknown column",
			input:     "CREATE TABLE t (id INTEGER, PRIMARY KEY (unknown_col));",
			wantDiags: true,
			diagMsg:   "unknown column",
		},
		{
			name:      "unique constraint references unknown column",
			input:     "CREATE TABLE t (id INTEGER, UNIQUE (unknown_col));",
			wantDiags: true,
			diagMsg:   "unknown column",
		},
		{
			name:      "foreign key references unknown column in source table",
			input:     "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (id INTEGER, FOREIGN KEY (unknown) REFERENCES parent(id));",
			wantDiags: true,
			diagMsg:   "unknown column",
		},
		{
			name:      "foreign key references unknown table",
			input:     "CREATE TABLE t (id INTEGER, FOREIGN KEY (id) REFERENCES unknown(id));",
			wantDiags: true,
			diagMsg:   "unknown table",
		},
		{
			name:      "foreign key references unknown column in target table",
			input:     "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (parent_id INTEGER, FOREIGN KEY (parent_id) REFERENCES parent(unknown));",
			wantDiags: true,
			diagMsg:   "unknown column",
		},
		{
			name:      "index references unknown column",
			input:     "CREATE TABLE t (id INTEGER); CREATE INDEX idx ON t (unknown_col);",
			wantDiags: true,
			diagMsg:   "unknown column",
		},
		{
			name:  "valid foreign key with no target columns",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (parent_id INTEGER REFERENCES parent);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			_, diags, err := Parse("test.sql", tokens)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
		})
	}
}

// TestHelperMethods covers various helper methods
func TestHelperMethods(t *testing.T) {
	t.Run("canonicalName", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"TableName", "tablename"},
			{"TABLE_NAME", "table_name"},
			{"mixedCase", "mixedcase"},
			{"", ""},
		}
		for _, tt := range tests {
			got := canonicalName(tt.input)
			if got != tt.want {
				t.Errorf("canonicalName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("needsSpace", func(t *testing.T) {
		tests := []struct {
			prev, next string
			want       bool
		}{
			{"hello", "world", true},
			{"hello", ",", false},
			{"(", "world", false},
			{"table", ".", false},
			{".", "column", false},
			{"", "hello", false},
			{"hello", "", false},
		}
		for _, tt := range tests {
			got := needsSpace(tt.prev, tt.next)
			if got != tt.want {
				t.Errorf("needsSpace(%q, %q) = %v, want %v", tt.prev, tt.next, got, tt.want)
			}
		}
	})

	t.Run("rebuildSQL", func(t *testing.T) {
		tests := []struct {
			name   string
			tokens []tokenizer.Token
			want   string
		}{
			{
				name:   "empty",
				tokens: nil,
				want:   "",
			},
			{
				name: "simple select",
				tokens: []tokenizer.Token{
					{Kind: tokenizer.KindKeyword, Text: "SELECT"},
					{Kind: tokenizer.KindSymbol, Text: "*"},
					{Kind: tokenizer.KindKeyword, Text: "FROM"},
					{Kind: tokenizer.KindIdentifier, Text: "t"},
				},
				want: "SELECT * FROM t",
			},
			{
				name: "function call",
				tokens: []tokenizer.Token{
					{Kind: tokenizer.KindIdentifier, Text: "func"},
					{Kind: tokenizer.KindSymbol, Text: "("},
					{Kind: tokenizer.KindNumber, Text: "1"},
					{Kind: tokenizer.KindSymbol, Text: ","},
					{Kind: tokenizer.KindNumber, Text: "2"},
					{Kind: tokenizer.KindSymbol, Text: ")"},
				},
				want: "func (1, 2)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := rebuildSQL(tt.tokens)
				if got != tt.want {
					t.Errorf("rebuildSQL() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("isClauseBoundaryKeyword", func(t *testing.T) {
		// Test all boundary keywords
		keywords := []string{
			"CONSTRAINT", "PRIMARY", "UNIQUE", "FOREIGN", "CHECK",
			"REFERENCES", "DEFAULT", "NOT", "GENERATED",
		}
		for _, kw := range keywords {
			if !isClauseBoundaryKeyword(kw) {
				t.Errorf("isClauseBoundaryKeyword(%q) = false, want true", kw)
			}
		}
		// Test non-boundary keyword
		if isClauseBoundaryKeyword("SELECT") {
			t.Error("isClauseBoundaryKeyword(SELECT) = true, want false")
		}
	})
}

// TestParserState covers Parser state management
func TestParserState(t *testing.T) {
	t.Run("empty token handling", func(t *testing.T) {
		p := &Parser{
			tokens:  []tokenizer.Token{},
			catalog: model.NewCatalog(),
			path:    "test.sql",
		}
		// Should handle empty tokens gracefully
		if !p.isEOF() {
			t.Error("expected EOF for empty tokens")
		}
	})

	t.Run("previous token edge case", func(t *testing.T) {
		p := &Parser{
			tokens: []tokenizer.Token{
				{Kind: tokenizer.KindKeyword, Text: "SELECT"},
			},
			catalog: model.NewCatalog(),
			path:    "test.sql",
		}
		// At position 0, previous should return empty token
		prev := p.previous()
		if prev.Kind != tokenizer.KindInvalid {
			t.Errorf("expected invalid token at pos 0, got %v", prev.Kind)
		}
	})

	t.Run("advance beyond bounds", func(t *testing.T) {
		p := &Parser{
			tokens: []tokenizer.Token{
				{Kind: tokenizer.KindKeyword, Text: "SELECT"},
			},
			catalog: model.NewCatalog(),
			path:    "test.sql",
		}
		p.advance()
		p.advance() // Should not panic
		p.advance() // Should not panic
		if !p.isEOF() {
			t.Error("expected EOF after advancing beyond bounds")
		}
	})
}

// TestDiagnosticMethods covers diagnostic creation
func TestDiagnosticMethods(t *testing.T) {
	t.Run("addDiagToken with empty file", func(t *testing.T) {
		p := &Parser{
			tokens:  []tokenizer.Token{},
			catalog: model.NewCatalog(),
			path:    "test.sql",
		}
		tok := tokenizer.Token{
			Kind:   tokenizer.KindKeyword,
			Text:   "SELECT",
			File:   "",
			Line:   0,
			Column: 0,
		}
		p.addDiagToken(tok, SeverityError, "test message")
		if len(p.diagnostics) != 1 {
			t.Fatal("expected 1 diagnostic")
		}
		diag := p.diagnostics[0]
		if diag.Path != "test.sql" {
			t.Errorf("expected path 'test.sql', got %q", diag.Path)
		}
		if diag.Line != 1 {
			t.Errorf("expected line 1, got %d", diag.Line)
		}
		if diag.Column != 1 {
			t.Errorf("expected column 1, got %d", diag.Column)
		}
	})

	t.Run("addDiagSpan with empty span", func(t *testing.T) {
		p := &Parser{
			tokens:  []tokenizer.Token{},
			catalog: model.NewCatalog(),
			path:    "test.sql",
		}
		span := tokenizer.Span{
			File:        "",
			StartLine:   0,
			StartColumn: 0,
			EndLine:     0,
			EndColumn:   0,
		}
		p.addDiagSpan(span, SeverityWarning, "test warning")
		if len(p.diagnostics) != 1 {
			t.Fatal("expected 1 diagnostic")
		}
		diag := p.diagnostics[0]
		if diag.Path != "test.sql" {
			t.Errorf("expected path 'test.sql', got %q", diag.Path)
		}
		if diag.Line != 1 {
			t.Errorf("expected line 1, got %d", diag.Line)
		}
	})
}

// TestSyncAndRecovery covers error recovery mechanisms
func TestSyncAndRecovery(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDiags  bool
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:      "recovery after semicolon in column list",
			input:     "CREATE TABLE t1 (id INTEGER); CREATE TABLE t2 (name TEXT);",
			wantDiags: false,
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// Both tables should be parsed
				if cat.Tables["t1"] == nil {
					t.Error("expected t1 table")
				}
				if cat.Tables["t2"] == nil {
					t.Error("expected t2 table")
				}
			},
		},
		{
			name:  "sync after unsupported statement",
			input: "DROP TABLE t1; CREATE TABLE t2 (name TEXT);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// t2 should still be parsed after error recovery
				if cat.Tables["t2"] == nil {
					t.Error("expected t2 table to be parsed after error recovery")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, _, _ := Parse("test.sql", tokens)
			if catalog != nil && tt.validateFn != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseForeignKeyRef covers parseForeignKeyRef edge cases
func TestParseForeignKeyRef(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "foreign key with multiple columns",
			input: "CREATE TABLE parent (a INTEGER, b INTEGER, PRIMARY KEY (a, b)); CREATE TABLE child (a INTEGER, b INTEGER, FOREIGN KEY (a, b) REFERENCES parent(a, b));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				child := cat.Tables["child"]
				if len(child.ForeignKeys) != 1 {
					t.Fatalf("expected 1 FK, got %d", len(child.ForeignKeys))
				}
				fk := child.ForeignKeys[0]
				if len(fk.Ref.Columns) != 2 {
					t.Errorf("expected 2 ref columns, got %d", len(fk.Ref.Columns))
				}
			},
		},
		{
			name:  "foreign key self-reference",
			input: "CREATE TABLE t (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES t(id));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				table := cat.Tables["t"]
				if len(table.ForeignKeys) != 1 {
					t.Fatalf("expected 1 FK, got %d", len(table.ForeignKeys))
				}
				if table.ForeignKeys[0].Ref.Table != "t" {
					t.Errorf("expected self-reference, got %q", table.ForeignKeys[0].Ref.Table)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseColumnNameList covers parseColumnNameList edge cases
func TestParseColumnNameList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDiags bool
		diagMsg   string
	}{
		{
			name:  "single column",
			input: "CREATE TABLE t (id INTEGER PRIMARY KEY);",
		},
		{
			name:  "empty column list in constraint",
			input: "CREATE TABLE t (id INTEGER, PRIMARY KEY ());",
		},
		{
			name:      "missing opening paren",
			input:     "CREATE TABLE t (id INTEGER, PRIMARY KEY id);",
			wantDiags: true,
			diagMsg:   "expected (",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			_, diags, err := Parse("test.sql", tokens)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			}
		})
	}
}

// TestParseObjectName covers parseObjectName edge cases
func TestParseObjectName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "schema qualified table name",
			input: "CREATE TABLE myschema.mytable (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// Should use the table name, not schema
				if cat.Tables["mytable"] == nil {
					t.Error("expected 'mytable' table")
				}
			},
		},
		{
			name:  "quoted identifier",
			input: "CREATE TABLE \"My Table\" (id INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				// Should use the unquoted name
				if cat.Tables["my table"] == nil {
					t.Error("expected 'my table' table")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestSeverityConstants ensures severity constants work correctly
func TestSeverityConstants(t *testing.T) {
	if SeverityError != 0 {
		t.Errorf("SeverityError = %d, want 0", SeverityError)
	}
	if SeverityWarning != 1 {
		t.Errorf("SeverityWarning = %d, want 1", SeverityWarning)
	}
}

// TestParseWithEOFToken ensures EOF token handling works
func TestParseWithEOFToken(t *testing.T) {
	// Test with tokens that already have EOF
	tokens := []tokenizer.Token{
		{Kind: tokenizer.KindKeyword, Text: "CREATE", File: "test.sql", Line: 1, Column: 1},
		{Kind: tokenizer.KindKeyword, Text: "TABLE", File: "test.sql", Line: 1, Column: 8},
		{Kind: tokenizer.KindIdentifier, Text: "t", File: "test.sql", Line: 1, Column: 14},
		{Kind: tokenizer.KindSymbol, Text: "(", File: "test.sql", Line: 1, Column: 15},
		{Kind: tokenizer.KindIdentifier, Text: "id", File: "test.sql", Line: 1, Column: 16},
		{Kind: tokenizer.KindIdentifier, Text: "INTEGER", File: "test.sql", Line: 1, Column: 19},
		{Kind: tokenizer.KindSymbol, Text: ")", File: "test.sql", Line: 1, Column: 26},
		{Kind: tokenizer.KindEOF, File: "test.sql", Line: 1, Column: 27},
	}
	catalog, diags, err := Parse("test.sql", tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if hasErrors(diags) {
		t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
	}
	if catalog.Tables["t"] == nil {
		t.Error("expected 't' table")
	}
}

// TestAdditionalEdgeCases covers additional edge cases for higher coverage
func TestAdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDiags  bool
		diagMsg    string
		validateFn func(t *testing.T, cat *model.Catalog)
	}{
		{
			name:  "column with complex default expression",
			input: "CREATE TABLE t (id INTEGER DEFAULT (CASE WHEN 1=1 THEN 2 ELSE 3 END));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil {
					t.Error("expected default value")
				}
			},
		},
		{
			name:  "index with complex WHERE clause",
			input: "CREATE TABLE t (a INTEGER, b INTEGER); CREATE INDEX idx ON t (a) WHERE b > 0 AND a IS NOT NULL;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["t"].Indexes) != 1 {
					t.Error("expected 1 index")
				}
			},
		},
		{
			name:  "foreign key with DEFERRABLE",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (pid INTEGER REFERENCES parent(id) DEFERRABLE INITIALLY DEFERRED);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["child"].ForeignKeys) != 1 {
					t.Error("expected 1 foreign key")
				}
			},
		},
		{
			name:  "column with COLLATE in index",
			input: "CREATE TABLE t (name TEXT); CREATE INDEX idx ON t (name COLLATE NOCASE ASC);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["t"].Indexes) != 1 {
					t.Error("expected 1 index")
				}
			},
		},
		{
			name:  "table with multiple constraints same type",
			input: "CREATE TABLE t (id INTEGER, a INTEGER, b INTEGER, UNIQUE (a), UNIQUE (b));",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["t"].UniqueKeys) != 2 {
					t.Errorf("expected 2 unique keys, got %d", len(cat.Tables["t"].UniqueKeys))
				}
			},
		},
		{
			name:  "alter table add column with unique constraint",
			input: "CREATE TABLE t (id INTEGER); ALTER TABLE t ADD COLUMN email TEXT UNIQUE;",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["t"].UniqueKeys) != 1 {
					t.Errorf("expected 1 unique key, got %d", len(cat.Tables["t"].UniqueKeys))
				}
			},
		},
		{
			name:  "alter table add column with foreign key",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE t (id INTEGER); ALTER TABLE t ADD COLUMN pid INTEGER REFERENCES parent(id);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["t"].ForeignKeys) != 1 {
					t.Errorf("expected 1 foreign key, got %d", len(cat.Tables["t"].ForeignKeys))
				}
			},
		},
		{
			name:  "view with complex subquery",
			input: "CREATE TABLE t (id INTEGER, val INTEGER); CREATE VIEW v AS SELECT * FROM t WHERE val > (SELECT AVG(val) FROM t);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Views["v"] == nil {
					t.Error("expected view 'v'")
				}
			},
		},
		{
			name:  "table with blob default",
			input: "CREATE TABLE t (data BLOB DEFAULT X'00');",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				col := cat.Tables["t"].Columns[0]
				if col.Default == nil || col.Default.Kind != model.ValueKindBlob {
					t.Error("expected blob default")
				}
			},
		},
		{
			name:  "multiple indexes on same table",
			input: "CREATE TABLE t (a INTEGER, b INTEGER, c INTEGER); CREATE INDEX idx_a ON t (a); CREATE INDEX idx_b ON t (b); CREATE INDEX idx_c ON t (c);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["t"].Indexes) != 3 {
					t.Errorf("expected 3 indexes, got %d", len(cat.Tables["t"].Indexes))
				}
			},
		},
		{
			name:  "foreign key with multiple actions",
			input: "CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (pid INTEGER, FOREIGN KEY (pid) REFERENCES parent(id) ON DELETE CASCADE ON UPDATE NO ACTION ON DELETE SET NULL);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if len(cat.Tables["child"].ForeignKeys) != 1 {
					t.Error("expected 1 foreign key")
				}
			},
		},
		{
			name:  "table with quoted identifiers",
			input: "CREATE TABLE [my table] ([my id] INTEGER PRIMARY KEY);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Tables["my table"] == nil {
					t.Error("expected 'my table'")
				}
			},
		},
		{
			name:  "table with backtick identifiers",
			input: "CREATE TABLE `my table` (`my id` INTEGER);",
			//nolint:thelper // Anonymous function in test table
			validateFn: func(t *testing.T, cat *model.Catalog) {
				if cat.Tables["my table"] == nil {
					t.Error("expected 'my table'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			catalog, diags, err := Parse("test.sql", tokens)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			} else if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if tt.validateFn != nil && catalog != nil {
				tt.validateFn(t, catalog)
			}
		})
	}
}

// TestParseErrors covers error handling paths
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		diagMsg   string
		wantDiags bool
	}{
		{
			name:      "create index missing ON keyword",
			input:     "CREATE TABLE t (id INTEGER); CREATE INDEX idx t (id);",
			diagMsg:   "expected ON",
			wantDiags: true,
		},
		{
			name:      "create view missing AS keyword",
			input:     "CREATE VIEW v SELECT 1;",
			diagMsg:   "expected AS",
			wantDiags: true,
		},
		{
			name:      "foreign key ref missing table",
			input:     "CREATE TABLE t (id INTEGER REFERENCES (id));",
			diagMsg:   "expected identifier",
			wantDiags: true,
		},
		{
			name:      "column list missing closing paren",
			input:     "CREATE TABLE t (id INTEGER PRIMARY KEY (a);",
			wantDiags: true,
		},
		{
			name:      "alter table missing ADD keyword",
			input:     "CREATE TABLE t (id INTEGER); ALTER TABLE t COLUMN name TEXT;",
			diagMsg:   "expected ADD",
			wantDiags: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Scan("test.sql", []byte(tt.input), true)
			if err != nil {
				t.Fatalf("tokenization error: %v", err)
			}
			_, diags, _ := Parse("test.sql", tokens)
			if tt.wantDiags {
				found := false
				for _, d := range diags {
					if tt.diagMsg == "" || strings.Contains(d.Message, tt.diagMsg) {
						found = true
						break
					}
				}
				if !found && tt.diagMsg != "" {
					t.Errorf("expected diagnostic containing %q, got: %v", tt.diagMsg, formatDiagnostics(diags))
				}
			}
		})
	}
}
