package postgres

import (
	"strings"

	"github.com/electwix/db-catalyst/internal/engine"
	"github.com/electwix/db-catalyst/internal/types"
)

// typeMapper implements engine.TypeMapper for PostgreSQL.
type typeMapper struct {
	opts engine.Options
}

// SQLToGo converts a PostgreSQL type to Go type information.
func (m *typeMapper) SQLToGo(sqlType string, nullable bool) engine.TypeInfo {
	base := m.sqlTypeToGo(sqlType)

	if nullable {
		return m.resolveNullableType(base, sqlType)
	}

	return engine.TypeInfo{GoType: base}
}

// SQLToSemantic converts a PostgreSQL type to a semantic type category.
func (m *typeMapper) SQLToSemantic(sqlType string, nullable bool) types.SemanticType {
	upperType := strings.ToUpper(sqlType)

	switch {
	case strings.Contains(upperType, "SMALLINT"), strings.Contains(upperType, "INT2"):
		return types.SemanticType{Category: types.CategorySmallInteger, Nullable: nullable}
	case strings.Contains(upperType, "BIGINT"), strings.Contains(upperType, "INT8"):
		return types.SemanticType{Category: types.CategoryBigInteger, Nullable: nullable}
	case strings.Contains(upperType, "INTEGER"), strings.Contains(upperType, "INT"), strings.Contains(upperType, "INT4"):
		return types.SemanticType{Category: types.CategoryInteger, Nullable: nullable}
	case strings.Contains(upperType, "SERIAL"):
		return types.SemanticType{Category: types.CategoryBigSerial, Nullable: false}
	case strings.Contains(upperType, "REAL"), strings.Contains(upperType, "FLOAT4"):
		return types.SemanticType{Category: types.CategoryFloat, Nullable: nullable}
	case strings.Contains(upperType, "DOUBLE"), strings.Contains(upperType, "FLOAT8"):
		return types.SemanticType{Category: types.CategoryDouble, Nullable: nullable}
	case strings.Contains(upperType, "NUMERIC"), strings.Contains(upperType, "DECIMAL"), strings.Contains(upperType, "MONEY"):
		return types.SemanticType{Category: types.CategoryNumeric, Nullable: nullable}
	case strings.Contains(upperType, "TEXT"), strings.Contains(upperType, "CHAR"), strings.Contains(upperType, "VARCHAR"):
		return types.SemanticType{Category: types.CategoryText, Nullable: nullable}
	case strings.Contains(upperType, "BYTEA"):
		return types.SemanticType{Category: types.CategoryBlob, Nullable: nullable}
	case strings.Contains(upperType, "BOOL"):
		return types.SemanticType{Category: types.CategoryBoolean, Nullable: nullable}
	case strings.Contains(upperType, "TIMESTAMPTZ"), strings.Contains(upperType, "TIMESTAMP WITH TIME ZONE"):
		return types.SemanticType{Category: types.CategoryTimestampTZ, Nullable: nullable}
	case strings.Contains(upperType, "TIMESTAMP"):
		return types.SemanticType{Category: types.CategoryTimestamp, Nullable: nullable}
	case strings.Contains(upperType, "DATE"):
		return types.SemanticType{Category: types.CategoryDate, Nullable: nullable}
	case strings.Contains(upperType, "TIME"):
		return types.SemanticType{Category: types.CategoryTime, Nullable: nullable}
	case strings.Contains(upperType, "INTERVAL"):
		return types.SemanticType{Category: types.CategoryInterval, Nullable: nullable}
	case strings.Contains(upperType, "UUID"):
		return types.SemanticType{Category: types.CategoryUUID, Nullable: nullable}
	case strings.Contains(upperType, "JSON"), strings.Contains(upperType, "JSONB"):
		return types.SemanticType{Category: types.CategoryJSON, Nullable: nullable}
	case strings.Contains(upperType, "XML"):
		return types.SemanticType{Category: types.CategoryXML, Nullable: nullable}
	case strings.HasSuffix(upperType, "[]"):
		return types.SemanticType{Category: types.CategoryArray, Nullable: nullable}
	default:
		return types.SemanticType{Category: types.CategoryUnknown, Nullable: nullable}
	}
}

// GetRequiredImports returns the imports needed for PostgreSQL generated code.
func (m *typeMapper) GetRequiredImports() map[string]string {
	imports := make(map[string]string)
	imports["github.com/jackc/pgx/v5/pgtype"] = "pgtype"
	imports["github.com/google/uuid"] = "uuid"
	imports["github.com/shopspring/decimal"] = "decimal"
	imports["time"] = "time"
	return imports
}

// SupportsPointersForNull reports whether PostgreSQL supports pointer nullables.
func (m *typeMapper) SupportsPointersForNull() bool {
	return true
}

//nolint:goconst // Type names are naturally repeated and don't need constants
func (m *typeMapper) sqlTypeToGo(sqlType string) string {
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
		return "int64"

	// Float types
	case strings.Contains(upperType, "REAL") || strings.Contains(upperType, "FLOAT4"):
		return "float32"
	case strings.Contains(upperType, "DOUBLE") || strings.Contains(upperType, "FLOAT8"):
		return "float64"

	// Decimal types
	case strings.Contains(upperType, "NUMERIC") || strings.Contains(upperType, "DECIMAL") || strings.Contains(upperType, "MONEY"):
		return "decimal.Decimal"

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
		elementType := strings.TrimSuffix(upperType, "[]")
		goElementType := m.sqlTypeToGo(elementType)
		return "[]" + goElementType

	default:
		return "any"
	}
}

// resolveNullableType handles nullable type resolution for PostgreSQL.
func (m *typeMapper) resolveNullableType(base, _ string) engine.TypeInfo {
	if m.opts.EmitPointersForNull {
		if !strings.HasPrefix(base, "*") {
			return engine.TypeInfo{GoType: "*" + base, IsPointer: true}
		}
		return engine.TypeInfo{GoType: base, IsPointer: true}
	}

	// Use pgtype types for nullable values
	switch base {
	case "int32":
		return engine.TypeInfo{GoType: "pgtype.Int4", Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "int64":
		return engine.TypeInfo{GoType: "pgtype.Int8", Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "float64":
		return engine.TypeInfo{GoType: "pgtype.Float8", Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "string":
		return engine.TypeInfo{GoType: "pgtype.Text", Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "bool":
		return engine.TypeInfo{GoType: "pgtype.Bool", Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "time.Time":
		return engine.TypeInfo{GoType: "pgtype.Timestamptz", Import: "github.com/jackc/pgx/v5/pgtype", Package: "pgtype"}
	case "uuid.UUID":
		return engine.TypeInfo{GoType: "uuid.UUID", Import: "github.com/google/uuid", Package: "uuid", IsPointer: true}
	case "decimal.Decimal":
		return engine.TypeInfo{GoType: "decimal.Decimal", Import: "github.com/shopspring/decimal", Package: "decimal", IsPointer: true}
	default:
		if !strings.HasPrefix(base, "*") {
			return engine.TypeInfo{GoType: "*" + base, IsPointer: true}
		}
		return engine.TypeInfo{GoType: base, IsPointer: true}
	}
}
