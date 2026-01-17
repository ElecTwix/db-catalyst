package sql_test

import (
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/codegen/sql"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestGenerateSQLiteSchema(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: true},
					{Name: "created_at", Type: "TEXT", Default: &model.Value{Kind: model.ValueKindKeyword, Text: "CURRENT_TIMESTAMP"}},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
			},
			"posts": {
				Name: "posts",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "user_id", Type: "INTEGER", NotNull: true},
					{Name: "title", Type: "TEXT", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []*model.ForeignKey{
					{
						Columns: []string{"user_id"},
						Ref: model.ForeignKeyRef{
							Table:   "users",
							Columns: []string{"id"},
						},
					},
				},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectSQLite})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)
	if files[0].Path != "schema.gen.sql" {
		t.Errorf("expected path schema.gen.sql, got %s", files[0].Path)
	}

	if !strings.Contains(content, "CREATE TABLE IF NOT EXISTS users") {
		t.Error("expected CREATE TABLE IF NOT EXISTS users")
	}
	if !strings.Contains(content, "CREATE TABLE IF NOT EXISTS posts") {
		t.Error("expected CREATE TABLE IF NOT EXISTS posts")
	}
	if !strings.Contains(content, "FOREIGN KEY") {
		t.Error("expected FOREIGN KEY")
	}
	if !strings.Contains(content, "REFERENCES users") {
		t.Error("expected REFERENCES users")
	}
}

func TestGenerateMySQLSchema(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "VARCHAR(255)", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectMySQL})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "DROP TABLE IF EXISTS users") {
		t.Error("expected DROP TABLE IF EXISTS users")
	}
	if !strings.Contains(content, "ENGINE=InnoDB") {
		t.Error("expected ENGINE=InnoDB")
	}
	if !strings.Contains(content, "utf8mb4") {
		t.Error("expected utf8mb4 charset")
	}
}

func TestGenerateSQLiteWithIndexes(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
				Indexes: []*model.Index{
					{Name: "idx_users_email", Columns: []string{"email"}},
				},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectSQLite})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "CREATE INDEX IF NOT EXISTS idx_users_email") {
		t.Error("expected CREATE INDEX IF NOT EXISTS idx_users_email")
	}
}

func TestGenerateSQLiteStrictTable(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name:   "users",
				Strict: true,
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectSQLite})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "STRICT") {
		t.Error("expected STRICT table option")
	}
}

func TestGenerateViews(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
			},
		},
		Views: map[string]*model.View{
			"active_users": {
				Name: "active_users",
				SQL:  "SELECT * FROM users WHERE active = 1",
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectSQLite})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files (schema + view), got %d", len(files))
	}

	var viewFile *sql.File
	for i := range files {
		if files[i].Path == "views/active_users.sql" {
			viewFile = &files[i]
			break
		}
	}

	if viewFile == nil {
		t.Fatal("expected view file at views/active_users.sql")
	}

	content := string(viewFile.Content)
	if !strings.Contains(content, "CREATE VIEW active_users") {
		t.Error("expected CREATE VIEW active_users")
	}
}

func TestDefaultToSQLite(t *testing.T) {
	g := sql.New(sql.Options{})
	if g == nil {
		t.Fatal("New() returned nil")
	}
}

func TestEmptyCatalog(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{},
		Views:  map[string]*model.View{},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectSQLite})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 0 {
		t.Fatalf("expected 0 files for empty catalog, got %d", len(files))
	}
}

func TestGenerateMySQLWithUniqueIndex(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
				Indexes: []*model.Index{
					{Name: "uniq_users_email", Columns: []string{"email"}, Unique: true},
				},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectMySQL})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "CREATE UNIQUE INDEX uniq_users_email") {
		t.Error("expected CREATE UNIQUE INDEX")
	}
}

func TestColumnTypes(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"INT", "INTEGER"},
		{"INTEGER", "INTEGER"},
		{"TEXT", "TEXT"},
		{"VARCHAR(255)", "TEXT"},
		{"BOOLEAN", "INTEGER"},
		{"BLOB", "BLOB"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			catalog := &model.Catalog{
				Tables: map[string]*model.Table{
					"test": {
						Name: "test",
						Columns: []*model.Column{
							{Name: "col", Type: tc.input, NotNull: true},
						},
						PrimaryKey: &model.PrimaryKey{Columns: []string{"col"}},
					},
				},
			}

			g := sql.New(sql.Options{Dialect: sql.DialectSQLite})
			files, err := g.Generate(catalog)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			content := string(files[0].Content)
			if !strings.Contains(content, tc.expected) {
				t.Errorf("expected type %q in output, got: %s", tc.expected, content)
			}
		})
	}
}

func TestGeneratePostgresSchema(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: true},
					{Name: "active", Type: "BOOLEAN", NotNull: true},
					{Name: "created_at", Type: "TIMESTAMP", Default: &model.Value{Kind: model.ValueKindKeyword, Text: "CURRENT_TIMESTAMP"}},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
			},
			"posts": {
				Name: "posts",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "user_id", Type: "INTEGER", NotNull: true},
					{Name: "title", Type: "TEXT", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []*model.ForeignKey{
					{
						Columns: []string{"user_id"},
						Ref: model.ForeignKeyRef{
							Table:   "users",
							Columns: []string{"id"},
						},
					},
				},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectPostgres})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "DROP TABLE IF EXISTS users") {
		t.Error("expected DROP TABLE IF EXISTS users")
	}
	if !strings.Contains(content, "CREATE TABLE") {
		t.Error("expected CREATE TABLE")
	}
	if !strings.Contains(content, "FOREIGN KEY") {
		t.Error("expected FOREIGN KEY")
	}
	if !strings.Contains(content, "REFERENCES users") {
		t.Error("expected REFERENCES users")
	}
	if !strings.Contains(content, "BOOLEAN") {
		t.Error("expected BOOLEAN type for PostgreSQL")
	}
	if !strings.Contains(content, "TIMESTAMP") {
		t.Error("expected TIMESTAMP type for PostgreSQL")
	}
}

func TestGeneratePostgresWithIndexes(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
				Indexes: []*model.Index{
					{Name: "idx_users_email", Columns: []string{"email"}},
				},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectPostgres})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "CREATE INDEX idx_users_email") {
		t.Error("expected CREATE INDEX")
	}
}

func TestGeneratePostgresWithUniqueIndex(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
					{Name: "email", Type: "TEXT", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
				Indexes: []*model.Index{
					{Name: "uniq_users_email", Columns: []string{"email"}, Unique: true},
				},
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectPostgres})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "CREATE UNIQUE INDEX uniq_users_email") {
		t.Error("expected CREATE UNIQUE INDEX")
	}
}

func TestGeneratePostgresViews(t *testing.T) {
	catalog := &model.Catalog{
		Tables: map[string]*model.Table{
			"users": {
				Name: "users",
				Columns: []*model.Column{
					{Name: "id", Type: "INTEGER", NotNull: true},
				},
				PrimaryKey: &model.PrimaryKey{Columns: []string{"id"}},
			},
		},
		Views: map[string]*model.View{
			"active_users": {
				Name: "active_users",
				SQL:  "SELECT * FROM users WHERE active = true",
			},
		},
	}

	g := sql.New(sql.Options{Dialect: sql.DialectPostgres})
	files, err := g.Generate(catalog)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files (schema + view), got %d", len(files))
	}

	var viewFile *sql.File
	for i := range files {
		if files[i].Path == "views/active_users.sql" {
			viewFile = &files[i]
			break
		}
	}

	if viewFile == nil {
		t.Fatal("expected view file at views/active_users.sql")
	}

	content := string(viewFile.Content)
	if !strings.Contains(content, "CREATE OR REPLACE VIEW active_users") {
		t.Error("expected CREATE OR REPLACE VIEW")
	}
}

func TestPostgresColumnTypes(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"INTEGER", "INTEGER"},
		{"BIGINT", "BIGINT"},
		{"REAL", "REAL"},
		{"DOUBLE PRECISION", "DOUBLE PRECISION"},
		{"TEXT", "TEXT"},
		{"BOOLEAN", "BOOLEAN"},
		{"TIMESTAMP", "TIMESTAMP"},
		{"DATE", "DATE"},
		{"TIME", "TIME"},
		{"JSON", "JSON"},
		{"JSONB", "JSONB"},
		{"UUID", "UUID"},
		{"BYTEA", "BYTEA"},
		{"SERIAL", "SERIAL"},
		{"BIGSERIAL", "BIGSERIAL"},
		{"VARCHAR(255)", "VARCHAR(255)"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			catalog := &model.Catalog{
				Tables: map[string]*model.Table{
					"test": {
						Name: "test",
						Columns: []*model.Column{
							{Name: "col", Type: tc.input, NotNull: true},
						},
						PrimaryKey: &model.PrimaryKey{Columns: []string{"col"}},
					},
				},
			}

			g := sql.New(sql.Options{Dialect: sql.DialectPostgres})
			files, err := g.Generate(catalog)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			content := string(files[0].Content)
			if !strings.Contains(content, tc.expected) {
				t.Errorf("expected type %q in output, got: %s", tc.expected, content)
			}
		})
	}
}
