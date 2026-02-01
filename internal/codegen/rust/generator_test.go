package rust

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

	// Should have 3 files: users.rs, posts.rs, mod.rs
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	// Check that mod.rs was generated
	var foundMod bool
	for _, f := range files {
		if f.Path == "models/mod.rs" {
			foundMod = true
			content := string(f.Content)
			if !strings.Contains(content, "pub mod users;") {
				t.Error("mod.rs should contain 'pub mod users;'")
			}
			if !strings.Contains(content, "pub mod posts;") {
				t.Error("mod.rs should contain 'pub mod posts;'")
			}
		}
	}
	if !foundMod {
		t.Error("mod.rs file not generated")
	}

	// Check users.rs content
	var usersFile *File
	for _, f := range files {
		if f.Path == "models/users.rs" {
			usersFile = &f
			break
		}
	}
	if usersFile == nil {
		t.Fatal("users.rs file not generated")
	}

	content := string(usersFile.Content)
	t.Logf("Generated users.rs:\n%s", content)

	// Check struct name (PascalCase)
	if !strings.Contains(content, "pub struct Users {") {
		t.Error("should contain 'pub struct Users {'")
	}

	// Check fields with proper types
	if !strings.Contains(content, "pub id: i64,") {
		t.Error("should contain 'pub id: i64,' (INTEGER -> i64)")
	}
	if !strings.Contains(content, "pub name: String,") {
		t.Error("should contain 'pub name: String,' (TEXT -> String)")
	}
	if !strings.Contains(content, "pub email: Option<String>,") {
		t.Error("should contain 'pub email: Option<String>,' (nullable TEXT -> Option<String>)")
	}
	if !strings.Contains(content, "pub created_at: i64,") {
		t.Error("should contain 'pub created_at: i64,' (INTEGER -> i64)")
	}
}

func TestNamingConversions(t *testing.T) {
	tests := []struct {
		input      string
		wantSnake  string
		wantPascal string
	}{
		{"users", "users", "Users"},
		{"userPosts", "user_posts", "UserPosts"},
		{"UserPosts", "user_posts", "UserPosts"},
		{"createdAt", "created_at", "CreatedAt"},
		{"XMLParser", "x_m_l_parser", "XMLParser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotSnake := toSnakeCase(tt.input)
			if gotSnake != tt.wantSnake {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, gotSnake, tt.wantSnake)
			}

			gotPascal := toPascalCase(tt.input)
			if gotPascal != tt.wantPascal {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, gotPascal, tt.wantPascal)
			}
		})
	}
}

func TestRustTypeMapping(t *testing.T) {
	gen, _ := NewGenerator()
	mapper := &rustMapper{}

	tests := []struct {
		sqlType  string
		notNull  bool
		expected string
	}{
		{"INTEGER", true, "i64"},
		{"INTEGER", false, "Option<i64>"},
		{"TEXT", true, "String"},
		{"TEXT", false, "Option<String>"},
		{"BOOLEAN", true, "bool"},
		{"BOOLEAN", false, "Option<bool>"},
		{"REAL", true, "f32"},
		{"BLOB", true, "Vec<u8>"},
		{"TIMESTAMP", true, "chrono::DateTime<chrono::Utc>"},
		{"UUID", true, "uuid::Uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			semantic := gen.sqliteToSemantic(tt.sqlType, tt.notNull)
			langType := mapper.Map(semantic)

			var got string
			if semantic.Nullable && !langType.IsNullable {
				got = "Option<" + langType.Name + ">"
			} else {
				got = langType.Name
			}

			if got != tt.expected {
				t.Errorf("type mapping: got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateModFile(t *testing.T) {
	gen, _ := NewGenerator()

	tables := []*model.Table{
		{Name: "users"},
		{Name: "posts"},
		{Name: "comments"},
	}

	content := gen.generateModFile(tables)

	// Check for module declarations
	if !strings.Contains(content, "pub mod users;") {
		t.Error("should declare users module")
	}
	if !strings.Contains(content, "pub mod posts;") {
		t.Error("should declare posts module")
	}
	if !strings.Contains(content, "pub mod comments;") {
		t.Error("should declare comments module")
	}

	// Check for re-exports
	if !strings.Contains(content, "pub use users::Users;") {
		t.Error("should re-export Users")
	}
	if !strings.Contains(content, "pub use posts::Posts;") {
		t.Error("should re-export Posts")
	}
}
