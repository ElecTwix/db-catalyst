package ast

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/transform"
)

type typeInfo struct {
	GoType      string
	UsesSQLNull bool
	Import      string // For custom types
	Package     string // For custom types
}

type TypeResolver struct {
	transformer *transform.Transformer
}

func NewTypeResolver(transformer *transform.Transformer) *TypeResolver {
	return &TypeResolver{transformer: transformer}
}

func (r *TypeResolver) ResolveType(sqlType string, nullable bool) typeInfo {
	// Check if this is a custom type first
	if r.transformer != nil && r.transformer.IsCustomType(sqlType) {
		goType, isPointer, err := r.transformer.GetGoTypeForCustomType(sqlType)
		if err != nil {
			// Fall back to interface{} if mapping is incomplete
			return typeInfo{GoType: "interface{}", UsesSQLNull: false}
		}

		importPath, packageName, err := r.transformer.GetImportsForCustomType(sqlType)
		if err != nil {
			// Custom type without import - use just the type name
			return typeInfo{GoType: goType, UsesSQLNull: false}
		}

		// Handle nullable custom types
		if nullable && !isPointer {
			goType = "*" + goType
		} else if isPointer {
			// Already a pointer type
		}

		return typeInfo{
			GoType:      goType,
			UsesSQLNull: false,
			Import:      importPath,
			Package:     packageName,
		}
	}

	// Handle standard SQLite types
	goType := r.sqliteTypeToGo(sqlType)
	return r.resolveStandardType(goType, nullable)
}

func (r *TypeResolver) sqliteTypeToGo(sqlType string) string {
	// Convert SQLite type to Go type
	upperType := strings.ToUpper(sqlType)

	switch {
	case strings.Contains(upperType, "INT"):
		if strings.Contains(upperType, "BIGINT") {
			return "int64"
		} else if strings.Contains(upperType, "SMALLINT") {
			return "int16"
		} else if strings.Contains(upperType, "TINYINT") {
			return "int8"
		} else {
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

func (r *TypeResolver) resolveStandardType(goType string, nullable bool) typeInfo {
	base := strings.TrimSpace(goType)
	if base == "" {
		base = "interface{}"
	}
	if nullable {
		switch base {
		case "int64":
			return typeInfo{GoType: "sql.NullInt64", UsesSQLNull: true}
		case "float64":
			return typeInfo{GoType: "sql.NullFloat64", UsesSQLNull: true}
		case "string":
			return typeInfo{GoType: "sql.NullString", UsesSQLNull: true}
		case "bool":
			return typeInfo{GoType: "sql.NullBool", UsesSQLNull: true}
		default:
			// For custom types or blobs, use pointer
			if !strings.HasPrefix(base, "*") {
				return typeInfo{GoType: "*" + base, UsesSQLNull: false}
			}
		}
	}
	return typeInfo{GoType: base, UsesSQLNull: strings.HasPrefix(base, "sql.Null")}
}

// Legacy function for backward compatibility
func resolveType(goType string, nullable bool) typeInfo {
	resolver := NewTypeResolver(nil)
	return resolver.resolveStandardType(goType, nullable)
}

// CollectImports gathers all unique imports from type information
func CollectImports(typeInfos []typeInfo) map[string]string {
	imports := make(map[string]string)
	for _, info := range typeInfos {
		if info.Import != "" && info.Package != "" {
			imports[info.Import] = info.Package
		}
	}
	return imports
}
