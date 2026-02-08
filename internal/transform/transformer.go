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
		// Match custom types as whole words in column definitions
		// This matches patterns like "id idwrap" or "trigger_type trigger_type"
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(mapping.CustomType))
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
		// Replace custom types with SQLite types
		result = t.typeRegexes[i].ReplaceAll(result, []byte(mapping.SQLiteType))
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

	// Simple regex to find column definitions
	// This looks for patterns like "column_name custom_type" in CREATE TABLE statements
	columnRegex := regexp.MustCompile(`\b(\w+)\s+(\w+)\b`)

	// Process the entire input, not just CREATE TABLE lines
	matches := columnRegex.FindAllSubmatch(input, -1)
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
	// Find all potential custom types in the schema using a more specific pattern
	// This looks for column definitions: column_name custom_type
	columnRegex := regexp.MustCompile(`(?m)^\s*[A-Za-z_][A-Za-z0-9_]*\s+([A-Za-z_][A-Za-z0-9_]*)\b`)

	usedTypes := make(map[string]bool)
	matches := columnRegex.FindAllSubmatch(input, -1)
	for _, match := range matches {
		if len(match) >= typeOnlyMatchGroups {
			typeName := string(match[1])
			// Skip if it's a standard SQLite type or common SQL keyword
			if !t.IsStandardSQLiteType(typeName) && !t.isSQLKeyword(typeName) {
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

// isSQLKeyword checks if a word is a common SQL keyword
func (t *Transformer) isSQLKeyword(word string) bool {
	keywords := map[string]bool{
		"PRIMARY": true, "KEY": true, "NOT": true, "NULL": true, "UNIQUE": true,
		"DEFAULT": true, "CHECK": true, "REFERENCES": true, "FOREIGN": true,
		"TABLE": true, "CREATE": true, "ALTER": true, "ADD": true, "COLUMN": true,
		"INDEX": true, "VIEW": true, "TRIGGER": true, "DROP": true, "IF": true,
		"EXISTS": true, "TEMP": true, "TEMPORARY": true, "WITHOUT": true, "ROWID": true,
		"AUTOINCREMENT": true, "ASC": true, "DESC": true, "COLLATE": true, "NOCASE": true,
		"RTRIM": true, "BINARY": true, "DEFERRABLE": true, "INITIALLY": true,
		"DEFERRED": true, "IMMEDIATE": true, "EXCLUSIVE": true, "CASCADE": true, "RESTRICT": true,
		"SET": true, "ACTION": true, "MATCH": true, "SIMPLE": true, "FULL": true,
		"PARTIAL": true, "ON": true, "DELETE": true, "UPDATE": true, "INSERT": true, "REPLACE": true,
		"INTO": true, "VALUES": true, "SELECT": true, "FROM": true, "WHERE": true, "GROUP": true,
		"BY": true, "HAVING": true, "ORDER": true, "LIMIT": true, "OFFSET": true, "UNION": true,
		"ALL": true, "DISTINCT": true, "INTERSECT": true, "EXCEPT": true, "INNER": true, "LEFT": true,
		"OUTER": true, "JOIN": true, "NATURAL": true, "CROSS": true, "USING": true, "AS": true,
		"CASE": true, "WHEN": true, "THEN": true, "ELSE": true, "END": true,
		"IN": true, "BETWEEN": true, "LIKE": true, "GLOB": true, "REGEXP": true, "IS": true,
		"AND": true, "OR": true, "CAST": true,
	}
	return keywords[strings.ToUpper(word)]
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
