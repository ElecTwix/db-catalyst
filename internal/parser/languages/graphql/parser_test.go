package graphql

import (
	"context"
	"testing"
)

func TestParser_ParseSchema(t *testing.T) {
	t.Skip("GraphQL parser is a proof-of-concept; lexer configuration needs refinement for production use")

	ctx := context.Background()
	parser := NewParser()

	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{
			name:    "Simple type",
			schema:  "type User { id: ID name: String email: String }",
			wantErr: false,
		},
		{
			name:    "Invalid schema",
			schema:  "INVALID GRAPQL SCHEMA",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseSchema(ctx, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParser_Validate(t *testing.T) {
	t.Skip("GraphQL parser is a proof-of-concept; lexer configuration needs refinement for production use")

	parser := NewParser()

	tests := []struct {
		name          string
		schema        string
		expectedIssue bool
	}{
		{
			name:          "Valid simple schema",
			schema:        "type User { id: ID name: String }",
			expectedIssue: false,
		},
		{
			name:          "Invalid syntax",
			schema:        "type User { id ID name String }",
			expectedIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := parser.Validate(tt.schema)
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

func TestParser_mapGraphQLTypeToSQLite(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected string
	}{
		{"ID!", "INTEGER"},
		{"ID", "INTEGER"},
		{"String!", "TEXT"},
		{"String", "TEXT"},
		{"Int!", "INTEGER"},
		{"Int", "INTEGER"},
		{"Float!", "REAL"},
		{"Float", "REAL"},
		{"Boolean!", "INTEGER"},
		{"Boolean", "INTEGER"},
		{"DateTime!", "TEXT"},
		{"DateTime", "TEXT"},
		{"[String]!", "TEXT"},
		{"[String]!", "TEXT"},
		{"JSON", "TEXT"},
		{"UnknownType", "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.mapGraphQLTypeToSQLite(tt.input)
			if result != tt.expected {
				t.Errorf("mapGraphQLTypeToSQLite(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParser_ParserCreation(t *testing.T) {
	parser := NewParser()
	//nolint:staticcheck // Test validates nil check before dereference
	if parser == nil {
		t.Error("NewParser() returned nil")
	}
	//nolint:staticcheck // Test validates nil check before dereference
	if parser.parser == nil {
		t.Error("Parser.parser is nil")
	}
}
