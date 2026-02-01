package config

import (
	"testing"
)

func TestNormalizeCustomTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []CustomTypeMapping
		expected []CustomTypeMapping
	}{
		{
			name: "extract from fully qualified type",
			input: []CustomTypeMapping{
				{
					CustomType: "user_id",
					SQLiteType: "INTEGER",
					GoType:     "github.com/example/types.UserID",
					Pointer:    false,
				},
			},
			expected: []CustomTypeMapping{
				{
					CustomType: "user_id",
					SQLiteType: "INTEGER",
					GoType:     "UserID",
					GoImport:   "github.com/example/types",
					GoPackage:  "types",
					Pointer:    false,
				},
			},
		},
		{
			name: "preserve explicit go_import",
			input: []CustomTypeMapping{
				{
					CustomType: "status",
					SQLiteType: "TEXT",
					GoType:     "Status",
					GoImport:   "github.com/example/types",
					GoPackage:  "types",
					Pointer:    false,
				},
			},
			expected: []CustomTypeMapping{
				{
					CustomType: "status",
					SQLiteType: "TEXT",
					GoType:     "Status",
					GoImport:   "github.com/example/types",
					GoPackage:  "types",
					Pointer:    false,
				},
			},
		},
		{
			name: "extract package when go_import provided but not go_package",
			input: []CustomTypeMapping{
				{
					CustomType: "money",
					SQLiteType: "INTEGER",
					GoType:     "Money",
					GoImport:   "github.com/example/types",
					Pointer:    false,
				},
			},
			expected: []CustomTypeMapping{
				{
					CustomType: "money",
					SQLiteType: "INTEGER",
					GoType:     "Money",
					GoImport:   "github.com/example/types",
					GoPackage:  "types",
					Pointer:    false,
				},
			},
		},
		{
			name: "handle pointer types",
			input: []CustomTypeMapping{
				{
					CustomType: "optional_id",
					SQLiteType: "INTEGER",
					GoType:     "*github.com/example/types.OptionalID",
					Pointer:    true,
				},
			},
			expected: []CustomTypeMapping{
				{
					CustomType: "optional_id",
					SQLiteType: "INTEGER",
					GoType:     "*OptionalID",
					GoImport:   "github.com/example/types",
					GoPackage:  "types",
					Pointer:    true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCustomTypes(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d mappings, got %d", len(tt.expected), len(result))
			}
			for i, got := range result {
				want := tt.expected[i]
				if got.CustomType != want.CustomType {
					t.Errorf("[%d] CustomType: got %q, want %q", i, got.CustomType, want.CustomType)
				}
				if got.GoType != want.GoType {
					t.Errorf("[%d] GoType: got %q, want %q", i, got.GoType, want.GoType)
				}
				if got.GoImport != want.GoImport {
					t.Errorf("[%d] GoImport: got %q, want %q", i, got.GoImport, want.GoImport)
				}
				if got.GoPackage != want.GoPackage {
					t.Errorf("[%d] GoPackage: got %q, want %q", i, got.GoPackage, want.GoPackage)
				}
			}
		})
	}
}

func TestExtractImportAndType(t *testing.T) {
	tests := []struct {
		input      string
		wantImport string
		wantType   string
	}{
		{
			input:      "github.com/example/types.UserID",
			wantImport: "github.com/example/types",
			wantType:   "UserID",
		},
		{
			input:      "*github.com/example/types.UserID",
			wantImport: "github.com/example/types",
			wantType:   "*UserID",
		},
		{
			input:      "UserID",
			wantImport: "",
			wantType:   "UserID",
		},
		{
			input:      "*UserID",
			wantImport: "",
			wantType:   "UserID", // Pointer stripped - use pointer field in config instead
		},
		{
			input:      "github.com/example/nested/pkg.Type",
			wantImport: "github.com/example/nested/pkg",
			wantType:   "Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotImport, gotType := extractImportAndType(tt.input)
			if gotImport != tt.wantImport {
				t.Errorf("import: got %q, want %q", gotImport, tt.wantImport)
			}
			if gotType != tt.wantType {
				t.Errorf("type: got %q, want %q", gotType, tt.wantType)
			}
		})
	}
}
