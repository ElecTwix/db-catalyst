// Package types provides language-agnostic type representations
// and mappings between SQL types and programming language types.
package types

// SemanticTypeCategory represents the semantic meaning of a type,
// independent of any programming language or database system.
type SemanticTypeCategory int

const (
	// CategoryUnknown represents an unrecognized or unspecified type.
	CategoryUnknown SemanticTypeCategory = iota

	// CategoryInteger represents a 32-bit signed integer.
	CategoryInteger
	// CategoryBigInteger represents a 64-bit signed integer.
	CategoryBigInteger
	// CategorySmallInteger represents a 16-bit signed integer.
	CategorySmallInteger
	// CategoryTinyInteger represents an 8-bit signed integer.
	CategoryTinyInteger
	// CategoryDecimal represents an exact decimal with precision/scale.
	CategoryDecimal
	// CategoryFloat represents a 32-bit IEEE 754 float.
	CategoryFloat
	// CategoryDouble represents a 64-bit IEEE 754 float.
	CategoryDouble
	// CategoryNumeric represents a generic numeric (database decides precision).
	CategoryNumeric

	// CategoryText represents a variable-length text.
	CategoryText
	// CategoryChar represents a fixed-length character.
	CategoryChar
	// CategoryVarchar represents a variable-length string with max size.
	CategoryVarchar
	// CategoryBlob represents binary data.
	CategoryBlob
	// CategoryBytea represents PostgreSQL binary type.
	CategoryBytea

	// CategoryTimestamp represents a date and time with timezone.
	CategoryTimestamp
	// CategoryTimestampTZ represents a timestamp with timezone.
	CategoryTimestampTZ
	// CategoryDate represents a date only.
	CategoryDate
	// CategoryTime represents a time only.
	CategoryTime
	// CategoryTimeTZ represents a time with timezone.
	CategoryTimeTZ
	// CategoryInterval represents a time duration.
	CategoryInterval

	// CategoryBoolean represents a boolean type.
	CategoryBoolean

	// CategoryUUID represents a UUID type.
	CategoryUUID
	// CategoryJSON represents a JSON type.
	CategoryJSON
	// CategoryJSONB represents binary JSON (PostgreSQL).
	CategoryJSONB
	// CategoryXML represents an XML type.
	CategoryXML
	// CategoryEnum represents an enumeration type.
	CategoryEnum
	// CategoryArray represents an array of another type.
	CategoryArray
	// CategoryComposite represents a struct-like composite type.
	CategoryComposite
	// CategoryCustom represents a user-defined custom type.
	CategoryCustom
	// CategorySerial represents an auto-incrementing integer.
	CategorySerial
	CategoryBigSerial // Auto-incrementing big integer
)

// String returns a human-readable name for the category
func (c SemanticTypeCategory) String() string {
	switch c {
	case CategoryUnknown:
		return "unknown"
	case CategoryInteger:
		return "integer"
	case CategoryBigInteger:
		return "biginteger"
	case CategorySmallInteger:
		return "smallinteger"
	case CategoryTinyInteger:
		return "tinyinteger"
	case CategoryDecimal:
		return "decimal"
	case CategoryFloat:
		return "float"
	case CategoryDouble:
		return "double"
	case CategoryNumeric:
		return "numeric"
	case CategoryText:
		return "text"
	case CategoryChar:
		return "char"
	case CategoryVarchar:
		return "varchar"
	case CategoryBlob:
		return "blob"
	case CategoryBytea:
		return "bytea"
	case CategoryTimestamp:
		return "timestamp"
	case CategoryTimestampTZ:
		return "timestamptz"
	case CategoryDate:
		return "date"
	case CategoryTime:
		return "time"
	case CategoryTimeTZ:
		return "timetz"
	case CategoryInterval:
		return "interval"
	case CategoryBoolean:
		return "boolean"
	case CategoryUUID:
		return "uuid"
	case CategoryJSON:
		return "json"
	case CategoryJSONB:
		return "jsonb"
	case CategoryXML:
		return "xml"
	case CategoryEnum:
		return "enum"
	case CategoryArray:
		return "array"
	case CategoryComposite:
		return "composite"
	case CategoryCustom:
		return "custom"
	case CategorySerial:
		return "serial"
	case CategoryBigSerial:
		return "bigserial"
	default:
		return "unknown"
	}
}

// SemanticType represents a type with semantic meaning,
// independent of SQL dialect or programming language.
type SemanticType struct {
	Category SemanticTypeCategory
	Nullable bool

	// Precision is the total number of digits for decimal types
	Precision int

	// Scale is the number of digits after the decimal point
	Scale int

	// MaxLength is the maximum length for string types (-1 = unlimited)
	MaxLength int

	// EnumValues contains the allowed values for enum types
	EnumValues []string

	// ElementType is the type of array elements (for CategoryArray)
	ElementType *SemanticType

	// Fields contains the field definitions for composite types
	Fields []FieldDef

	// CustomName is the name of custom type (for CategoryCustom)
	CustomName string

	// CustomPackage is the package/module for custom type
	CustomPackage string
}

// FieldDef defines a field in a composite type
type FieldDef struct {
	Name string
	Type SemanticType
}

// IsNumeric returns true if this is a numeric type
func (s SemanticType) IsNumeric() bool {
	switch s.Category {
	case CategoryInteger, CategoryBigInteger, CategorySmallInteger, CategoryTinyInteger,
		CategoryDecimal, CategoryFloat, CategoryDouble, CategoryNumeric,
		CategorySerial, CategoryBigSerial:
		return true
	default:
		return false
	}
}

// IsText returns true if this is a text/string type
func (s SemanticType) IsText() bool {
	switch s.Category {
	case CategoryText, CategoryChar, CategoryVarchar:
		return true
	default:
		return false
	}
}

// IsTemporal returns true if this is a date/time type
func (s SemanticType) IsTemporal() bool {
	switch s.Category {
	case CategoryTimestamp, CategoryTimestampTZ, CategoryDate, CategoryTime, CategoryTimeTZ, CategoryInterval:
		return true
	default:
		return false
	}
}

// Clone creates a deep copy of the semantic type
func (s SemanticType) Clone() SemanticType {
	clone := s
	if s.ElementType != nil {
		et := s.ElementType.Clone()
		clone.ElementType = &et
	}
	if len(s.EnumValues) > 0 {
		clone.EnumValues = make([]string, len(s.EnumValues))
		copy(clone.EnumValues, s.EnumValues)
	}
	if len(s.Fields) > 0 {
		clone.Fields = make([]FieldDef, len(s.Fields))
		copy(clone.Fields, s.Fields)
	}
	return clone
}
