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
	if info, ok := r.resolveGoType(typeOrSQLType, nullable); ok {
		return info
	}

	// This is a SQLite type, check if it has a custom type mapping
	if info, ok := r.resolveCustomType(typeOrSQLType, nullable); ok {
		return info
	}

	// Handle standard SQLite types
	goType := r.sqliteTypeToGo(typeOrSQLType)
	return r.resolveStandardType(goType, nullable)
}

// resolveGoType handles types that are already Go types.
func (r *TypeResolver) resolveGoType(typeOrSQLType string, nullable bool) (TypeInfo, bool) {
	if !strings.Contains(typeOrSQLType, ".") && !strings.HasPrefix(typeOrSQLType, "*") {
		return TypeInfo{}, false
	}
	goType := typeOrSQLType
	if nullable && !strings.HasPrefix(goType, "*") {
		goType = "*" + goType
	}
	return TypeInfo{GoType: goType, UsesSQLNull: false}, true
}

// resolveCustomType handles custom type mappings.
func (r *TypeResolver) resolveCustomType(sqlType string, nullable bool) (TypeInfo, bool) {
	if r.transformer == nil {
		return TypeInfo{}, false
	}

	customMapping := r.findCustomMappingBySQLiteType(sqlType)
	if customMapping == nil {
		return TypeInfo{}, false
	}

	goType, isPointer, err := r.transformer.GetGoTypeForCustomType(customMapping.CustomType)
	if err != nil {
		return TypeInfo{GoType: "interface{}", UsesSQLNull: false}, true
	}

	goType = r.applyNullability(goType, isPointer, nullable)
	return r.buildCustomTypeInfo(customMapping, goType)
}

// applyNullability adds pointer for nullable types.
func (r *TypeResolver) applyNullability(goType string, isPointer, nullable bool) string {
	if isPointer && !strings.HasPrefix(goType, "*") {
		return "*" + goType
	}
	if nullable && !strings.HasPrefix(goType, "*") {
		return "*" + goType
	}
	return goType
}

// buildCustomTypeInfo creates TypeInfo for custom types with import info.
func (r *TypeResolver) buildCustomTypeInfo(mapping *config.CustomTypeMapping, goType string) (TypeInfo, bool) {
	importPath, packageName, err := r.transformer.GetImportsForCustomType(mapping.CustomType)
	if err != nil {
		return TypeInfo{GoType: goType, UsesSQLNull: false}, true
	}

	return TypeInfo{
		GoType:      goType,
		UsesSQLNull: false,
		Import:      importPath,
		Package:     packageName,
	}, true
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
