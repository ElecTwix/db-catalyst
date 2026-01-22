package dialects

import (
	"context"
	"testing"

	"github.com/electwix/db-catalyst/internal/parser/grammars"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		name    string
		dialect grammars.Dialect
		wantErr bool
	}{
		{
			name:    "SQLite parser",
			dialect: grammars.DialectSQLite,
			wantErr: false,
		},
		{
			name:    "PostgreSQL parser",
			dialect: grammars.DialectPostgreSQL,
			wantErr: false,
		},
		{
			name:    "MySQL parser",
			dialect: grammars.DialectMySQL,
			wantErr: false,
		},
		{
			name:    "Unknown dialect",
			dialect: grammars.Dialect("unknown"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.dialect)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && parser == nil {
				t.Error("NewParser() returned nil parser for valid dialect")
			}
		})
	}
}

func TestSQLiteParser_ParseDDL(t *testing.T) {
	ctx := context.Background()
	parser := NewSQLiteParser()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "Invalid SQL",
			sql:     "INVALID SQL STATEMENT",
			wantErr: true,
		},
		{
			name:    "Empty SQL",
			sql:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseDDL(ctx, tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDDL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgreSQLParser_ParseDDL(t *testing.T) {
	ctx := context.Background()
	parser := NewPostgreSQLParser()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "Invalid SQL",
			sql:     "INVALID SQL STATEMENT",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseDDL(ctx, tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDDL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMySQLParser_ParseDDL(t *testing.T) {
	ctx := context.Background()
	parser := NewMySQLParser()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "Invalid SQL",
			sql:     "INVALID SQL STATEMENT",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseDDL(ctx, tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDDL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDialectParser_Validate(t *testing.T) {
	tests := []struct {
		name          string
		dialect       grammars.Dialect
		sql           string
		expectedIssue bool
	}{
		{
			name:          "SQLite - valid",
			dialect:       grammars.DialectSQLite,
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
			expectedIssue: false,
		},
		{
			name:          "SQLite - invalid SERIAL",
			dialect:       grammars.DialectSQLite,
			sql:           "CREATE TABLE users (id SERIAL, name TEXT)",
			expectedIssue: true,
		},
		{
			name:          "PostgreSQL - valid",
			dialect:       grammars.DialectPostgreSQL,
			sql:           "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT)",
			expectedIssue: false,
		},
		{
			name:          "PostgreSQL - invalid AUTOINCREMENT",
			dialect:       grammars.DialectPostgreSQL,
			sql:           "CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)",
			expectedIssue: true,
		},
		{
			name:          "MySQL - valid",
			dialect:       grammars.DialectMySQL,
			sql:           "CREATE TABLE users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))",
			expectedIssue: false,
		},
		{
			name:          "MySQL - invalid SERIAL",
			dialect:       grammars.DialectMySQL,
			sql:           "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(255))",
			expectedIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.dialect)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			issues, err := parser.Validate(tt.sql)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			hasIssue := len(issues) > 0
			if hasIssue != tt.expectedIssue {
				t.Errorf("Validate() issues = %v, expectedIssue %v", issues, tt.expectedIssue)
			}
		})
	}
}

func TestDialectParser_Dialect(t *testing.T) {
	tests := []struct {
		name    string
		dialect grammars.Dialect
		want    grammars.Dialect
	}{
		{
			name:    "SQLite",
			dialect: grammars.DialectSQLite,
			want:    grammars.DialectSQLite,
		},
		{
			name:    "PostgreSQL",
			dialect: grammars.DialectPostgreSQL,
			want:    grammars.DialectPostgreSQL,
		},
		{
			name:    "MySQL",
			dialect: grammars.DialectMySQL,
			want:    grammars.DialectMySQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.dialect)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			if got := parser.Dialect(); got != tt.want {
				t.Errorf("Dialect() = %v, want %v", got, tt.want)
			}
		})
	}
}
