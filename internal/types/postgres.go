//nolint:goconst // PostgreSQL type names are naturally repeated
package types

import (
	"strconv"
	"strings"
)

// PostgresMapper converts PostgreSQL type names to semantic types.
// Supports PostgreSQL-specific types like arrays, JSONB, UUID, etc.
type PostgresMapper struct{}

// NewPostgresMapper creates a new PostgreSQL type mapper.
func NewPostgresMapper() *PostgresMapper {
	return &PostgresMapper{}
}

// Map converts a PostgreSQL type declaration to a semantic type.
// PostgreSQL types are case-insensitive and may include type modifiers.
func (m *PostgresMapper) Map(sqlType string, nullable bool) SemanticType {
	upper := strings.ToUpper(strings.TrimSpace(sqlType))

	// Handle arrays (e.g., TEXT[], INTEGER[])
	if baseType, ok := strings.CutSuffix(upper, "[]"); ok {
		baseSemantic := m.Map(baseType, false)
		return SemanticType{
			Category:    CategoryArray,
			Nullable:    nullable,
			ElementType: &baseSemantic,
		}
	}

	// Handle type with length/precision (e.g., VARCHAR(255), NUMERIC(10,2))
	baseType, length, precision, scale := parsePostgresType(upper)

	switch baseType {
	// Integer types
	case "SMALLINT", "INT2":
		return SemanticType{Category: CategorySmallInteger, Nullable: nullable}
	case "INTEGER", "INT", "INT4":
		return SemanticType{Category: CategoryInteger, Nullable: nullable}
	case "BIGINT", "INT8":
		return SemanticType{Category: CategoryBigInteger, Nullable: nullable}
	case "SERIAL":
		return SemanticType{Category: CategorySerial, Nullable: false}
	case "BIGSERIAL", "SERIAL8":
		return SemanticType{Category: CategoryBigSerial, Nullable: false}
	case "SMALLSERIAL", "SERIAL2":
		return SemanticType{Category: CategorySerial, Nullable: false} // Maps to smallint internally

	// Floating point types
	case "REAL", "FLOAT4":
		return SemanticType{Category: CategoryFloat, Nullable: nullable}
	case "DOUBLE PRECISION", "FLOAT8":
		return SemanticType{Category: CategoryDouble, Nullable: nullable}

	// Decimal/numeric types
	case "NUMERIC", "DECIMAL":
		return SemanticType{
			Category:  CategoryDecimal,
			Nullable:  nullable,
			Precision: precision,
			Scale:     scale,
		}
	case "MONEY":
		// MONEY is fixed precision (19,2) in PostgreSQL
		return SemanticType{
			Category:  CategoryDecimal,
			Nullable:  nullable,
			Precision: 19,
			Scale:     2,
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
	case "BYTEA":
		return SemanticType{Category: CategoryBlob, Nullable: nullable}

	// Boolean type
	case "BOOLEAN", "BOOL":
		return SemanticType{Category: CategoryBoolean, Nullable: nullable}

	// Temporal types
	case "DATE":
		return SemanticType{Category: CategoryDate, Nullable: nullable}
	case "TIME", "TIME WITHOUT TIME ZONE":
		return SemanticType{Category: CategoryTime, Nullable: nullable}
	case "TIMETZ", "TIME WITH TIME ZONE":
		return SemanticType{Category: CategoryTimeTZ, Nullable: nullable}
	case "TIMESTAMP", "TIMESTAMP WITHOUT TIME ZONE":
		return SemanticType{Category: CategoryTimestamp, Nullable: nullable}
	case "TIMESTAMPTZ", "TIMESTAMP WITH TIME ZONE":
		return SemanticType{Category: CategoryTimestampTZ, Nullable: nullable}
	case "INTERVAL":
		return SemanticType{Category: CategoryInterval, Nullable: nullable}

	// UUID type
	case "UUID":
		return SemanticType{Category: CategoryUUID, Nullable: nullable}

	// JSON types
	case "JSON":
		return SemanticType{Category: CategoryJSON, Nullable: nullable}
	case "JSONB":
		return SemanticType{Category: CategoryJSONB, Nullable: nullable}

	// XML type
	case "XML":
		return SemanticType{Category: CategoryXML, Nullable: nullable}

	// Network address types (map to string for simplicity)
	case "INET", "CIDR", "MACADDR", "MACADDR8":
		return SemanticType{Category: CategoryText, Nullable: nullable}

	// Geometric types (map to string for simplicity)
	case "POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE":
		return SemanticType{Category: CategoryText, Nullable: nullable}

	// Range types (map to custom for now)
	case "INT4RANGE", "INT8RANGE", "NUMRANGE", "TSRANGE", "TSTZRANGE", "DATERANGE":
		return SemanticType{
			Category:   CategoryCustom,
			Nullable:   nullable,
			CustomName: baseType,
		}

	// Text search types
	case "TSVECTOR", "TSQUERY":
		return SemanticType{Category: CategoryText, Nullable: nullable}

	// Bit string types
	case "BIT":
		return SemanticType{Category: CategoryBlob, Nullable: nullable}
	case "BIT VARYING", "VARBIT":
		return SemanticType{Category: CategoryBlob, Nullable: nullable}

	// OID types (map to int64)
	case "OID", "REGPROC", "REGPROCEDURE", "REGOPER", "REGOPERATOR",
		"REGCLASS", "REGTYPE", "REGROLE", "REGNAMESPACE", "REGCONFIG", "REGDICTIONARY":
		return SemanticType{Category: CategoryBigInteger, Nullable: nullable}

	// Default to custom type for unrecognized types
	default:
		return SemanticType{
			Category:   CategoryCustom,
			Nullable:   nullable,
			CustomName: sqlType,
		}
	}
}

// parsePostgresType extracts base type and modifiers from PostgreSQL type declarations.
// Handles: VARCHAR(255), NUMERIC(10,2), DECIMAL(5)
func parsePostgresType(sqlType string) (baseType string, length, precision, scale int) {
	// Check for parentheses
	openIdx := strings.Index(sqlType, "(")
	if openIdx == -1 {
		return sqlType, -1, 0, 0
	}

	closeIdx := strings.Index(sqlType[openIdx:], ")")
	if closeIdx == -1 {
		return sqlType, -1, 0, 0
	}
	closeIdx += openIdx

	baseType = strings.TrimSpace(sqlType[:openIdx])
	content := sqlType[openIdx+1 : closeIdx]

	// Check for comma (indicates precision,scale for DECIMAL/NUMERIC)
	if before, after, ok := strings.Cut(content, ","); ok {
		precision, _ = strconv.Atoi(strings.TrimSpace(before))
		scale, _ = strconv.Atoi(strings.TrimSpace(after))
	} else {
		// Single number is length for VARCHAR/CHAR, or precision for NUMERIC
		val, _ := strconv.Atoi(strings.TrimSpace(content))
		if baseType == "NUMERIC" || baseType == "DECIMAL" {
			precision = val
		} else {
			length = val
		}
	}

	return baseType, length, precision, scale
}
