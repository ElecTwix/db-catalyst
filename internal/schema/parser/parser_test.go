package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

func TestParseBasicCatalog(t *testing.T) {
	catalog, diags := parseFixture(t, "basic.sql")
	if hasErrors(diags) {
		t.Fatalf("unexpected diagnostics: %s", formatDiagnostics(diags))
	}
	table := lookupTable(t, catalog, "users")
	if table.Doc != "Users table catalog entry" {
		t.Fatalf("users doc mismatch: got %q", table.Doc)
	}
	if !table.WithoutRowID {
		t.Fatalf("users without rowid flag not set")
	}
	if len(table.Columns) != 3 {
		t.Fatalf("users column count = %d, want 3", len(table.Columns))
	}
	profileCol := table.Columns[2]
	if profileCol.References == nil || profileCol.References.Table != "profiles" {
		t.Fatalf("profile_id should reference profiles, got %+v", profileCol.References)
	}
	view := lookupView(t, catalog, "active_users")
	if view.Doc != "Active users view" {
		t.Fatalf("view doc mismatch: got %q", view.Doc)
	}
	wantSQL := "SELECT u.id, u.email FROM users u WHERE u.profile_id IS NOT NULL"
	if view.SQL != wantSQL {
		t.Fatalf("view sql mismatch:\n got: %q\nwant: %q", view.SQL, wantSQL)
	}
	got := formatCatalog(catalog)
	want := strings.TrimSpace(`TABLE profiles doc="Profiles catalog entry" without_rowid=false
  COLUMN id type="INTEGER" notnull=true default=<nil>
  COLUMN bio type="TEXT" notnull=false default='none'
  PRIMARY KEY columns=[id]
TABLE users doc="Users table catalog entry" without_rowid=true
  COLUMN id type="INTEGER" notnull=true default=<nil>
  COLUMN email type="TEXT" notnull=true default=<nil>
  COLUMN profile_id type="INTEGER" notnull=false default=<nil> ref=profiles(id)
  PRIMARY KEY columns=[id]
  UNIQUE users_email_unique columns=[email]
  FOREIGN  columns=[profile_id] ref=profiles(id)
  INDEX idx_users_email unique=true columns=[email]
VIEW active_users doc="Active users view"
  SQL SELECT u.id, u.email FROM users u WHERE u.profile_id IS NOT NULL`)
	if got != want {
		t.Fatalf("catalog mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestParseDuplicateColumnDiagnostics(t *testing.T) {
	_, diags := parseFixture(t, "duplicate_columns.sql")
	if !containsMessage(diags, "duplicate column") {
		t.Fatalf("expected duplicate column diagnostic, got %s", formatDiagnostics(diags))
	}
}

func TestParseMissingForeignKeyDiagnostics(t *testing.T) {
	_, diags := parseFixture(t, "missing_fk.sql")
	if !containsMessage(diags, "unknown table") {
		t.Fatalf("expected foreign key table diagnostic, got %s", formatDiagnostics(diags))
	}
}

func TestAlterTableAddColumn(t *testing.T) {
	catalog, diags := parseFixture(t, "alter_add_column.sql")
	if hasErrors(diags) {
		t.Fatalf("unexpected diagnostics: %s", formatDiagnostics(diags))
	}
	table := lookupTable(t, catalog, "accounts")
	if len(table.Columns) != 2 {
		t.Fatalf("accounts columns = %d, want 2", len(table.Columns))
	}
	if table.Columns[1].Name != "email" {
		t.Fatalf("expected new column 'email', got %q", table.Columns[1].Name)
	}
	if !table.Columns[1].NotNull {
		t.Fatalf("ALTER TABLE column should preserve NOT NULL")
	}
}

func TestIndexUnknownTableDiagnostic(t *testing.T) {
	_, diags := parseFixture(t, "bad_index.sql")
	if !containsMessage(diags, "index") {
		t.Fatalf("expected index diagnostic, got %s", formatDiagnostics(diags))
	}
}

func BenchmarkParseBasic(b *testing.B) {
	path := fixturePath("basic.sql")
	src := readFile(b, path)
	tokens, err := tokenizer.Scan(path, src, true)
	if err != nil {
		b.Fatalf("scan failed: %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := Parse(path, tokens); err != nil {
			b.Fatalf("parse failed: %v", err)
		}
	}
}

func parseFixture(t *testing.T, filename string) (*model.Catalog, []Diagnostic) {
	t.Helper()
	path := fixturePath(filename)
	src := readFile(t, path)
	tokens, err := tokenizer.Scan(path, src, true)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	catalog, diags, err := Parse(path, tokens)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return catalog, diags
}

func fixturePath(name string) string {
	return filepath.Join("testdata", name)
}

func readFile(tb testing.TB, path string) []byte {
	tb.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		tb.Fatalf("read %s: %v", path, err)
	}
	return data
}

func hasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

func containsMessage(diags []Diagnostic, snippet string) bool {
	for _, d := range diags {
		if strings.Contains(d.Message, snippet) {
			return true
		}
	}
	return false
}

func formatDiagnostics(diags []Diagnostic) string {
	parts := make([]string, 0, len(diags))
	for _, d := range diags {
		parts = append(parts, d.Message)
	}
	return strings.Join(parts, "; ")
}

func formatCatalog(cat *model.Catalog) string {
	var b strings.Builder
	tables := make([]*model.Table, 0, len(cat.Tables))
	for _, table := range cat.Tables {
		tables = append(tables, table)
	}
	slices.SortFunc(tables, func(a, b *model.Table) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, table := range tables {
		model.SortUniqueKeys(table.UniqueKeys)
		model.SortForeignKeys(table.ForeignKeys)
		model.SortIndexes(table.Indexes)
		fmtf(&b, "TABLE %s doc=%q without_rowid=%v\n", table.Name, table.Doc, table.WithoutRowID)
		for _, col := range table.Columns {
			defaultText := "<nil>"
			if col.Default != nil {
				defaultText = col.Default.Text
			}
			refText := ""
			if col.References != nil {
				refText = col.References.Table + "(" + strings.Join(col.References.Columns, ",") + ")"
			}
			if refText == "" {
				fmtf(&b, "  COLUMN %s type=%q notnull=%v default=%s\n", col.Name, col.Type, col.NotNull, defaultText)
			} else {
				fmtf(&b, "  COLUMN %s type=%q notnull=%v default=%s ref=%s\n", col.Name, col.Type, col.NotNull, defaultText, refText)
			}
		}
		if table.PrimaryKey != nil {
			fmtf(&b, "  PRIMARY KEY columns=[%s]\n", strings.Join(table.PrimaryKey.Columns, ","))
		}
		for _, uk := range table.UniqueKeys {
			fmtf(&b, "  UNIQUE %s columns=[%s]\n", uk.Name, strings.Join(uk.Columns, ","))
		}
		for _, fk := range table.ForeignKeys {
			fmtf(&b, "  FOREIGN %s columns=[%s] ref=%s(%s)\n", fk.Name, strings.Join(fk.Columns, ","), fk.Ref.Table, strings.Join(fk.Ref.Columns, ","))
		}
		for _, idx := range table.Indexes {
			fmtf(&b, "  INDEX %s unique=%v columns=[%s]\n", idx.Name, idx.Unique, strings.Join(idx.Columns, ","))
		}
	}
	views := make([]*model.View, 0, len(cat.Views))
	for _, view := range cat.Views {
		views = append(views, view)
	}
	slices.SortFunc(views, func(a, b *model.View) int {
		return strings.Compare(a.Name, b.Name)
	})
	for _, view := range views {
		fmtf(&b, "VIEW %s doc=%q\n", view.Name, view.Doc)
		fmtf(&b, "  SQL %s\n", view.SQL)
	}
	return strings.TrimSpace(b.String())
}

func fmtf(b *strings.Builder, format string, args ...any) {
	_, _ = fmt.Fprintf(b, format, args...)
}

func lookupTable(t *testing.T, cat *model.Catalog, name string) *model.Table {
	t.Helper()
	for _, table := range cat.Tables {
		if strings.EqualFold(table.Name, name) {
			return table
		}
	}
	t.Fatalf("table %s not found", name)
	return nil
}

func lookupView(t *testing.T, cat *model.Catalog, name string) *model.View {
	t.Helper()
	for _, view := range cat.Views {
		if strings.EqualFold(view.Name, name) {
			return view
		}
	}
	t.Fatalf("view %s not found", name)
	return nil
}

func TestNewSchemaParser(t *testing.T) {
	t.Run("sqlite parser", func(t *testing.T) {
		p, err := NewSchemaParser("sqlite")
		if err != nil {
			t.Fatalf("NewSchemaParser(sqlite) error = %v", err)
		}
		if p == nil {
			t.Fatal("NewSchemaParser(sqlite) returned nil")
		}
	})

	t.Run("unsupported dialect", func(t *testing.T) {
		_, err := NewSchemaParser("postgresql")
		if err == nil {
			t.Fatal("expected error for unsupported dialect")
		}
	})
}

func TestSqliteParser_Parse(t *testing.T) {
	ctx := context.Background()
	p := &sqliteParser{}

	t.Run("simple table", func(t *testing.T) {
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

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		input := []byte("CREATE TABLE test (id INTEGER);")
		_, _, err := p.Parse(ctx, "test.sql", input)
		if err == nil {
			t.Error("expected context cancellation error")
		}
	})

	t.Run("invalid syntax", func(t *testing.T) {
		input := []byte("INVALID SQL")
		catalog, diags, err := p.Parse(ctx, "test.sql", input)
		// Should return diagnostics, not necessarily error
		if err != nil {
			t.Logf("Parse() returned error (may be expected): %v", err)
		}
		if catalog == nil {
			t.Error("expected catalog even for invalid syntax")
		}
		// May have diagnostics about invalid syntax
		_ = diags
	})
}

func TestParseVirtualTable(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantWarning    string
		wantTableCount int
	}{
		{
			name:           "fts5 virtual table",
			input:          "CREATE VIRTUAL TABLE posts_fts USING fts5(title, content, content=posts, content_rowid=id);",
			wantWarning:    "fts5",
			wantTableCount: 0,
		},
		{
			name: "fts5 with regular tables",
			input: `CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT, content TEXT);
CREATE VIRTUAL TABLE posts_fts USING fts5(title, content, content=posts, content_rowid=id);`,
			wantWarning:    "fts5",
			wantTableCount: 1,
		},
		{
			name:           "fts4 virtual table",
			input:          "CREATE VIRTUAL TABLE docs_fts USING fts4(content);",
			wantWarning:    "fts4",
			wantTableCount: 0,
		},
		{
			name:           "rtree virtual table",
			input:          "CREATE VIRTUAL TABLE rtree_index USING rtree(id, minX, maxX, minY, maxY);",
			wantWarning:    "rtree",
			wantTableCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog, diags, err := Parse("test.sql", mustScan(t, tt.input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !containsMessage(diags, "virtual tables are not fully supported") {
				t.Errorf("expected virtual table warning, got: %s", formatDiagnostics(diags))
			}
			if !containsMessage(diags, tt.wantWarning) {
				t.Errorf("expected warning to contain %q, got: %s", tt.wantWarning, formatDiagnostics(diags))
			}
			if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if len(catalog.Tables) != tt.wantTableCount {
				t.Errorf("expected %d tables, got %d", tt.wantTableCount, len(catalog.Tables))
			}
		})
	}
}

func TestParseTrigger(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantWarning    string
		wantTableCount int
	}{
		{
			name: "after insert trigger",
			input: `CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT);
CREATE TRIGGER posts_ai AFTER INSERT ON posts BEGIN
    INSERT INTO posts_fts(rowid, title) VALUES (new.id, new.title);
END;`,
			wantWarning:    "posts_ai",
			wantTableCount: 1,
		},
		{
			name: "before update trigger",
			input: `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT);
CREATE TRIGGER validate_email BEFORE UPDATE ON users BEGIN
    SELECT RAISE(ABORT, 'Invalid email') WHERE NEW.email NOT LIKE '%@%';
END;`,
			wantWarning:    "validate_email",
			wantTableCount: 1,
		},
		{
			name: "instead of trigger",
			input: `CREATE VIEW user_view AS SELECT id, email FROM users;
CREATE TRIGGER user_view_insert INSTEAD OF INSERT ON user_view BEGIN
    INSERT INTO users(id, email) VALUES (NEW.id, NEW.email);
END;`,
			wantWarning:    "user_view_insert",
			wantTableCount: 0,
		},
		{
			name: "trigger with if not exists",
			input: `CREATE TABLE items (id INTEGER PRIMARY KEY);
CREATE TRIGGER IF NOT EXISTS items_check BEFORE DELETE ON items BEGIN
    SELECT RAISE(FAIL, 'Cannot delete items') WHERE 1;
END;`,
			wantWarning:    "items_check",
			wantTableCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog, diags, err := Parse("test.sql", mustScan(t, tt.input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !containsMessage(diags, "triggers are not supported") {
				t.Errorf("expected trigger warning, got: %s", formatDiagnostics(diags))
			}
			if !containsMessage(diags, tt.wantWarning) {
				t.Errorf("expected warning to contain %q, got: %s", tt.wantWarning, formatDiagnostics(diags))
			}
			if hasErrors(diags) {
				t.Errorf("unexpected error diagnostics: %s", formatDiagnostics(diags))
			}
			if len(catalog.Tables) != tt.wantTableCount {
				t.Errorf("expected %d tables, got %d", tt.wantTableCount, len(catalog.Tables))
			}
		})
	}
}

func mustScan(t *testing.T, input string) []tokenizer.Token {
	t.Helper()
	tokens, err := tokenizer.Scan("test.sql", []byte(input), true)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	return tokens
}
