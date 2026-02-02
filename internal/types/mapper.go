package types

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
)

// LanguageType represents a type in a specific programming language
type LanguageType struct {
	// Name is the type name as it appears in code (e.g., "int64", "String")
	Name string

	// Import is the import path if an external package is needed
	// (e.g., "github.com/google/uuid", "time")
	Import string

	// Package is the package/module name for the import
	// (e.g., "uuid", "time", "std::collections")
	Package string

	// PointerPrefix is added before the type for pointers/references
	// (e.g., "*" for Go, "Option<" for Rust)
	PointerPrefix string

	// PointerSuffix is added after the type for pointers/references
	// (e.g., ">" for Rust Option, empty for Go pointers)
	PointerSuffix string

	// IsNullable indicates if this type naturally supports null
	IsNullable bool
}

// FullType returns the type name with pointer/null handling
func (lt LanguageType) FullType(nullable bool) string {
	if nullable && !lt.IsNullable {
		return lt.PointerPrefix + lt.Name + lt.PointerSuffix
	}
	return lt.Name
}

// LanguageMapper is the interface for language-specific type mappings
type LanguageMapper interface {
	// Map converts a semantic type to a language-specific type
	Map(semantic SemanticType) LanguageType

	// Name returns the language identifier (e.g., "go", "rust", "typescript")
	Name() string
}

// GoMapper converts semantic types to Go types
type GoMapper struct {
	emitPointersForNull bool
	customTypes         map[string]config.CustomTypeMapping
}

// NewGoMapper creates a new Go type mapper
func NewGoMapper(customTypes []config.CustomTypeMapping, emitPointersForNull bool) *GoMapper {
	customMap := make(map[string]config.CustomTypeMapping)
	for _, ct := range customTypes {
		customMap[ct.CustomType] = ct
	}

	return &GoMapper{
		emitPointersForNull: emitPointersForNull,
		customTypes:         customMap,
	}
}

// Name returns the language identifier
func (m *GoMapper) Name() string {
	return "go"
}

// Map converts a semantic type to a Go type
func (m *GoMapper) Map(semantic SemanticType) LanguageType {
	// Check for custom type mappings first
	if semantic.Category == CategoryCustom && semantic.CustomName != "" {
		if custom, ok := m.customTypes[semantic.CustomName]; ok {
			return LanguageType{
				Name:    custom.GoType,
				Import:  custom.GoImport,
				Package: custom.GoPackage,
			}
		}
	}

	// Map semantic categories to Go types
	switch semantic.Category {
	case CategoryInteger:
		return LanguageType{Name: "int32"}
	case CategoryBigInteger, CategorySerial, CategoryBigSerial:
		return LanguageType{Name: "int64"}
	case CategorySmallInteger:
		return LanguageType{Name: "int16"}
	case CategoryTinyInteger:
		return LanguageType{Name: "int8"}

	case CategoryFloat:
		return LanguageType{Name: "float32"}
	case CategoryDouble:
		return LanguageType{Name: "float64"}
	case CategoryDecimal, CategoryNumeric:
		return LanguageType{Name: "float64"} // SQLite stores as REAL

	case CategoryText, CategoryChar, CategoryVarchar:
		if semantic.Nullable && !m.emitPointersForNull {
			return LanguageType{
				Name:          "sql.NullString",
				Import:        "database/sql",
				Package:       "sql",
				IsNullable:    true,
				PointerPrefix: "",
			}
		}
		return LanguageType{
			Name:          "string",
			PointerPrefix: "*",
		}

	case CategoryBlob:
		return LanguageType{Name: "[]byte"}

	case CategoryBoolean:
		if semantic.Nullable && !m.emitPointersForNull {
			return LanguageType{
				Name:       "sql.NullBool",
				Import:     "database/sql",
				Package:    "sql",
				IsNullable: true,
			}
		}
		return LanguageType{
			Name:          "bool",
			PointerPrefix: "*",
		}

	case CategoryTimestamp, CategoryTimestampTZ, CategoryDate, CategoryTime:
		if semantic.Nullable && !m.emitPointersForNull {
			return LanguageType{
				Name:       "sql.NullTime",
				Import:     "database/sql",
				Package:    "sql",
				IsNullable: true,
			}
		}
		return LanguageType{
			Name:          "time.Time",
			Import:        "time",
			Package:       "time",
			PointerPrefix: "*",
		}

	case CategoryUUID:
		return LanguageType{
			Name:          "uuid.UUID",
			Import:        "github.com/google/uuid",
			Package:       "uuid",
			PointerPrefix: "*",
		}

	case CategoryJSON:
		return LanguageType{
			Name:          "json.RawMessage",
			Import:        "encoding/json",
			Package:       "json",
			PointerPrefix: "*",
		}

	case CategoryEnum:
		return LanguageType{
			Name:          "string",
			PointerPrefix: "*",
		}

	default:
		return LanguageType{
			Name:          "any",
			PointerPrefix: "",
			IsNullable:    true,
		}
	}
}

// GetDefaultValue returns the default zero value for a Go type
func (m *GoMapper) GetDefaultValue(semantic SemanticType) string {
	switch semantic.Category {
	case CategoryInteger, CategoryBigInteger, CategorySmallInteger, CategoryTinyInteger:
		return "0"
	case CategoryFloat, CategoryDouble, CategoryDecimal, CategoryNumeric:
		return "0.0"
	case CategoryText, CategoryChar, CategoryVarchar:
		return `""`
	case CategoryBoolean:
		return "false"
	case CategoryTimestamp, CategoryTimestampTZ, CategoryDate, CategoryTime:
		return "time.Time{}"
	case CategoryBlob:
		return "nil"
	default:
		return "nil"
	}
}

// RustMapper (for future use)
// Placeholder showing how to add new languages

type RustMapper struct {
	customTypes map[string]string // Map custom type names to Rust types
}

func NewRustMapper() *RustMapper {
	return &RustMapper{
		customTypes: make(map[string]string),
	}
}

func (m *RustMapper) Name() string {
	return "rust"
}

func (m *RustMapper) Map(semantic SemanticType) LanguageType {
	switch semantic.Category {
	case CategoryInteger:
		return LanguageType{Name: "i32"}
	case CategoryBigInteger:
		return LanguageType{Name: "i64"}
	case CategorySmallInteger:
		return LanguageType{Name: "i16"}
	case CategoryTinyInteger:
		return LanguageType{Name: "i8"}
	case CategoryFloat:
		return LanguageType{Name: "f32"}
	case CategoryDouble:
		return LanguageType{Name: "f64"}
	case CategoryText, CategoryChar, CategoryVarchar:
		return LanguageType{Name: "String"}
	case CategoryBoolean:
		return LanguageType{Name: "bool"}
	case CategoryTimestamp, CategoryTimestampTZ:
		return LanguageType{
			Name:    "chrono::DateTime<chrono::Utc>",
			Import:  "chrono",
			Package: "chrono",
		}
	case CategoryDate:
		return LanguageType{
			Name:    "chrono::NaiveDate",
			Import:  "chrono",
			Package: "chrono",
		}
	case CategoryBlob:
		return LanguageType{Name: "Vec<u8>"}
	case CategoryUUID:
		return LanguageType{
			Name:    "uuid::Uuid",
			Import:  "uuid",
			Package: "uuid",
		}
	case CategoryJSON:
		return LanguageType{
			Name:    "serde_json::Value",
			Import:  "serde_json",
			Package: "serde_json",
		}
	default:
		return LanguageType{Name: "serde_json::Value"}
	}
}

// Helper function to check if a type is a Go built-in
func isGoBuiltin(typeName string) bool {
	builtins := []string{
		"bool", "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128",
	}
	for _, b := range builtins {
		if typeName == b {
			return true
		}
	}
	return false
}

// ExtractPackageName extracts the package name from a Go import path
// e.g., "github.com/example/types" -> "types"
func ExtractPackageName(importPath string) string {
	parts := strings.Split(importPath, "/")
	return parts[len(parts)-1]
}
