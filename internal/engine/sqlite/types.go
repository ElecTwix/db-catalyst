package sqlite

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/engine"
	"github.com/electwix/db-catalyst/internal/types"
)

// customTypeMapping represents a custom type mapping.
type customTypeMapping struct {
	customType string
	sqliteType string
	goType     string
	goImport   string
	goPackage  string
	pointer    bool
}

// typeMapper implements engine.TypeMapper for SQLite.
type typeMapper struct {
	opts        engine.Options
	customTypes map[string]customTypeMapping
}

// newTypeMapper creates a new SQLite type mapper.
func newTypeMapper(opts engine.Options) *typeMapper {
	m := &typeMapper{
		opts:        opts,
		customTypes: make(map[string]customTypeMapping),
	}

	// Load custom types
	for _, ct := range opts.CustomTypes {
		m.customTypes[ct.CustomType] = customTypeMapping{
			customType: ct.CustomType,
			sqliteType: ct.SQLiteType,
			goType:     ct.GoType,
			goImport:   ct.GoImport,
			goPackage:  ct.GoPackage,
			pointer:    ct.Pointer,
		}
	}

	return m
}

// SQLToGo converts a SQLite type to Go type information.
func (m *typeMapper) SQLToGo(sqlType string, nullable bool) engine.TypeInfo {
	// Check if this is a custom type name first
	if custom, ok := m.customTypes[sqlType]; ok {
		return m.resolveCustomType(custom, nullable)
	}

	// Check if this SQL type maps to a custom type
	for _, custom := range m.customTypes {
		if custom.sqliteType == sqlType {
			return m.resolveCustomType(custom, nullable)
		}
	}

	base := m.sqlTypeToGo(sqlType)

	if nullable {
		return m.resolveNullableType(base)
	}

	return engine.TypeInfo{GoType: base}
}

// SQLToSemantic converts a SQLite type to a semantic type category.
func (m *typeMapper) SQLToSemantic(sqlType string, nullable bool) types.SemanticType {
	upperType := strings.ToUpper(sqlType)

	switch {
	case strings.Contains(upperType, "INT"):
		if strings.Contains(upperType, "BIG") {
			return types.SemanticType{Category: types.CategoryBigInteger, Nullable: nullable}
		}
		if strings.Contains(upperType, "SMALL") {
			return types.SemanticType{Category: types.CategorySmallInteger, Nullable: nullable}
		}
		if strings.Contains(upperType, "TINY") {
			return types.SemanticType{Category: types.CategoryTinyInteger, Nullable: nullable}
		}
		return types.SemanticType{Category: types.CategoryInteger, Nullable: nullable}
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"), strings.Contains(upperType, "CLOB"):
		return types.SemanticType{Category: types.CategoryText, Nullable: nullable}
	case strings.Contains(upperType, "BLOB"):
		return types.SemanticType{Category: types.CategoryBlob, Nullable: nullable}
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOAT"), strings.Contains(upperType, "DOUBLE"):
		return types.SemanticType{Category: types.CategoryDouble, Nullable: nullable}
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"):
		return types.SemanticType{Category: types.CategoryNumeric, Nullable: nullable}
	case strings.Contains(upperType, "BOOLEAN"), strings.Contains(upperType, "BOOL"):
		return types.SemanticType{Category: types.CategoryBoolean, Nullable: nullable}
	case strings.Contains(upperType, "DATE"), strings.Contains(upperType, "DATETIME"), strings.Contains(upperType, "TIMESTAMP"):
		return types.SemanticType{Category: types.CategoryTimestamp, Nullable: nullable}
	default:
		return types.SemanticType{Category: types.CategoryUnknown, Nullable: nullable}
	}
}

// GetRequiredImports returns the imports needed for SQLite generated code.
func (m *typeMapper) GetRequiredImports() map[string]string {
	imports := make(map[string]string)
	imports["database/sql"] = "sql"

	// Add custom type imports
	for _, custom := range m.customTypes {
		if custom.goImport != "" {
			pkg := custom.goPackage
			if pkg == "" {
				parts := strings.Split(custom.goImport, "/")
				pkg = parts[len(parts)-1]
			}
			imports[custom.goImport] = pkg
		}
	}

	return imports
}

// SupportsPointersForNull reports whether SQLite supports pointer nullables.
func (m *typeMapper) SupportsPointersForNull() bool {
	return true
}

// resolveCustomType resolves a custom type mapping.
func (m *typeMapper) resolveCustomType(custom customTypeMapping, nullable bool) engine.TypeInfo {
	goType := custom.goType

	if custom.pointer || nullable {
		if !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}
	}

	return engine.TypeInfo{
		GoType:      goType,
		Import:      custom.goImport,
		Package:     custom.goPackage,
		IsPointer:   custom.pointer || nullable,
		UsesSQLNull: false,
	}
}

//nolint:goconst // Type names are naturally repeated and don't need constants
func (m *typeMapper) sqlTypeToGo(sqlType string) string {
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
			return "int64" // SQLite INTEGER is int64
		}
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"), strings.Contains(upperType, "CLOB"):
		return "string"
	case strings.Contains(upperType, "BLOB"):
		return "[]byte"
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOAT"), strings.Contains(upperType, "DOUBLE"):
		return "float64"
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"):
		return "float64"
	case strings.Contains(upperType, "BOOLEAN"), strings.Contains(upperType, "BOOL"):
		return "bool"
	case strings.Contains(upperType, "DATE"), strings.Contains(upperType, "DATETIME"), strings.Contains(upperType, "TIMESTAMP"):
		return "time.Time"
	default:
		return "any"
	}
}

// resolveNullableType handles nullable type resolution for SQLite.
func (m *typeMapper) resolveNullableType(base string) engine.TypeInfo {
	if m.opts.EmitPointersForNull {
		if !strings.HasPrefix(base, "*") {
			return engine.TypeInfo{GoType: "*" + base, IsPointer: true}
		}
		return engine.TypeInfo{GoType: base, IsPointer: true}
	}

	// Use sql.Null* types for nullable values
	switch base {
	case "int64":
		return engine.TypeInfo{GoType: "sql.NullInt64", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "int32":
		return engine.TypeInfo{GoType: "sql.NullInt32", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "float64":
		return engine.TypeInfo{GoType: "sql.NullFloat64", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "string":
		return engine.TypeInfo{GoType: "sql.NullString", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "bool":
		return engine.TypeInfo{GoType: "sql.NullBool", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "time.Time":
		return engine.TypeInfo{GoType: "sql.NullTime", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	default:
		if !strings.HasPrefix(base, "*") {
			return engine.TypeInfo{GoType: "*" + base, IsPointer: true}
		}
		return engine.TypeInfo{GoType: base, IsPointer: true}
	}
}
