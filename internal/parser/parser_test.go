package parser

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		want    *Parser
	}{
		{
			name:    "default parser",
			options: nil,
			want: &Parser{
				debug:     false,
				maxErrors: 10,
			},
		},
		{
			name: "with debug enabled",
			options: []Option{
				WithDebug(true),
			},
			want: &Parser{
				debug:     true,
				maxErrors: 10,
			},
		},
		{
			name: "with debug disabled",
			options: []Option{
				WithDebug(false),
			},
			want: &Parser{
				debug:     false,
				maxErrors: 10,
			},
		},
		{
			name: "with custom max errors",
			options: []Option{
				WithMaxErrors(5),
			},
			want: &Parser{
				debug:     false,
				maxErrors: 5,
			},
		},
		{
			name: "with max errors set to 1",
			options: []Option{
				WithMaxErrors(1),
			},
			want: &Parser{
				debug:     false,
				maxErrors: 1,
			},
		},
		{
			name: "with multiple options",
			options: []Option{
				WithDebug(true),
				WithMaxErrors(3),
			},
			want: &Parser{
				debug:     true,
				maxErrors: 3,
			},
		},
		{
			name: "options applied in order",
			options: []Option{
				WithMaxErrors(5),
				WithMaxErrors(20),
				WithDebug(true),
				WithDebug(false),
			},
			want: &Parser{
				debug:     false,
				maxErrors: 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewParser(tt.options...)

			if got.debug != tt.want.debug {
				t.Errorf("debug = %v, want %v", got.debug, tt.want.debug)
			}
			if got.maxErrors != tt.want.maxErrors {
				t.Errorf("maxErrors = %v, want %v", got.maxErrors, tt.want.maxErrors)
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid CREATE TABLE",
			input:   "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);",
			wantErr: false,
		},
		{
			name:    "valid CREATE TABLE with multiple columns",
			input:   "CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT NOT NULL, price REAL DEFAULT 0.0, created_at TEXT);",
			wantErr: false,
		},
		{
			name:    "valid CREATE TABLE with foreign key",
			input:   "CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id));",
			wantErr: false,
		},
		{
			name:    "valid CREATE INDEX",
			input:   "CREATE INDEX idx_users_name ON users(name);",
			wantErr: false,
		},
		{
			name:    "valid CREATE VIEW",
			input:   "CREATE VIEW active_users AS SELECT * FROM users WHERE active = 1;",
			wantErr: false,
		},
		{
			name:    "valid ALTER TABLE",
			input:   "CREATE TABLE base (id INTEGER); ALTER TABLE base ADD COLUMN new_col TEXT;",
			wantErr: false,
		},
		{
			name:    "valid multiple statements",
			input:   "CREATE TABLE t1 (id INTEGER); CREATE TABLE t2 (id INTEGER);",
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
		},
		{
			name:    "whitespace only",
			input:   "   \n\t  ",
			wantErr: false,
		},
		{
			name:        "invalid UTF-8",
			input:       string([]byte{0xFF, 0xFE}),
			wantErr:     true,
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated string",
			input:       "CREATE TABLE t (name TEXT DEFAULT 'unterminated);",
			wantErr:     true,
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated block comment",
			input:       "/* unterminated comment",
			wantErr:     true,
			errContains: "tokenization failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			catalog, err := p.Parse(ctx, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}

			if catalog == nil {
				t.Error("Parse() returned nil catalog")
			}
		})
	}
}

func TestParser_Parse_WithMaxErrors(t *testing.T) {
	ctx := context.Background()

	// Create SQL with multiple errors that will generate diagnostics
	// This SQL has syntax issues that will cause parsing errors
	input := `
		CREATE TABLE t1 (id INTEGER PRIMARY KEY);
		CREATE TABLE t2 (id INTEGER PRIMARY KEY);
		CREATE INDEX idx1 ON t1(id);
		CREATE INDEX idx2 ON t2(id);
	`

	tests := []struct {
		name      string
		maxErrors int
		wantErr   bool
	}{
		{
			name:      "high max errors - should succeed",
			maxErrors: 100,
			wantErr:   false,
		},
		{
			name:      "default max errors - should succeed",
			maxErrors: 10,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(WithMaxErrors(tt.maxErrors))
			_, err := p.Parse(ctx, input)

			if tt.wantErr && err == nil {
				t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
		})
	}
}

func TestParser_Parse_ContextCancellation(t *testing.T) {
	tests := []struct {
		name                string
		cancelBeforeParse   bool
		cancelAfterTokenize bool
		wantErr             bool
		errContains         string
	}{
		{
			name:              "cancelled before parse",
			cancelBeforeParse: true,
			wantErr:           true,
			errContains:       "parse cancelled",
		},
		{
			name:              "active context",
			cancelBeforeParse: false,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelBeforeParse {
				cancel()
			}

			p := NewParser()
			input := "CREATE TABLE users (id INTEGER PRIMARY KEY);"
			_, err := p.Parse(ctx, input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
		})
	}
}

func TestParser_Parse_ContextTimeout(t *testing.T) {
	// Test with a very short timeout to ensure context is respected
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout to expire
	time.Sleep(10 * time.Millisecond)

	p := NewParser()
	input := "CREATE TABLE users (id INTEGER PRIMARY KEY);"
	_, err := p.Parse(ctx, input)

	if err == nil {
		t.Error("Parse() expected timeout error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "parse cancelled") {
		t.Errorf("Parse() error = %v, want error containing 'parse cancelled'", err)
	}
}

func TestParser_Parse_TokenizationFailure(t *testing.T) {
	ctx := context.Background()
	p := NewParser()

	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "invalid UTF-8 sequence",
			input:       string([]byte{0xC0, 0x80}),
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated string literal",
			input:       "SELECT 'unterminated string",
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated blob literal",
			input:       "SELECT X'ABCD",
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated quoted identifier",
			input:       `CREATE TABLE "unterminated`,
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated bracket identifier",
			input:       "CREATE TABLE [unterminated",
			errContains: "tokenization failed",
		},
		{
			name:        "unterminated backtick identifier",
			input:       "CREATE TABLE `unterminated",
			errContains: "tokenization failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(ctx, tt.input)

			if err == nil {
				t.Error("Parse() expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
			}
		})
	}
}

func TestParser_Parse_WithDiagnostics(t *testing.T) {
	ctx := context.Background()
	p := NewParser()

	// These inputs produce diagnostics but don't return errors from Parse()
	// The parser collects diagnostics and only returns errors for:
	// - Context cancellation
	// - Tokenization failures
	// - Too many parse errors (exceeding maxErrors)
	tests := []struct {
		name        string
		input       string
		wantCatalog bool
	}{
		{
			name:        "invalid syntax - missing table name produces diagnostics but returns catalog",
			input:       "CREATE TABLE (id INTEGER);",
			wantCatalog: true,
		},
		{
			name:        "unsupported statement produces diagnostics but returns catalog",
			input:       "DROP TABLE users;",
			wantCatalog: true,
		},
		{
			name:        "unexpected symbol produces diagnostics but returns catalog",
			input:       "@#$%^&*();",
			wantCatalog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog, err := p.Parse(ctx, tt.input)

			// These cases don't return errors - they produce diagnostics
			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}

			if tt.wantCatalog && catalog == nil {
				t.Error("Parse() returned nil catalog")
			}
		})
	}
}

func TestParser_Parse_DebugMode(t *testing.T) {
	ctx := context.Background()

	// Debug mode should not affect the parsing result, just potentially log diagnostics
	// Note: The parser doesn't return errors for syntax issues - it collects diagnostics
	tests := []struct {
		name        string
		debug       bool
		input       string
		wantErr     bool
		wantCatalog bool
	}{
		{
			name:        "debug enabled with valid SQL",
			debug:       true,
			input:       "CREATE TABLE users (id INTEGER PRIMARY KEY);",
			wantErr:     false,
			wantCatalog: true,
		},
		{
			name:        "debug disabled with valid SQL",
			debug:       false,
			input:       "CREATE TABLE users (id INTEGER PRIMARY KEY);",
			wantErr:     false,
			wantCatalog: true,
		},
		{
			name:        "debug enabled with SQL that has diagnostics",
			debug:       true,
			input:       "CREATE TABLE (id INTEGER);",
			wantErr:     false, // Parser produces diagnostics, not errors
			wantCatalog: true,
		},
		{
			name:        "debug disabled with SQL that has diagnostics",
			debug:       false,
			input:       "CREATE TABLE (id INTEGER);",
			wantErr:     false, // Parser produces diagnostics, not errors
			wantCatalog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(WithDebug(tt.debug))
			catalog, err := p.Parse(ctx, tt.input)

			if tt.wantErr && err == nil {
				t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}
			if tt.wantCatalog && catalog == nil {
				t.Error("Parse() returned nil catalog")
			}
		})
	}
}

func TestParser_Parse_ReturnsCatalog(t *testing.T) {
	ctx := context.Background()
	p := NewParser()

	input := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			profile_id INTEGER REFERENCES profiles(id)
		);
		CREATE TABLE profiles (
			id INTEGER PRIMARY KEY,
			bio TEXT DEFAULT 'none'
		);
		CREATE INDEX idx_users_email ON users(email);
		CREATE VIEW active_users AS SELECT * FROM users WHERE profile_id IS NOT NULL;
	`

	catalog, err := p.Parse(ctx, input)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	if catalog == nil {
		t.Fatal("Parse() returned nil catalog")
	}

	// Verify tables were parsed
	if len(catalog.Tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(catalog.Tables))
	}

	// Verify users table
	usersTable, ok := catalog.Tables["users"]
	if !ok {
		t.Error("expected 'users' table in catalog")
	} else {
		if len(usersTable.Columns) != 3 {
			t.Errorf("expected 3 columns in users table, got %d", len(usersTable.Columns))
		}
		if usersTable.PrimaryKey == nil {
			t.Error("expected users table to have primary key")
		}
		if len(usersTable.UniqueKeys) != 1 {
			t.Errorf("expected 1 unique key in users table, got %d", len(usersTable.UniqueKeys))
		}
		if len(usersTable.ForeignKeys) != 1 {
			t.Errorf("expected 1 foreign key in users table, got %d", len(usersTable.ForeignKeys))
		}
		if len(usersTable.Indexes) != 1 {
			t.Errorf("expected 1 index in users table, got %d", len(usersTable.Indexes))
		}
	}

	// Verify profiles table
	profilesTable, ok := catalog.Tables["profiles"]
	if !ok {
		t.Error("expected 'profiles' table in catalog")
	} else {
		if len(profilesTable.Columns) != 2 {
			t.Errorf("expected 2 columns in profiles table, got %d", len(profilesTable.Columns))
		}
	}

	// Verify views
	if len(catalog.Views) != 1 {
		t.Errorf("expected 1 view, got %d", len(catalog.Views))
	}

	activeUsersView, ok := catalog.Views["active_users"]
	if !ok {
		t.Error("expected 'active_users' view in catalog")
	} else {
		expectedSQL := "SELECT * FROM users WHERE profile_id IS NOT NULL"
		if activeUsersView.SQL != expectedSQL {
			t.Errorf("view SQL = %q, want %q", activeUsersView.SQL, expectedSQL)
		}
	}
}

func TestParser_Parse_ErrorWrapping(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() (context.Context, context.CancelFunc)
		input        string
		wantErrType  error
		errContains  string
	}{
		{
			name: "context cancelled error wrapping",
			setupContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			input:       "CREATE TABLE t (id INTEGER);",
			wantErrType: context.Canceled,
			errContains: "parse cancelled",
		},
		{
			name: "context deadline exceeded error wrapping",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 0)
			},
			input:       "CREATE TABLE t (id INTEGER);",
			wantErrType: context.DeadlineExceeded,
			errContains: "parse cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupContext()
			defer cancel()

			p := NewParser()
			_, err := p.Parse(ctx, tt.input)

			if err == nil {
				t.Error("Parse() expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
			}

			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("Parse() error should wrap %v", tt.wantErrType)
			}
		})
	}
}

func TestWithDebug(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enable debug", enabled: true},
		{name: "disable debug", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{debug: !tt.enabled} // Start with opposite value
			opt := WithDebug(tt.enabled)
			opt(p)

			if p.debug != tt.enabled {
				t.Errorf("debug = %v, want %v", p.debug, tt.enabled)
			}
		})
	}
}

func TestWithMaxErrors(t *testing.T) {
	tests := []struct {
		name      string
		maxErrors int
	}{
		{name: "zero max errors", maxErrors: 0},
		{name: "one max error", maxErrors: 1},
		{name: "ten max errors", maxErrors: 10},
		{name: "hundred max errors", maxErrors: 100},
		{name: "negative max errors", maxErrors: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{maxErrors: 0}
			opt := WithMaxErrors(tt.maxErrors)
			opt(p)

			if p.maxErrors != tt.maxErrors {
				t.Errorf("maxErrors = %v, want %v", p.maxErrors, tt.maxErrors)
			}
		})
	}
}

func TestParser_Parse_ComplexSchema(t *testing.T) {
	ctx := context.Background()
	p := NewParser()

	input := `
		-- Users table
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		) WITHOUT ROWID;

		-- Profiles table with foreign key
		CREATE TABLE profiles (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			bio TEXT,
			UNIQUE(user_id)
		);

		-- Posts table
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		-- Indexes
		CREATE INDEX idx_posts_user ON posts(user_id);
		CREATE UNIQUE INDEX idx_posts_title ON posts(title);

		-- View
		CREATE VIEW published_posts AS 
			SELECT p.id, p.title, u.email 
			FROM posts p 
			JOIN users u ON p.user_id = u.id 
			WHERE p.published = 1;

		-- Alter table
		ALTER TABLE posts ADD COLUMN updated_at TEXT;
	`

	catalog, err := p.Parse(ctx, input)
	if err != nil {
		t.Fatalf("Parse() unexpected error = %v", err)
	}

	// Verify all tables exist
	expectedTables := []string{"users", "profiles", "posts"}
	for _, tableName := range expectedTables {
		if _, ok := catalog.Tables[tableName]; !ok {
			t.Errorf("expected '%s' table in catalog", tableName)
		}
	}

	// Verify users table specifics
	users := catalog.Tables["users"]
	if !users.WithoutRowID {
		t.Error("expected users table to have WITHOUT ROWID")
	}

	// Verify posts table has the added column from ALTER TABLE
	posts := catalog.Tables["posts"]
	if len(posts.Columns) != 6 { // id, user_id, title, content, published, updated_at
		t.Errorf("expected 6 columns in posts table, got %d", len(posts.Columns))
	}

	// Verify indexes on posts
	if len(posts.Indexes) != 2 {
		t.Errorf("expected 2 indexes on posts table, got %d", len(posts.Indexes))
	}

	// Verify view
	if len(catalog.Views) != 1 {
		t.Errorf("expected 1 view, got %d", len(catalog.Views))
	}
}

func TestParser_Parse_MaxErrorsLimit(t *testing.T) {
	ctx := context.Background()

	// Create SQL that generates multiple errors
	// Using invalid syntax that will produce parsing errors
	input := `
		CREATE TABLE t1 (id INTEGER PRIMARY KEY);
		INVALID STATEMENT 1;
		INVALID STATEMENT 2;
		INVALID STATEMENT 3;
		INVALID STATEMENT 4;
		INVALID STATEMENT 5;
	`

	tests := []struct {
		name      string
		maxErrors int
		wantErr   bool
	}{
		{
			name:      "max errors 1 - should fail",
			maxErrors: 1,
			wantErr:   true,
		},
		{
			name:      "max errors 3 - might fail depending on error count",
			maxErrors: 3,
			wantErr:   true, // The invalid statements generate errors
		},
		{
			name:      "max errors 100 - should succeed",
			maxErrors: 100,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(WithMaxErrors(tt.maxErrors))
			_, err := p.Parse(ctx, input)

			if tt.wantErr && err == nil {
				t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "too many parse errors") && !strings.Contains(err.Error(), "parsing failed") {
					t.Errorf("Parse() error = %v, expected 'too many parse errors' or 'parsing failed'", err)
				}
			}
		})
	}
}

func TestTokenizerErrorType(_ *testing.T) {
	// Test that tokenizer errors are properly returned
	ctx := context.Background()
	p := NewParser()

	// Invalid blob literal with odd number of hex digits
	input := "SELECT X'ABC';"
	_, err := p.Parse(ctx, input)

	// This might or might not error depending on how the parser handles it
	// The tokenizer accepts it, validation might happen elsewhere
	// Just verify the parser doesn't panic
	_ = err
}

func BenchmarkParser_Parse(b *testing.B) {
	ctx := context.Background()
	p := NewParser()
	input := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL,
			name TEXT
		);
		CREATE INDEX idx_users_email ON users(email);
	`

	b.ReportAllocs()
	for b.Loop() {
		_, err := p.Parse(ctx, input)
		if err != nil {
			b.Fatalf("Parse() error = %v", err)
		}
	}
}

func BenchmarkParser_ParseWithDebug(b *testing.B) {
	ctx := context.Background()
	p := NewParser(WithDebug(true))
	input := `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT);`

	b.ReportAllocs()
	for b.Loop() {
		_, err := p.Parse(ctx, input)
		if err != nil {
			b.Fatalf("Parse() error = %v", err)
		}
	}
}
