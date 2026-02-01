package types

import (
	"fmt"
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
)

// GoPostgresMapper converts semantic types to Go types for PostgreSQL (pgx).
type GoPostgresMapper struct {
	customTypes map[string]config.CustomTypeMapping
}

// NewGoPostgresMapper creates a new Go type mapper for PostgreSQL.
func NewGoPostgresMapper(customTypes []config.CustomTypeMapping) *GoPostgresMapper {
	customMap := make(map[string]config.CustomTypeMapping)
	for _, ct := range customTypes {
		customMap[ct.CustomType] = ct
	}
	return &GoPostgresMapper{customTypes: customMap}
}

// Name returns the mapper name.
func (m *GoPostgresMapper) Name() string { return "go-postgres" }

// Map converts a semantic type to a Go type for PostgreSQL.
func (m *GoPostgresMapper) Map(semantic SemanticType) LanguageType {
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

	switch semantic.Category {
	// Integer types - pgx uses standard Go types
	case CategoryInteger:
		return LanguageType{Name: "int32"}
	case CategoryBigInteger, CategorySerial, CategoryBigSerial:
		return LanguageType{Name: "int64"}
	case CategorySmallInteger:
		return LanguageType{Name: "int16"}
	case CategoryTinyInteger:
		return LanguageType{Name: "int8"}

	// Floating point
	case CategoryFloat:
		return LanguageType{Name: "float32"}
	case CategoryDouble:
		return LanguageType{Name: "float64"}
	case CategoryDecimal, CategoryNumeric:
		// Use shopspring/decimal for proper decimal support
		return LanguageType{
			Name:    "decimal.Decimal",
			Import:  "github.com/shopspring/decimal",
			Package: "decimal",
		}

	// String types
	case CategoryText, CategoryChar, CategoryVarchar:
		return LanguageType{Name: "string"}
	case CategoryBlob:
		// BYTEA maps to []byte
		return LanguageType{Name: "[]byte"}

	// Boolean
	case CategoryBoolean:
		return LanguageType{Name: "bool"}

	// Temporal types - pgx has excellent support
	case CategoryTimestamp, CategoryTimestampTZ:
		return LanguageType{
			Name:    "time.Time",
			Import:  "time",
			Package: "time",
		}
	case CategoryDate:
		// pgtype.Date for PostgreSQL DATE
		return LanguageType{
			Name:    "pgtype.Date",
			Import:  "github.com/jackc/pgx/v5/pgtype",
			Package: "pgtype",
		}
	case CategoryTime:
		// pgtype.Time for PostgreSQL TIME
		return LanguageType{
			Name:    "pgtype.Time",
			Import:  "github.com/jackc/pgx/v5/pgtype",
			Package: "pgtype",
		}
	case CategoryTimeTZ:
		// pgtype.TimeTZ for PostgreSQL TIMETZ
		return LanguageType{
			Name:    "pgtype.TimeTZ",
			Import:  "github.com/jackc/pgx/v5/pgtype",
			Package: "pgtype",
		}
	case CategoryInterval:
		// pgtype.Interval for PostgreSQL INTERVAL
		return LanguageType{
			Name:    "pgtype.Interval",
			Import:  "github.com/jackc/pgx/v5/pgtype",
			Package: "pgtype",
		}

	// UUID - pgx has built-in support
	case CategoryUUID:
		return LanguageType{
			Name:    "uuid.UUID",
			Import:  "github.com/google/uuid",
			Package: "uuid",
		}

	// JSON types
	case CategoryJSON, CategoryJSONB:
		// Use interface{} or json.RawMessage, or a custom type
		return LanguageType{
			Name:    "[]byte",
			Import:  "encoding/json",
			Package: "json",
		}

	// XML
	case CategoryXML:
		return LanguageType{Name: "string"}

	// Arrays - pgx handles arrays well
	case CategoryArray:
		if semantic.ElementType != nil {
			elementType := m.Map(*semantic.ElementType)
			return LanguageType{
				Name:    "[]" + elementType.Name,
				Import:  elementType.Import,
				Package: elementType.Package,
			}
		}
		return LanguageType{Name: "[]interface{}"}

	// Enums
	case CategoryEnum:
		return LanguageType{Name: "string"}

	// Custom types
	default:
		return LanguageType{Name: "interface{}"}
	}
}

// GetScanType returns the type used for scanning from pgx.Rows.
// For nullable types, pgx often uses pgtype types.
func (m *GoPostgresMapper) GetScanType(semantic SemanticType) string {
	langType := m.Map(semantic)

	// If nullable and not already a pgtype, use pgtype equivalent
	if semantic.Nullable {
		switch semantic.Category {
		case CategoryText, CategoryVarchar, CategoryChar:
			return "pgtype.Text"
		case CategoryInteger:
			return "pgtype.Int4"
		case CategoryBigInteger:
			return "pgtype.Int8"
		case CategoryBoolean:
			return "pgtype.Bool"
		case CategoryTimestamp, CategoryTimestampTZ:
			return "pgtype.Timestamptz"
		}
	}

	return langType.Name
}

// GetDriverImport returns the import path for the PostgreSQL driver.
func GetDriverImport() string {
	return "github.com/jackc/pgx/v5/pgxpool"
}

// GetDriverPackage returns the package name for the PostgreSQL driver.
func GetDriverPackage() string {
	return "pgxpool"
}

// PostgresDriverType returns the driver type string for sql.Open.
func PostgresDriverType() string {
	return "pgx"
}

// PostgresConnectionString formats a connection string for PostgreSQL.
func PostgresConnectionString(host string, port int, database, user, password string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, database)
}

// GetNullPackage returns the package to use for NULL handling.
// For pgx, we use pgtype package.
func GetNullPackage() (importPath, packageName string) {
	return "github.com/jackc/pgx/v5/pgtype", "pgtype"
}

// GetCommonNullTypes returns commonly used pgtype NULL wrapper types.
func GetCommonNullTypes() map[string]string {
	return map[string]string{
		"Text":        "pgtype.Text",
		"Int4":        "pgtype.Int4",
		"Int8":        "pgtype.Int8",
		"Bool":        "pgtype.Bool",
		"Timestamptz": "pgtype.Timestamptz",
		"Date":        "pgtype.Date",
		"Float8":      "pgtype.Float8",
	}
}

// IsPgxNativeType returns true if the type is natively supported by pgx.
func IsPgxNativeType(semantic SemanticType) bool {
	switch semantic.Category {
	case CategoryInteger, CategoryBigInteger, CategorySmallInteger,
		CategoryFloat, CategoryDouble, CategoryBoolean,
		CategoryText, CategoryTimestamp, CategoryTimestampTZ:
		return true
	default:
		return false
	}
}

// GetPgxScanFunc returns the appropriate pgx scan function for the type.
func GetPgxScanFunc(semantic SemanticType) string {
	switch semantic.Category {
	case CategoryInteger:
		return "ScanInt32"
	case CategoryBigInteger:
		return "ScanInt64"
	case CategoryFloat:
		return "ScanFloat32"
	case CategoryDouble:
		return "ScanFloat64"
	case CategoryBoolean:
		return "ScanBool"
	case CategoryText:
		return "ScanString"
	case CategoryTimestamp, CategoryTimestampTZ:
		return "ScanTime"
	default:
		return "Scan"
	}
}

// FormatPostgresArray formats a Go slice for PostgreSQL array parameter.
func FormatPostgresArray(slice interface{}) string {
	// This would be generated code to convert Go slice to PostgreSQL array
	return fmt.Sprintf("pgtype.Array[%T]{}", slice)
}

// GetPostgresArrayType returns the pgtype array type name.
func GetPostgresArrayType(elementType string) string {
	switch strings.ToLower(elementType) {
	case "int", "int32":
		return "pgtype.Array[int32]"
	case "int64":
		return "pgtype.Array[int64]"
	case "string":
		return "pgtype.Array[string]"
	case "float64":
		return "pgtype.Array[float64]"
	case "bool":
		return "pgtype.Array[bool]"
	default:
		return fmt.Sprintf("pgtype.Array[%s]", elementType)
	}
}
