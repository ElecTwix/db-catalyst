package grammars

import (
	"testing"
)

func TestGetDialectGrammar(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		wantErr bool
	}{
		{
			name:    "SQLite grammar",
			dialect: DialectSQLite,
			wantErr: false,
		},
		{
			name:    "PostgreSQL grammar",
			dialect: DialectPostgreSQL,
			wantErr: false,
		},
		{
			name:    "MySQL grammar",
			dialect: DialectMySQL,
			wantErr: false,
		},
		{
			name:    "Unknown dialect",
			dialect: Dialect("unknown"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDialectGrammar(tt.dialect)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDialectGrammar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Error("GetDialectGrammar() returned empty grammar for valid dialect")
			}
		})
	}
}

func TestValidateSyntax_SQLite(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		expectedIssue bool
		issueContains []string
	}{
		{
			name:          "Valid SQLite CREATE TABLE",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL)",
			expectedIssue: false,
		},
		{
			name:          "SQLite with AUTOINCREMENT",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)",
			expectedIssue: false,
		},
		{
			name:          "SQLite WITHOUT ROWID",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY) WITHOUT ROWID",
			expectedIssue: false,
		},
		{
			name:          "Invalid SERIAL in SQLite",
			sql:           "CREATE TABLE users (id SERIAL, name TEXT)",
			expectedIssue: true,
			issueContains: []string{"SERIAL", "INTEGER PRIMARY KEY"},
		},
		{
			name:          "Invalid AUTO_INCREMENT in SQLite",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY AUTO_INCREMENT, name TEXT)",
			expectedIssue: true,
			issueContains: []string{"AUTO_INCREMENT", "AUTOINCREMENT"},
		},
		{
			name:          "Invalid JSONB in SQLite",
			sql:           "CREATE TABLE data (id INTEGER, metadata JSONB)",
			expectedIssue: true,
			issueContains: []string{"JSONB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := ValidateSyntax(DialectSQLite, tt.sql)
			if err != nil {
				t.Fatalf("ValidateSyntax() error = %v", err)
			}

			hasIssue := len(issues) > 0
			if hasIssue != tt.expectedIssue {
				t.Errorf("ValidateSyntax() issues = %v, expectedIssue %v", issues, tt.expectedIssue)
			}

			if tt.expectedIssue && len(tt.issueContains) > 0 {
				for _, expected := range tt.issueContains {
					found := false
					for _, issue := range issues {
						if contains(issue, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected issue to contain %q, got %v", expected, issues)
					}
				}
			}
		})
	}
}

func TestValidateSyntax_PostgreSQL(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		expectedIssue bool
		issueContains []string
	}{
		{
			name:          "Valid PostgreSQL CREATE TABLE with SERIAL",
			sql:           "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL)",
			expectedIssue: false,
		},
		{
			name:          "Valid PostgreSQL with JSONB",
			sql:           "CREATE TABLE data (id INTEGER, metadata JSONB)",
			expectedIssue: false,
		},
		{
			name:          "Valid PostgreSQL with TIMESTAMP",
			sql:           "CREATE TABLE events (id SERIAL, occurred_at TIMESTAMP WITH TIME ZONE)",
			expectedIssue: false,
		},
		{
			name:          "Invalid AUTOINCREMENT in PostgreSQL",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)",
			expectedIssue: true,
			issueContains: []string{"AUTOINCREMENT"},
		},
		{
			name:          "Invalid WITHOUT ROWID in PostgreSQL",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY) WITHOUT ROWID",
			expectedIssue: true,
			issueContains: []string{"WITHOUT ROWID"},
		},
		{
			name:          "Invalid INTEGER PRIMARY KEY with AUTOINCREMENT",
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)",
			expectedIssue: true,
			issueContains: []string{"AUTOINCREMENT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := ValidateSyntax(DialectPostgreSQL, tt.sql)
			if err != nil {
				t.Fatalf("ValidateSyntax() error = %v", err)
			}

			hasIssue := len(issues) > 0
			if hasIssue != tt.expectedIssue {
				t.Errorf("ValidateSyntax() issues = %v, expectedIssue %v", issues, tt.expectedIssue)
			}

			if tt.expectedIssue && len(tt.issueContains) > 0 {
				for _, expected := range tt.issueContains {
					found := false
					for _, issue := range issues {
						if contains(issue, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected issue to contain %q, got %v", expected, issues)
					}
				}
			}
		})
	}
}

func TestValidateSyntax_MySQL(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		expectedIssue bool
		issueContains []string
	}{
		{
			name:          "Valid MySQL CREATE TABLE",
			sql:           "CREATE TABLE users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL)",
			expectedIssue: false,
		},
		{
			name:          "Valid MySQL with JSON",
			sql:           "CREATE TABLE data (id INT AUTO_INCREMENT PRIMARY KEY, metadata JSON)",
			expectedIssue: false,
		},
		{
			name:          "Valid MySQL with DATETIME",
			sql:           "CREATE TABLE events (id INT AUTO_INCREMENT PRIMARY KEY, occurred_at DATETIME)",
			expectedIssue: false,
		},
		{
			name:          "Invalid SERIAL in MySQL",
			sql:           "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT)",
			expectedIssue: true,
			issueContains: []string{"SERIAL", "AUTO_INCREMENT"},
		},
		{
			name:          "Invalid WITHOUT ROWID in MySQL",
			sql:           "CREATE TABLE users (id INT PRIMARY KEY) WITHOUT ROWID",
			expectedIssue: true,
			issueContains: []string{"WITHOUT ROWID"},
		},
		{
			name:          "Invalid JSONB in MySQL",
			sql:           "CREATE TABLE data (id INT, metadata JSONB)",
			expectedIssue: true,
			issueContains: []string{"JSONB", "JSON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := ValidateSyntax(DialectMySQL, tt.sql)
			if err != nil {
				t.Fatalf("ValidateSyntax() error = %v", err)
			}

			hasIssue := len(issues) > 0
			if hasIssue != tt.expectedIssue {
				t.Errorf("ValidateSyntax() issues = %v, expectedIssue %v", issues, tt.expectedIssue)
			}

			if tt.expectedIssue && len(tt.issueContains) > 0 {
				for _, expected := range tt.issueContains {
					found := false
					for _, issue := range issues {
						if contains(issue, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected issue to contain %q, got %v", expected, issues)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
