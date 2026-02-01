package typescript

import (
	"strings"
	"testing"

	"github.com/electwix/db-catalyst/internal/schema/model"
)

func TestNewGenerator(t *testing.T) {
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}
	if gen == nil {
		t.Fatal("NewGenerator() returned nil")
	}
	if gen.tmpl == nil {
		t.Fatal("generator template is nil")
	}
}

func TestGenerateModels(t *testing.T) {
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	tables := []*model.Table{
		{
			Name: "users",
			Columns: []*model.Column{
				{Name: "id", Type: "INTEGER", NotNull: true},
				{Name: "name", Type: "TEXT", NotNull: true},
				{Name: "email", Type: "TEXT", NotNull: false},
				{Name: "created_at", Type: "INTEGER", NotNull: true},
			},
		},
		{
			Name: "posts",
			Columns: []*model.Column{
				{Name: "id", Type: "INTEGER", NotNull: true},
				{Name: "user_id", Type: "INTEGER", NotNull: true},
				{Name: "title", Type: "TEXT", NotNull: true},
				{Name: "published", Type: "BOOLEAN", NotNull: true},
			},
		},
	}

	files, err := gen.GenerateModels(tables)
	if err != nil {
		t.Fatalf("GenerateModels() error = %v", err)
	}

	// Should have 3 files: users.ts, posts.ts, index.ts
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	// Check that index.ts was generated
	var foundIndex bool
	for _, f := range files {
		if f.Path == "models/index.ts" {
			foundIndex = true
			content := string(f.Content)
			if !strings.Contains(content, "export * from './users';") {
				t.Error("index.ts should contain 'export * from './users';'")
			}
			if !strings.Contains(content, "export * from './posts';") {
				t.Error("index.ts should contain 'export * from './posts';'")
			}
		}
	}
	if !foundIndex {
		t.Error("index.ts file not generated")
	}

	// Check users.ts content
	var usersFile *File
	for _, f := range files {
		if f.Path == "models/users.ts" {
			usersFile = &f
			break
		}
	}
	if usersFile == nil {
		t.Fatal("users.ts file not generated")
	}

	content := string(usersFile.Content)
	t.Logf("Generated users.ts:\n%s", content)

	// Check interface name (PascalCase)
	if !strings.Contains(content, "export interface Users {") {
		t.Error("should contain 'export interface Users {'")
	}

	// Check fields with proper types
	if !strings.Contains(content, "id: number;") {
		t.Error("should contain 'id: number;' (INTEGER → number)")
	}
	if !strings.Contains(content, "name: string;") {
		t.Error("should contain 'name: string;' (TEXT → string)")
	}
	if !strings.Contains(content, "email: string | null;") {
		t.Error("should contain 'email: string | null;' (nullable TEXT → string | null)")
	}
	if !strings.Contains(content, "createdAt: number;") {
		t.Error("should contain 'createdAt: number;' (INTEGER → number)")
	}
}

func TestNamingConversions(t *testing.T) {
	tests := []struct {
		input      string
		wantCamel  string
		wantPascal string
	}{
		{"users", "users", "Users"},
		{"userPosts", "userPosts", "UserPosts"},
		{"UserPosts", "userPosts", "UserPosts"},
		{"createdAt", "createdAt", "CreatedAt"},
		{"XMLParser", "xMLParser", "XMLParser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotCamel := toCamelCase(tt.input)
			if gotCamel != tt.wantCamel {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, gotCamel, tt.wantCamel)
			}

			gotPascal := toPascalCase(tt.input)
			if gotPascal != tt.wantPascal {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, gotPascal, tt.wantPascal)
			}
		})
	}
}

func TestTypescriptTypeMapping(t *testing.T) {
	gen, _ := NewGenerator()
	mapper := &typescriptMapper{}

	tests := []struct {
		sqlType  string
		notNull  bool
		expected string
	}{
		{"INTEGER", true, "number"},
		{"INTEGER", false, "number | null"},
		{"TEXT", true, "string"},
		{"TEXT", false, "string | null"},
		{"BOOLEAN", true, "boolean"},
		{"BOOLEAN", false, "boolean | null"},
		{"REAL", true, "number"},
		{"BLOB", true, "Buffer"},
		{"TIMESTAMP", true, "Date"},
		{"UUID", true, "string"},
		{"JSON", true, "any"},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			semantic := gen.sqliteToSemantic(tt.sqlType, tt.notNull)
			langType := mapper.Map(semantic)

			var got string
			if semantic.Nullable && !langType.IsNullable {
				got = langType.Name + " | null"
			} else {
				got = langType.Name
			}

			if got != tt.expected {
				t.Errorf("type mapping: got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateIndexFile(t *testing.T) {
	gen, _ := NewGenerator()

	tables := []*model.Table{
		{Name: "users"},
		{Name: "posts"},
		{Name: "comments"},
	}

	content := gen.generateIndexFile(tables)

	// Check for exports
	if !strings.Contains(content, "export * from './users';") {
		t.Error("should export users module")
	}
	if !strings.Contains(content, "export * from './posts';") {
		t.Error("should export posts module")
	}
	if !strings.Contains(content, "export * from './comments';") {
		t.Error("should export comments module")
	}
}
