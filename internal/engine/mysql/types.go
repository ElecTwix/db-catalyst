package mysql

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/engine"
	"github.com/electwix/db-catalyst/internal/types"
)

// typeMapper implements engine.TypeMapper for MySQL.
type typeMapper struct {
	opts engine.Options
}

// newTypeMapper creates a new MySQL type mapper.
func newTypeMapper(opts engine.Options) *typeMapper {
	return &typeMapper{opts: opts}
}

// SQLToGo converts a MySQL type to Go type information.
func (m *typeMapper) SQLToGo(sqlType string, nullable bool) engine.TypeInfo {
	base := m.sqlTypeToGo(sqlType)

	if nullable {
		return m.resolveNullableType(base)
	}

	return engine.TypeInfo{GoType: base}
}

// SQLToSemantic converts a MySQL type to a semantic type category.
func (m *typeMapper) SQLToSemantic(sqlType string, nullable bool) types.SemanticType {
	upperType := strings.ToUpper(sqlType)

	switch {
	// Integer types
	case strings.Contains(upperType, "TINYINT"):
		return types.SemanticType{Category: types.CategoryTinyInteger, Nullable: nullable}
	case strings.Contains(upperType, "SMALLINT"):
		return types.SemanticType{Category: types.CategorySmallInteger, Nullable: nullable}
	case strings.Contains(upperType, "MEDIUMINT"):
		return types.SemanticType{Category: types.CategoryMediumInteger, Nullable: nullable}
	case strings.Contains(upperType, "INT") || strings.Contains(upperType, "INTEGER"):
		return types.SemanticType{Category: types.CategoryInteger, Nullable: nullable}
	case strings.Contains(upperType, "BIGINT"):
		return types.SemanticType{Category: types.CategoryBigInteger, Nullable: nullable}

	// Serial types
	case strings.Contains(upperType, "SERIAL"):
		return types.SemanticType{Category: types.CategoryBigSerial, Nullable: false}

	// Float types
	case strings.Contains(upperType, "FLOAT"):
		return types.SemanticType{Category: types.CategoryFloat, Nullable: nullable}
	case strings.Contains(upperType, "DOUBLE"):
		return types.SemanticType{Category: types.CategoryDouble, Nullable: nullable}
	case strings.Contains(upperType, "DECIMAL"), strings.Contains(upperType, "DEC"), strings.Contains(upperType, "NUMERIC"):
		return types.SemanticType{Category: types.CategoryNumeric, Nullable: nullable}

	// String types
	case strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"):
		return types.SemanticType{Category: types.CategoryText, Nullable: nullable}
	case strings.Contains(upperType, "TINYTEXT"):
		return types.SemanticType{Category: types.CategoryTinyText, Nullable: nullable}
	case strings.Contains(upperType, "TEXT"):
		return types.SemanticType{Category: types.CategoryText, Nullable: nullable}
	case strings.Contains(upperType, "MEDIUMTEXT"):
		return types.SemanticType{Category: types.CategoryMediumText, Nullable: nullable}
	case strings.Contains(upperType, "LONGTEXT"):
		return types.SemanticType{Category: types.CategoryLongText, Nullable: nullable}

	// Blob types
	case strings.Contains(upperType, "TINYBLOB"):
		return types.SemanticType{Category: types.CategoryTinyBlob, Nullable: nullable}
	case strings.Contains(upperType, "BLOB"):
		return types.SemanticType{Category: types.CategoryBlob, Nullable: nullable}
	case strings.Contains(upperType, "MEDIUMBLOB"):
		return types.SemanticType{Category: types.CategoryMediumBlob, Nullable: nullable}
	case strings.Contains(upperType, "LONGBLOB"):
		return types.SemanticType{Category: types.CategoryLongBlob, Nullable: nullable}
	case strings.Contains(upperType, "BINARY"), strings.Contains(upperType, "VARBINARY"):
		return types.SemanticType{Category: types.CategoryBinary, Nullable: nullable}

	// Boolean
	case strings.Contains(upperType, "BOOL") || strings.Contains(upperType, "BOOLEAN"):
		return types.SemanticType{Category: types.CategoryBoolean, Nullable: nullable}

	// Temporal types
	case strings.Contains(upperType, "DATE"):
		return types.SemanticType{Category: types.CategoryDate, Nullable: nullable}
	case strings.Contains(upperType, "DATETIME"):
		return types.SemanticType{Category: types.CategoryDateTime, Nullable: nullable}
	case strings.Contains(upperType, "TIMESTAMP"):
		return types.SemanticType{Category: types.CategoryTimestamp, Nullable: nullable}
	case strings.Contains(upperType, "TIME"):
		return types.SemanticType{Category: types.CategoryTime, Nullable: nullable}
	case strings.Contains(upperType, "YEAR"):
		return types.SemanticType{Category: types.CategoryYear, Nullable: nullable}

	// JSON
	case strings.Contains(upperType, "JSON"):
		return types.SemanticType{Category: types.CategoryJSON, Nullable: nullable}

	// Enum and Set
	case strings.Contains(upperType, "ENUM"):
		return types.SemanticType{Category: types.CategoryEnum, Nullable: nullable}
	case strings.Contains(upperType, "SET"):
		return types.SemanticType{Category: types.CategorySet, Nullable: nullable}

	default:
		return types.SemanticType{Category: types.CategoryUnknown, Nullable: nullable}
	}
}

// GetRequiredImports returns the imports needed for MySQL generated code.
func (m *typeMapper) GetRequiredImports() map[string]string {
	imports := make(map[string]string)
	imports["database/sql"] = "sql"
	imports["time"] = "time"
	return imports
}

// SupportsPointersForNull reports whether MySQL supports pointer nullables.
func (m *typeMapper) SupportsPointersForNull() bool {
	return true
}

//nolint:goconst // Type names are naturally repeated
func (m *typeMapper) sqlTypeToGo(sqlType string) string {
	upperType := strings.ToUpper(sqlType)

	switch {
	// Integer types with optional display width
	case strings.Contains(upperType, "TINYINT"):
		return "int8"
	case strings.Contains(upperType, "SMALLINT"):
		return "int16"
	case strings.Contains(upperType, "MEDIUMINT"):
		return "int32"
	case strings.Contains(upperType, "INT") || strings.Contains(upperType, "INTEGER"):
		return "int32"
	case strings.Contains(upperType, "BIGINT"):
		return "int64"

	// Serial (auto-increment BIGINT UNSIGNED)
	case strings.Contains(upperType, "SERIAL"):
		return "int64"

	// Float types
	case strings.Contains(upperType, "FLOAT"):
		return "float32"
	case strings.Contains(upperType, "DOUBLE"), strings.Contains(upperType, "REAL"):
		return "float64"
	case strings.Contains(upperType, "DECIMAL"), strings.Contains(upperType, "DEC"), strings.Contains(upperType, "NUMERIC"):
		return "float64" // Using float64 for DECIMAL; consider shopspring/decimal for precision

	// String types
	case strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"),
		strings.Contains(upperType, "TINYTEXT"), strings.Contains(upperType, "TEXT"),
		strings.Contains(upperType, "MEDIUMTEXT"), strings.Contains(upperType, "LONGTEXT"):
		return "string"

	// Blob types
	case strings.Contains(upperType, "BLOB"), strings.Contains(upperType, "BINARY"), strings.Contains(upperType, "VARBINARY"):
		return "[]byte"

	// Boolean (MySQL BOOLEAN is alias for TINYINT(1))
	case strings.Contains(upperType, "BOOL") || strings.Contains(upperType, "BOOLEAN"):
		return "bool"

	// Temporal types
	case strings.Contains(upperType, "DATE"), strings.Contains(upperType, "DATETIME"),
		strings.Contains(upperType, "TIMESTAMP"), strings.Contains(upperType, "TIME"):
		return "time.Time"
	case strings.Contains(upperType, "YEAR"):
		return "int16"

	// JSON
	case strings.Contains(upperType, "JSON"):
		return "[]byte" // Or json.RawMessage with encoding/json

	// Enum and Set
	case strings.Contains(upperType, "ENUM"), strings.Contains(upperType, "SET"):
		return "string"

	default:
		return "any"
	}
}

// resolveNullableType handles nullable type resolution for MySQL.
func (m *typeMapper) resolveNullableType(base string) engine.TypeInfo {
	if m.opts.EmitPointersForNull {
		if !strings.HasPrefix(base, "*") {
			return engine.TypeInfo{GoType: "*" + base, IsPointer: true}
		}
		return engine.TypeInfo{GoType: base, IsPointer: true}
	}

	// Use sql.Null* types for nullable values (same as SQLite)
	switch base {
	case "int8":
		return engine.TypeInfo{GoType: "sql.NullInt64", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "int16":
		return engine.TypeInfo{GoType: "sql.NullInt16", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "int32":
		return engine.TypeInfo{GoType: "sql.NullInt32", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "int64":
		return engine.TypeInfo{GoType: "sql.NullInt64", Import: "database/sql", Package: "sql", UsesSQLNull: true}
	case "float32":
		return engine.TypeInfo{GoType: "sql.NullFloat64", Import: "database/sql", Package: "sql", UsesSQLNull: true}
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
