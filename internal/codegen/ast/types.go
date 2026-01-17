package ast

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/transform"
)

// TypeInfo describes a resolved Go type.
type TypeInfo struct {
	GoType      string
	UsesSQLNull bool
	Import      string // For custom types
	Package     string // For custom types
}

// TypeResolver handles mapping between SQL types and Go types.
type TypeResolver struct {
	transformer         *transform.Transformer
	emitPointersForNull bool
}

// NewTypeResolver creates a new TypeResolver with optional custom type support.
func NewTypeResolver(transformer *transform.Transformer) *TypeResolver {
	return &TypeResolver{transformer: transformer}
}

// NewTypeResolverWithOptions creates a TypeResolver with all options configured.
func NewTypeResolverWithOptions(transformer *transform.Transformer, emitPointersForNull bool) *TypeResolver {
	return &TypeResolver{
		transformer:         transformer,
		emitPointersForNull: emitPointersForNull,
	}
}

// findCustomMappingBySQLiteType looks up a custom type mapping by SQLite type
func (r *TypeResolver) findCustomMappingBySQLiteType(sqlType string) *config.CustomTypeMapping {
	if r.transformer == nil {
		return nil
	}

	// Iterate through all custom type mappings to find one with matching SQLite type
	for _, customTypeName := range r.transformer.GetCustomTypes() {
		mapping := r.transformer.FindCustomTypeMapping(customTypeName)
		if mapping != nil && mapping.SQLiteType == sqlType {
			return mapping
		}
	}
	return nil
}

// ResolveType determines the Go type for a given SQL type or existing Go type.
func (r *TypeResolver) ResolveType(typeOrSQLType string, nullable bool) TypeInfo {
	// Check if this is already a Go type (contains package qualifiers like "example.IDWrap")
	if strings.Contains(typeOrSQLType, ".") || strings.HasPrefix(typeOrSQLType, "*") {
		// This is already a Go type, handle nullable logic
		goType := typeOrSQLType
		if nullable && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}
		return TypeInfo{GoType: goType, UsesSQLNull: false}
	}

	// This is a SQLite type, check if it has a custom type mapping
	if r.transformer != nil {
		// Look for a custom type mapping for this SQLite type
		if customMapping := r.findCustomMappingBySQLiteType(typeOrSQLType); customMapping != nil {
			goType, isPointer, err := r.transformer.GetGoTypeForCustomType(customMapping.CustomType)
			if err != nil {
				// Fall back to interface{} if mapping is incomplete
				return TypeInfo{GoType: "interface{}", UsesSQLNull: false}
			}

			// Handle nullable custom types BEFORE import check
			if isPointer {
				// Config says this should always be a pointer type
				if !strings.HasPrefix(goType, "*") {
					goType = "*" + goType
				}
			} else if nullable {
				// Add pointer for nullable columns only if not already a pointer
				if !strings.HasPrefix(goType, "*") {
					goType = "*" + goType
				}
			}

			importPath, packageName, err := r.transformer.GetImportsForCustomType(customMapping.CustomType)
			if err != nil {
				// Custom type without import - use just the type name
				return TypeInfo{GoType: goType, UsesSQLNull: false}
			}

			return TypeInfo{
				GoType:      goType,
				UsesSQLNull: false,
				Import:      importPath,
				Package:     packageName,
			}
		}
	}

	// Handle standard SQLite types
	goType := r.sqliteTypeToGo(typeOrSQLType)
	return r.resolveStandardType(goType, nullable)
}

func (r *TypeResolver) sqliteTypeToGo(sqlType string) string {
	// Convert SQLite type to Go type
	upperType := strings.ToUpper(sqlType)

	switch {
	case strings.Contains(upperType, "INT"):
		switch {
		case strings.Contains(upperType, "BIGINT"):
			return "int64"
		case strings.Contains(upperType, "SMALLINT"):
			return "int16"
		case strings.Contains(upperType, "TINYINT"):
			return "int8"
		default:
			return "int32"
		}
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"):
		return "string"
	case strings.Contains(upperType, "BLOB"):
		return "[]byte"
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOAT"), strings.Contains(upperType, "DOUBLE"):
		return "float64"
	case strings.Contains(upperType, "BOOLEAN"), strings.Contains(upperType, "BOOL"):
		return "bool"
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"):
		return "float64"
	default:
		return "interface{}"
	}
}

func (r *TypeResolver) resolveStandardType(goType string, nullable bool) TypeInfo {
	base := strings.TrimSpace(goType)
	if base == "" {
		base = "interface{}"
	}
	if nullable {
		if r.emitPointersForNull {
			// Use pointer types instead of sql.Null*
			if !strings.HasPrefix(base, "*") {
				return TypeInfo{GoType: "*" + base, UsesSQLNull: false}
			}
			return TypeInfo{GoType: base, UsesSQLNull: false}
		}
		switch base {
		case "int64":
			return TypeInfo{GoType: "sql.NullInt64", UsesSQLNull: true}
		case "float64":
			return TypeInfo{GoType: "sql.NullFloat64", UsesSQLNull: true}
		case "string":
			return TypeInfo{GoType: "sql.NullString", UsesSQLNull: true}
		case "bool":
			return TypeInfo{GoType: "sql.NullBool", UsesSQLNull: true}
		default:
			// For custom types or blobs, use pointer
			if !strings.HasPrefix(base, "*") {
				return TypeInfo{GoType: "*" + base, UsesSQLNull: false}
			}
		}
	}
	return TypeInfo{GoType: base, UsesSQLNull: strings.HasPrefix(base, "sql.Null")}
}

// Legacy function for backward compatibility
func resolveType(goType string, nullable bool) TypeInfo {
	resolver := NewTypeResolver(nil)
	return resolver.resolveStandardType(goType, nullable)
}

// CollectImports gathers all unique imports from type information
func CollectImports(typeInfos []TypeInfo) map[string]string {
	imports := make(map[string]string)
	for _, info := range typeInfos {
		if info.Import != "" && info.Package != "" {
			imports[info.Import] = info.Package
		}
	}
	return imports
}
