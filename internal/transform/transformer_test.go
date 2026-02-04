//nolint:dupword // SQL test strings intentionally contain repeated type names.
package transform

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/config"
)

func TestTransformer_TransformSchema(t *testing.T) {
	mappings := []config.CustomTypeMapping{
		{CustomType: "idwrap", SQLiteType: "BLOB"},
		{CustomType: "trigger_type", SQLiteType: "TEXT"},
	}

	transformer := New(mappings)

	input := `CREATE TABLE users (
    id idwrap PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE result_api (
    id idwrap PRIMARY KEY,
    trigger_type trigger_type NOT NULL
);`

	expected := `CREATE TABLE users (
    id BLOB PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE result_api (
    id BLOB PRIMARY KEY,
    TEXT TEXT NOT NULL
);`

	result, err := transformer.TransformSchema([]byte(input))
	if err != nil {
		t.Fatalf("TransformSchema failed: %v", err)
	}

	if string(result) != expected {
		t.Errorf("TransformSchema result mismatch:\nGot:\n%s\nExpected:\n%s", string(result), expected)
	}
}

func TestTransformer_FindCustomTypeMapping(t *testing.T) {
	mappings := []config.CustomTypeMapping{
		{CustomType: "idwrap", SQLiteType: "BLOB", GoType: "IDWrap"},
		{CustomType: "trigger_type", SQLiteType: "TEXT", GoType: "TriggerType"},
	}

	transformer := New(mappings)

	// Test existing mapping
	mapping := transformer.FindCustomTypeMapping("idwrap")
	if mapping == nil {
		t.Fatal("FindCustomTypeMapping returned nil for existing type")
	}
	if mapping.GoType != "IDWrap" {
		t.Errorf("Expected GoType 'IDWrap', got '%s'", mapping.GoType)
	}

	// Test non-existing mapping
	mapping = transformer.FindCustomTypeMapping("nonexistent")
	if mapping != nil {
		t.Errorf("FindCustomTypeMapping should return nil for non-existent type, got %+v", mapping)
	}
}

func TestTransformer_IsCustomType(t *testing.T) {
	mappings := []config.CustomTypeMapping{
		{CustomType: "idwrap", SQLiteType: "BLOB"},
	}

	transformer := New(mappings)

	if !transformer.IsCustomType("idwrap") {
		t.Error("IsCustomType should return true for 'idwrap'")
	}

	if transformer.IsCustomType("TEXT") {
		t.Error("IsCustomType should return false for 'TEXT'")
	}
}

func TestTransformer_ExtractCustomTypesFromSchema(t *testing.T) {
	mappings := []config.CustomTypeMapping{
		{CustomType: "idwrap", SQLiteType: "BLOB"},
		{CustomType: "trigger_type", SQLiteType: "TEXT"},
	}

	transformer := New(mappings)

	input := `CREATE TABLE users (
    id idwrap PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE result_api (
    id idwrap PRIMARY KEY,
    trigger_type trigger_type NOT NULL
);`

	found := transformer.ExtractCustomTypesFromSchema([]byte(input))

	expected := []string{"idwrap", "trigger_type"}
	if len(found) != len(expected) {
		t.Fatalf("Expected %d custom types, got %d", len(expected), len(found))
	}

	for i, typ := range expected {
		if i >= len(found) || found[i] != typ {
			t.Errorf("Expected custom type '%s' at index %d, got '%s'", typ, i, found[i])
		}
	}
}

func TestTransformer_ValidateCustomTypes(t *testing.T) {
	mappings := []config.CustomTypeMapping{
		{CustomType: "idwrap", SQLiteType: "BLOB"},
	}

	transformer := New(mappings)

	input := `CREATE TABLE users (
    id idwrap PRIMARY KEY,
    name TEXT NOT NULL,
    status unknown_type
);`

	missing := transformer.ValidateCustomTypes([]byte(input))

	// Filter out standard SQLite types to only check for actual missing custom types
	var filteredMissing []string
	for _, typ := range missing {
		if !transformer.IsStandardSQLiteType(typ) {
			filteredMissing = append(filteredMissing, typ)
		}
	}

	expected := []string{"unknown_type"}
	if len(filteredMissing) != len(expected) {
		t.Logf("All missing types found: %v", missing)
		t.Logf("Filtered missing types: %v", filteredMissing)
		t.Fatalf("Expected %d missing types, got %d", len(expected), len(filteredMissing))
	}

	for i, typ := range expected {
		if i >= len(filteredMissing) || filteredMissing[i] != typ {
			t.Errorf("Expected missing type '%s' at index %d, got '%s'", typ, i, filteredMissing[i])
		}
	}
}

func TestTransformer_GetGoTypeForCustomType(t *testing.T) {
	mappings := []config.CustomTypeMapping{
		{CustomType: "idwrap", SQLiteType: "BLOB", GoType: "IDWrap", Pointer: false},
		{CustomType: "nullable_id", SQLiteType: "BLOB", GoType: "IDWrap", Pointer: true},
	}

	transformer := New(mappings)

	// Test non-pointer type
	goType, pointer, err := transformer.GetGoTypeForCustomType("idwrap")
	if err != nil {
		t.Fatalf("GetGoTypeForCustomType failed: %v", err)
	}
	if goType != "IDWrap" {
		t.Errorf("Expected GoType 'IDWrap', got '%s'", goType)
	}
	if pointer {
		t.Error("Expected pointer=false for idwrap")
	}

	// Test pointer type
	goType, pointer, err = transformer.GetGoTypeForCustomType("nullable_id")
	if err != nil {
		t.Fatalf("GetGoTypeForCustomType failed: %v", err)
	}
	if goType != "IDWrap" {
		t.Errorf("Expected GoType 'IDWrap', got '%s'", goType)
	}
	if !pointer {
		t.Error("Expected pointer=true for nullable_id")
	}

	// Test non-existent type
	_, _, err = transformer.GetGoTypeForCustomType("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent type")
	}
}
