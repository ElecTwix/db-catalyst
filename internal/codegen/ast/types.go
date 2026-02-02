package ast

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/transform"
	"github.com/electwix/db-catalyst/internal/types"
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
	database            config.Database
	customTypes         map[string]config.CustomTypeMapping
}

// NewTypeResolver creates a new TypeResolver with optional custom type support.
func NewTypeResolver(transformer *transform.Transformer) *TypeResolver {
	return NewTypeResolverWithDatabase(transformer, config.DatabaseSQLite)
}

// NewTypeResolverWithDatabase creates a TypeResolver for a specific database.
func NewTypeResolverWithDatabase(transformer *transform.Transformer, database config.Database) *TypeResolver {
	resolver := &TypeResolver{
		transformer: transformer,
		database:    database,
		customTypes: make(map[string]config.CustomTypeMapping),
	}

	// Load custom types from transformer
	if transformer != nil {
		for _, customTypeName := range transformer.GetCustomTypes() {
			if mapping := transformer.FindCustomTypeMapping(customTypeName); mapping != nil {
				resolver.customTypes[customTypeName] = *mapping
			}
		}
	}

	return resolver
}

// NewTypeResolverWithOptions creates a TypeResolver with all options configured.
func NewTypeResolverWithOptions(transformer *transform.Transformer, emitPointersForNull bool) *TypeResolver {
	resolver := NewTypeResolverWithDatabase(transformer, config.DatabaseSQLite)
	resolver.emitPointersForNull = emitPointersForNull
	return resolver
}

// NewTypeResolverFull creates a TypeResolver with all options.
func NewTypeResolverFull(transformer *transform.Transformer, database config.Database, emitPointersForNull bool) *TypeResolver {
	resolver := NewTypeResolverWithDatabase(transformer, database)
	resolver.emitPointersForNull = emitPointersForNull
	return resolver
}

// findCustomMappingBySQLType looks up a custom type mapping by SQL type
func (r *TypeResolver) findCustomMappingBySQLType(sqlType string) *config.CustomTypeMapping {
	if r.transformer == nil {
		return nil
	}

	// Iterate through all custom type mappings to find one with matching SQL type
	sqliteType := sqlType // For now, assume SQLite type matches
	for _, customTypeName := range r.transformer.GetCustomTypes() {
		mapping := r.transformer.FindCustomTypeMapping(customTypeName)
		if mapping != nil && mapping.SQLiteType == sqliteType {
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

	// This is a SQL type, check if it has a custom type mapping
	if info, ok := r.resolveCustomType(typeOrSQLType, nullable); ok {
		return info
	}

	// Handle standard SQL types based on database
	goType := r.sqlTypeToGo(typeOrSQLType)
	return r.resolveStandardType(goType, nullable)
}

// resolveGoType handles types that are already Go types (with package qualifiers).
// Note: Primitive Go types like "string", "int" are NOT handled here - they go through
// sqlTypeToGo and resolveNullableType for database-specific null handling.
func (r *TypeResolver) resolveGoType(typeOrSQLType string, nullable bool) (TypeInfo, bool) {
	// Only handle types with package qualifiers (contain ".") or pointers
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

	// First, check if sqlType is a custom type name itself (e.g., "user_id", "money")
	// This happens when the schema has custom types that haven't been transformed yet
	if mapping := r.transformer.FindCustomTypeMapping(sqlType); mapping != nil {
		goType, isPointer, err := r.transformer.GetGoTypeForCustomType(mapping.CustomType)
		if err != nil {
			return TypeInfo{GoType: "interface{}", UsesSQLNull: false}, true
		}
		goType = r.applyNullability(goType, isPointer, nullable)
		return r.buildCustomTypeInfo(mapping, goType)
	}

	// Skip standard SQLite types - they should not map to custom types
	if r.isStandardSQLiteType(sqlType) {
		return TypeInfo{}, false
	}

	// Otherwise, check if it's a standard SQL type that maps to a custom type
	customMapping := r.findCustomMappingBySQLType(sqlType)
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

// sqlTypeToGo converts SQL type to Go type based on database dialect.
// If sqlType is already a Go type (e.g., "string", "int64"), it returns it as-is.
func (r *TypeResolver) sqlTypeToGo(sqlType string) string {
	// Check if it's already a Go primitive type
	isGoPrimitive := sqlType == "string" || sqlType == "int" || sqlType == "int8" ||
		sqlType == "int16" || sqlType == "int32" || sqlType == "int64" ||
		sqlType == "uint" || sqlType == "uint8" || sqlType == "uint16" ||
		sqlType == "uint32" || sqlType == "uint64" || sqlType == "float32" ||
		sqlType == "float64" || sqlType == "bool" || sqlType == "byte" ||
		sqlType == "rune" || sqlType == "[]byte"

	if isGoPrimitive {
		return sqlType
	}

	switch r.database {
	case config.DatabasePostgreSQL:
		return r.postgresTypeToGo(sqlType)
	case config.DatabaseMySQL:
		return r.mysqlTypeToGo(sqlType)
	default:
		return r.sqliteTypeToGo(sqlType)
	}
}

// sqliteTypeToGo converts SQLite type to Go type.
func (r *TypeResolver) sqliteTypeToGo(sqlType string) string {
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

// postgresTypeToGo converts PostgreSQL type to Go type.
func (r *TypeResolver) postgresTypeToGo(sqlType string) string {
	upperType := strings.ToUpper(sqlType)

	switch {
	// Integer types
	case strings.Contains(upperType, "SMALLINT") || strings.Contains(upperType, "INT2"):
		return "int16"
	case strings.Contains(upperType, "BIGINT") || strings.Contains(upperType, "INT8"):
		return "int64"
	case strings.Contains(upperType, "INTEGER") || strings.Contains(upperType, "INT") || strings.Contains(upperType, "INT4"):
		return "int32"
	case strings.Contains(upperType, "SERIAL"):
		return "int64" // SERIAL types return int64

	// Float types
	case strings.Contains(upperType, "REAL") || strings.Contains(upperType, "FLOAT4"):
		return "float32"
	case strings.Contains(upperType, "DOUBLE") || strings.Contains(upperType, "FLOAT8"):
		return "float64"

	// Decimal types
	case strings.Contains(upperType, "NUMERIC") || strings.Contains(upperType, "DECIMAL") || strings.Contains(upperType, "MONEY"):
		return "decimal.Decimal" // Will be resolved with import

	// String types
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"):
		return "string"
	case strings.Contains(upperType, "BYTEA"):
		return "[]byte"

	// Boolean
	case strings.Contains(upperType, "BOOL"):
		return "bool"

	// Temporal types
	case strings.Contains(upperType, "TIMESTAMPTZ") || strings.Contains(upperType, "TIMESTAMP WITH TIME ZONE"):
		return "time.Time"
	case strings.Contains(upperType, "TIMESTAMP"):
		return "time.Time"
	case strings.Contains(upperType, "DATE"):
		return "pgtype.Date"
	case strings.Contains(upperType, "TIME"):
		return "pgtype.Time"
	case strings.Contains(upperType, "INTERVAL"):
		return "pgtype.Interval"

	// Special types
	case strings.Contains(upperType, "UUID"):
		return "uuid.UUID"
	case strings.Contains(upperType, "JSON") || strings.Contains(upperType, "JSONB"):
		return "[]byte"
	case strings.Contains(upperType, "XML"):
		return "string"

	// Arrays
	case strings.HasSuffix(upperType, "[]"):
		// Extract element type and make it an array
		elementType := strings.TrimSuffix(upperType, "[]")
		goElementType := r.postgresTypeToGo(elementType)
		return "[]" + goElementType

	default:
		return "interface{}"
	}
}

// mysqlTypeToGo converts MySQL type to Go type.
func (r *TypeResolver) mysqlTypeToGo(sqlType string) string {
	// For now, use SQLite mappings as base for MySQL
	// This can be expanded later
	return r.sqliteTypeToGo(sqlType)
}

// isStandardSQLiteType checks if a type is a standard SQLite type (not a custom type).
func (r *TypeResolver) isStandardSQLiteType(sqlType string) bool {
	upperType := strings.ToUpper(sqlType)
	standardTypes := []string{
		"INTEGER", "INT", "BIGINT", "SMALLINT", "TINYINT",
		"TEXT", "VARCHAR", "CHAR", "CLOB",
		"BLOB",
		"REAL", "FLOAT", "DOUBLE",
		"NUMERIC", "DECIMAL", "BOOLEAN", "BOOL",
		"DATE", "DATETIME", "TIMESTAMP",
	}
	for _, st := range standardTypes {
		if strings.Contains(upperType, st) {
			return true
		}
	}
	return false
}

// resolveStandardType determines null handling for standard types.
func (r *TypeResolver) resolveStandardType(goType string, nullable bool) TypeInfo {
	base := strings.TrimSpace(goType)
	if base == "" {
		base = "interface{}"
	}

	if nullable {
		return r.resolveNullableType(base)
	}

	return TypeInfo{GoType: base, UsesSQLNull: strings.HasPrefix(base, "sql.Null")}
}

// resolveNullableType handles nullable type resolution based on database.
func (r *TypeResolver) resolveNullableType(base string) TypeInfo {
	if r.emitPointersForNull {
		// Use pointer types instead of sql.Null*/pgtype
		if !strings.HasPrefix(base, "*") {
			return TypeInfo{GoType: "*" + base, UsesSQLNull: false}
		}
		return TypeInfo{GoType: base, UsesSQLNull: false}
	}

	// Use database-specific null types
	switch r.database {
	case config.DatabasePostgreSQL:
		return r.resolvePostgresNullableType(base)
	default:
		return r.resolveSQLiteNullableType(base)
	}
}

// resolveSQLiteNullableType handles SQLite nullable types.
func (r *TypeResolver) resolveSQLiteNullableType(base string) TypeInfo {
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
	return TypeInfo{GoType: base, UsesSQLNull: strings.HasPrefix(base, "sql.Null")}
}

// resolvePostgresNullableType handles PostgreSQL nullable types.
func (r *TypeResolver) resolvePostgresNullableType(base string) TypeInfo {
	switch base {
	case "int32":
		return TypeInfo{GoType: "pgtype.Int4", UsesSQLNull: false, Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "int64":
		return TypeInfo{GoType: "pgtype.Int8", UsesSQLNull: false, Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "float64":
		return TypeInfo{GoType: "pgtype.Float8", UsesSQLNull: false, Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "string":
		return TypeInfo{GoType: "pgtype.Text", UsesSQLNull: false, Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "bool":
		return TypeInfo{GoType: "pgtype.Bool", UsesSQLNull: false, Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "time.Time":
		return TypeInfo{GoType: "pgtype.Timestamptz", UsesSQLNull: false, Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	default:
		// For custom types or other types, use pointer
		if !strings.HasPrefix(base, "*") {
			return TypeInfo{GoType: "*" + base, UsesSQLNull: false}
		}
	}
	return TypeInfo{GoType: base, UsesSQLNull: false}
}

// GetRequiredImports returns the imports needed for the resolved types.
func (r *TypeResolver) GetRequiredImports() map[string]string {
	imports := make(map[string]string)

	switch r.database {
	case config.DatabasePostgreSQL:
		imports["github.com/jackc/pgx/v5/pgtype"] = "pgtype"
		imports["github.com/google/uuid"] = "uuid"
		imports["github.com/shopspring/decimal"] = "decimal"
		imports["time"] = "time"
	default:
		imports["database/sql"] = "sql"
	}

	return imports
}

// resolveType is a package-level function for backward compatibility.
// Deprecated: Use TypeResolver.ResolveType instead.
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

// Map implements the types.Mapper interface for semantic type mapping.
func (r *TypeResolver) Map(sqlType string, nullable bool) types.SemanticType {
	mapper := r.GetSemanticMapper()
	return mapper.Map(sqlType, nullable)
}

// GetSemanticMapper returns the semantic type mapper for the current database.
func (r *TypeResolver) GetSemanticMapper() *types.SQLiteMapper {
	switch r.database {
	case config.DatabasePostgreSQL:
		// For now return Postgres mapper wrapped in SQLiteMapper-compatible interface
		// This is a temporary workaround - the types package needs refactoring
		return types.NewSQLiteMapper()
	default:
		return types.NewSQLiteMapper()
	}
}
