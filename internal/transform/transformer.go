// Package transform handles custom type transformations.
package transform

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
)

// Regex match group counts for extracting type information.
const (
	columnTypeMatchGroups = 3 // Full match + column name + type name
	typeOnlyMatchGroups   = 2 // Full match + type name only
)

// Precompiled regexes for schema parsing (compiled once at init)
var (
	columnTypeRegex = regexp.MustCompile(`\b(\w+)\s+(\w+)\b`)
	columnLineRegex = regexp.MustCompile(`(?m)^\s*[A-Za-z_][A-Za-z0-9_]*\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	sqlKeywordSet   func(string) bool
)

func init() {
	// Pre-compile SQL keyword set for fast lookup
	keywords := map[string]struct{}{
		"PRIMARY": {}, "KEY": {}, "NOT": {}, "NULL": {}, "UNIQUE": {},
		"DEFAULT": {}, "CHECK": {}, "REFERENCES": {}, "FOREIGN": {},
		"TABLE": {}, "CREATE": {}, "ALTER": {}, "ADD": {}, "COLUMN": {},
		"DROP": {}, "INDEX": {}, "VIEW": {}, "TRIGGER": {}, "ON": {},
		"DELETE": {}, "UPDATE": {}, "INSERT": {}, "INTO": {}, "VALUES": {},
		"SELECT": {}, "FROM": {}, "WHERE": {}, "AND": {}, "OR": {},
		"ORDER": {}, "BY": {}, "GROUP": {}, "HAVING": {}, "LIMIT": {},
		"OFFSET": {}, "JOIN": {}, "LEFT": {}, "RIGHT": {}, "INNER": {},
		"OUTER": {}, "CROSS": {}, "AS": {}, "DISTINCT": {}, "ALL": {},
		"UNION": {}, "INTERSECT": {}, "EXCEPT": {}, "CASE": {},
		"WHEN": {}, "THEN": {}, "ELSE": {}, "END": {}, "EXISTS": {},
		"IN": {}, "LIKE": {}, "BETWEEN": {}, "IS": {}, "ASC": {},
		"DESC": {}, "NULLS": {}, "FIRST": {}, "LAST": {}, "USING": {},
		"CONSTRAINT": {}, "CASCADE": {}, "SET": {}, "RESTRICT": {},
		"NO": {}, "ACTION": {}, "DEFERRABLE": {}, "INITIALLY": {},
		"DEFERRED": {}, "IMMEDIATE": {}, "EXCLUDE": {}, "WITH": {},
		"RECURSIVE": {}, "TEMP": {}, "TEMPORARY": {}, "IF": {},
		"REPLACE": {}, "ABORT": {}, "FAIL": {}, "IGNORE": {},
		"INTEGER": {}, "TEXT": {}, "REAL": {}, "BLOB": {}, "NUMERIC": {},
		"VARCHAR": {}, "CHAR": {}, "BOOLEAN": {}, "DATE": {}, "TIME": {},
		"TIMESTAMP": {}, "INT": {}, "BIGINT": {}, "SMALLINT": {},
		"FLOAT": {}, "DOUBLE": {}, "PRECISION": {},
	}
	sqlKeywordSet = func(word string) bool {
		_, ok := keywords[strings.ToUpper(word)]
		return ok
	}
}

// Transformer handles conversion of custom types to SQLite-compatible types.
type Transformer struct {
	mappings []config.CustomTypeMapping
	// Cache compiled regexes for performance
	typeRegexes []*regexp.Regexp
}

// New creates a new transformer with the given custom type mappings.
func New(mappings []config.CustomTypeMapping) *Transformer {
	t := &Transformer{
		mappings: mappings,
	}
	// Pre-compile regexes for each custom type
	for _, mapping := range mappings {
		// Match custom types only when they appear as types (after a column name)
		// Pattern: match custom type when preceded by a word (column name) and whitespace
		// This avoids replacing the type when it appears as a column name itself
		// Example: "user_id user_id INTEGER" -> only the second "user_id" is the type
		pattern := fmt.Sprintf(`(?m)(\w+\s+)%s\b`, regexp.QuoteMeta(mapping.CustomType))
		t.typeRegexes = append(t.typeRegexes, regexp.MustCompile(pattern))
	}
	return t
}

// TransformSchema converts custom types in schema SQL to SQLite-compatible types.
func (t *Transformer) TransformSchema(input []byte) ([]byte, error) {
	if len(t.mappings) == 0 {
		return input, nil
	}

	result := input
	for i, mapping := range t.mappings {
		// Replace custom types with SQLite types, preserving the captured column name
		// The regex captures (column_name + whitespace) as group 1, so we include it in replacement
		replacement := []byte("${1}" + mapping.SQLiteType)
		result = t.typeRegexes[i].ReplaceAll(result, replacement)
	}

	return result, nil
}

// FindCustomTypeMapping looks up a custom type mapping by the custom type name.
func (t *Transformer) FindCustomTypeMapping(customType string) *config.CustomTypeMapping {
	for i, mapping := range t.mappings {
		if mapping.CustomType == customType {
			return &t.mappings[i]
		}
	}
	return nil
}

// IsCustomType checks if a given type is a custom type.
func (t *Transformer) IsCustomType(typeName string) bool {
	return t.FindCustomTypeMapping(typeName) != nil
}

// GetCustomTypes returns all custom type names.
func (t *Transformer) GetCustomTypes() []string {
	types := make([]string, len(t.mappings))
	for i, mapping := range t.mappings {
		types[i] = mapping.CustomType
	}
	slices.Sort(types)
	return types
}

// GetImportsForCustomType returns the Go import path for a custom type.
func (t *Transformer) GetImportsForCustomType(customType string) (string, string, error) {
	mapping := t.FindCustomTypeMapping(customType)
	if mapping == nil {
		return "", "", fmt.Errorf("no mapping found for custom type %s", customType)
	}
	if mapping.GoImport == "" {
		return "", "", fmt.Errorf("custom type %s has no Go import defined", customType)
	}
	return mapping.GoImport, mapping.GoPackage, nil
}

// GetGoTypeForCustomType returns the Go type name for a custom type.
func (t *Transformer) GetGoTypeForCustomType(customType string) (string, bool, error) {
	mapping := t.FindCustomTypeMapping(customType)
	if mapping == nil {
		return "", false, fmt.Errorf("no mapping found for custom type %s", customType)
	}
	return mapping.GoType, mapping.Pointer, nil
}

// ExtractCustomTypesFromSchema scans schema SQL and returns a list of custom types found.
func (t *Transformer) ExtractCustomTypesFromSchema(input []byte) []string {
	found := make(map[string]bool)

	// Use precompiled regex
	matches := columnTypeRegex.FindAllSubmatch(input, -1)
	for _, match := range matches {
		if len(match) >= columnTypeMatchGroups {
			typeName := string(match[2])
			if t.IsCustomType(typeName) {
				found[typeName] = true
			}
		}
	}

	// Use maps.Keys to extract keys from the map
	result := slices.Collect(maps.Keys(found))
	slices.Sort(result)
	return result
}

// ValidateCustomTypes checks that all custom types used in the schema have mappings.
func (t *Transformer) ValidateCustomTypes(input []byte) []string {
	// Use precompiled regex
	usedTypes := make(map[string]bool)
	matches := columnLineRegex.FindAllSubmatch(input, -1)
	for _, match := range matches {
		if len(match) >= typeOnlyMatchGroups {
			typeName := string(match[1])
			// Skip if it's a standard SQLite type or common SQL keyword
			if !t.IsStandardSQLiteType(typeName) && !sqlKeywordSet(typeName) {
				usedTypes[typeName] = true
			}
		}
	}

	// Use maps.Keys + slices.DeleteFunc for filtering
	missing := slices.Collect(maps.Keys(usedTypes))
	missing = slices.DeleteFunc(missing, func(customType string) bool {
		return t.IsCustomType(customType) // Delete if it's a known custom type
	})
	slices.Sort(missing)
	return missing
}

// IsStandardSQLiteType checks if a type is a standard SQLite type
func (t *Transformer) IsStandardSQLiteType(typeName string) bool {
	standardTypes := map[string]bool{
		"INTEGER": true, "INT": true, "BIGINT": true, "SMALLINT": true, "TINYINT": true,
		"TEXT": true, "VARCHAR": true, "CHAR": true, "CLOB": true,
		"BLOB": true,
		"REAL": true, "FLOAT": true, "DOUBLE": true,
		"NUMERIC": true, "DECIMAL": true, "BOOLEAN": true, "BOOL": true,
		"DATE": true, "DATETIME": true, "TIMESTAMP": true,
	}
	return standardTypes[strings.ToUpper(typeName)]
}
