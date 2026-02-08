// Package types provides SQLite-specific type mappings.
//
package types

import (
	"strconv"
	"strings"
)

// Minimum parts needed for DECIMAL(precision, scale) format.
const decimalScaleParts = 2

// SQLiteMapper converts SQLite type names to semantic types.
// SQLite uses dynamic typing, so we infer the semantic meaning
// from the declared type name.
type SQLiteMapper struct{}

// NewSQLiteMapper creates a new SQLite type mapper.
func NewSQLiteMapper() *SQLiteMapper {
	return &SQLiteMapper{}
}

// Map converts a SQLite type declaration to a semantic type.
// SQLite types are case-insensitive and may include length constraints.
func (m *SQLiteMapper) Map(sqlType string, nullable bool) SemanticType {
	upper := strings.ToUpper(strings.TrimSpace(sqlType))

	// Handle type with length constraints like VARCHAR(255)
	baseType, length := parseTypeWithLength(upper)

	switch baseType {
	// Integer types
	case "INTEGER", "INT":
		return SemanticType{Category: CategoryBigInteger, Nullable: nullable}
	case "SMALLINT", "SMALL":
		return SemanticType{Category: CategorySmallInteger, Nullable: nullable}
	case "TINYINT":
		return SemanticType{Category: CategoryTinyInteger, Nullable: nullable}
	case "BIGINT":
		return SemanticType{Category: CategoryBigInteger, Nullable: nullable}
	case "SERIAL":
		return SemanticType{Category: CategorySerial, Nullable: false}
	case "BIGSERIAL":
		return SemanticType{Category: CategoryBigSerial, Nullable: false}

	// Floating point types
	case "REAL":
		return SemanticType{Category: CategoryFloat, Nullable: nullable}
	case "FLOAT", "DOUBLE", "DOUBLE PRECISION":
		return SemanticType{Category: CategoryDouble, Nullable: nullable}

	// Decimal/numeric types
	case "NUMERIC", "DECIMAL", "DEC":
		precision, scale := parseDecimalPrecision(upper)
		return SemanticType{
			Category:  CategoryDecimal,
			Nullable:  nullable,
			Precision: precision,
			Scale:     scale,
		}

	// String types
	case "TEXT":
		return SemanticType{Category: CategoryText, Nullable: nullable}
	case "CHAR", "CHARACTER":
		return SemanticType{
			Category:  CategoryChar,
			Nullable:  nullable,
			MaxLength: length,
		}
	case "VARCHAR", "CHARACTER VARYING":
		return SemanticType{
			Category:  CategoryVarchar,
			Nullable:  nullable,
			MaxLength: length,
		}
	case "CLOB":
		return SemanticType{Category: CategoryText, Nullable: nullable}

	// Binary types
	case "BLOB":
		return SemanticType{Category: CategoryBlob, Nullable: nullable}

	// Boolean types
	case "BOOLEAN", "BOOL":
		return SemanticType{Category: CategoryBoolean, Nullable: nullable}

	// Temporal types (SQLite doesn't have native support, but accepts these)
	case "DATE":
		return SemanticType{Category: CategoryDate, Nullable: nullable}
	case "TIME":
		return SemanticType{Category: CategoryTime, Nullable: nullable}
	case "TIMESTAMP", "DATETIME":
		return SemanticType{Category: CategoryTimestamp, Nullable: nullable}

	// Special types
	case "UUID":
		return SemanticType{Category: CategoryUUID, Nullable: nullable}
	case "JSON":
		return SemanticType{Category: CategoryJSON, Nullable: nullable}

	// Default to custom type for unrecognized types
	default:
		return SemanticType{
			Category:   CategoryCustom,
			Nullable:   nullable,
			CustomName: sqlType,
		}
	}
}

// parseTypeWithLength extracts the base type and length from declarations like "VARCHAR(255)"
func parseTypeWithLength(sqlType string) (baseType string, length int) {
	// Find opening parenthesis
	idx := strings.Index(sqlType, "(")
	if idx == -1 {
		return sqlType, -1
	}

	baseType = strings.TrimSpace(sqlType[:idx])

	// Find closing parenthesis
	endIdx := strings.Index(sqlType[idx:], ")")
	if endIdx == -1 {
		return baseType, -1
	}

	// Extract the number inside parentheses
	lengthStr := sqlType[idx+1 : idx+endIdx]
	length, _ = strconv.Atoi(lengthStr)

	return baseType, length
}

// parseDecimalPrecision extracts precision and scale from DECIMAL(p,s) or NUMERIC(p,s)
func parseDecimalPrecision(sqlType string) (precision, scale int) {
	idx := strings.Index(sqlType, "(")
	if idx == -1 {
		return 0, 0
	}

	endIdx := strings.Index(sqlType[idx:], ")")
	if endIdx == -1 {
		return 0, 0
	}

	parts := strings.Split(sqlType[idx+1:idx+endIdx], ",")
	if len(parts) >= 1 {
		precision, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
	}
	if len(parts) >= decimalScaleParts {
		scale, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}

	return precision, scale
}
