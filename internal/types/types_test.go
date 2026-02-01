package types

import (
	"testing"
)

func TestSemanticTypeCategoryString(t *testing.T) {
	tests := []struct {
		category SemanticTypeCategory
		want     string
	}{
		{CategoryInteger, "integer"},
		{CategoryText, "text"},
		{CategoryBoolean, "boolean"},
		{CategoryTimestamp, "timestamp"},
		{CategoryUUID, "uuid"},
		{CategoryCustom, "custom"},
		{CategoryUnknown, "unknown"},
		{SemanticTypeCategory(999), "unknown"}, // Invalid category
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.category.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSemanticTypeIsNumeric(t *testing.T) {
	numericTypes := []SemanticTypeCategory{
		CategoryInteger,
		CategoryBigInteger,
		CategorySmallInteger,
		CategoryDecimal,
		CategoryFloat,
		CategoryDouble,
	}

	nonNumericTypes := []SemanticTypeCategory{
		CategoryText,
		CategoryBoolean,
		CategoryTimestamp,
		CategoryBlob,
		CategoryUUID,
	}

	for _, cat := range numericTypes {
		t.Run(cat.String(), func(t *testing.T) {
			st := SemanticType{Category: cat}
			if !st.IsNumeric() {
				t.Errorf("IsNumeric() = false for %s", cat)
			}
		})
	}

	for _, cat := range nonNumericTypes {
		t.Run(cat.String(), func(t *testing.T) {
			st := SemanticType{Category: cat}
			if st.IsNumeric() {
				t.Errorf("IsNumeric() = true for %s", cat)
			}
		})
	}
}

func TestSQLiteMapper(t *testing.T) {
	mapper := NewSQLiteMapper()

	tests := []struct {
		sqlType  string
		nullable bool
		want     SemanticType
	}{
		{
			sqlType:  "INTEGER",
			nullable: false,
			want:     SemanticType{Category: CategoryBigInteger, Nullable: false},
		},
		{
			sqlType:  "TEXT",
			nullable: true,
			want:     SemanticType{Category: CategoryText, Nullable: true},
		},
		{
			sqlType:  "VARCHAR(255)",
			nullable: false,
			want:     SemanticType{Category: CategoryVarchar, Nullable: false, MaxLength: 255},
		},
		{
			sqlType:  "DECIMAL(10,2)",
			nullable: false,
			want:     SemanticType{Category: CategoryDecimal, Nullable: false, Precision: 10, Scale: 2},
		},
		{
			sqlType:  "BOOLEAN",
			nullable: true,
			want:     SemanticType{Category: CategoryBoolean, Nullable: true},
		},
		{
			sqlType:  "TIMESTAMP",
			nullable: false,
			want:     SemanticType{Category: CategoryTimestamp, Nullable: false},
		},
		{
			sqlType:  "UUID",
			nullable: false,
			want:     SemanticType{Category: CategoryUUID, Nullable: false},
		},
		{
			sqlType:  "MY_CUSTOM_TYPE",
			nullable: false,
			want:     SemanticType{Category: CategoryCustom, Nullable: false, CustomName: "MY_CUSTOM_TYPE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			got := mapper.Map(tt.sqlType, tt.nullable)
			if got.Category != tt.want.Category {
				t.Errorf("Category = %v, want %v", got.Category, tt.want.Category)
			}
			if got.Nullable != tt.want.Nullable {
				t.Errorf("Nullable = %v, want %v", got.Nullable, tt.want.Nullable)
			}
			if got.MaxLength != tt.want.MaxLength {
				t.Errorf("MaxLength = %v, want %v", got.MaxLength, tt.want.MaxLength)
			}
			if got.Precision != tt.want.Precision {
				t.Errorf("Precision = %v, want %v", got.Precision, tt.want.Precision)
			}
			if got.Scale != tt.want.Scale {
				t.Errorf("Scale = %v, want %v", got.Scale, tt.want.Scale)
			}
		})
	}
}

func TestGoMapper(t *testing.T) {
	mapper := NewGoMapper(nil, false)

	tests := []struct {
		name     string
		semantic SemanticType
		wantName string
	}{
		{
			name:     "integer",
			semantic: SemanticType{Category: CategoryInteger},
			wantName: "int32",
		},
		{
			name:     "big integer",
			semantic: SemanticType{Category: CategoryBigInteger},
			wantName: "int64",
		},
		{
			name:     "text",
			semantic: SemanticType{Category: CategoryText},
			wantName: "string",
		},
		{
			name:     "text nullable",
			semantic: SemanticType{Category: CategoryText, Nullable: true},
			wantName: "sql.NullString",
		},
		{
			name:     "float",
			semantic: SemanticType{Category: CategoryFloat},
			wantName: "float32",
		},
		{
			name:     "boolean",
			semantic: SemanticType{Category: CategoryBoolean},
			wantName: "bool",
		},
		{
			name:     "timestamp",
			semantic: SemanticType{Category: CategoryTimestamp},
			wantName: "time.Time",
		},
		{
			name:     "blob",
			semantic: SemanticType{Category: CategoryBlob},
			wantName: "[]byte",
		},
		{
			name:     "uuid",
			semantic: SemanticType{Category: CategoryUUID},
			wantName: "uuid.UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.Map(tt.semantic)
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
		})
	}
}

func TestGoMapperWithPointers(t *testing.T) {
	mapper := NewGoMapper(nil, true) // emitPointersForNull = true

	// When emitPointersForNull is true, nullable types should use pointers
	textNullable := SemanticType{Category: CategoryText, Nullable: true}
	got := mapper.Map(textNullable)

	if got.Name != "string" {
		t.Errorf("Name = %q, want %q", got.Name, "string")
	}

	if got.PointerPrefix != "*" {
		t.Errorf("PointerPrefix = %q, want %q", got.PointerPrefix, "*")
	}
}

func TestLanguageTypeFullType(t *testing.T) {
	tests := []struct {
		name     string
		lt       LanguageType
		nullable bool
		want     string
	}{
		{
			name:     "non-nullable",
			lt:       LanguageType{Name: "int64"},
			nullable: false,
			want:     "int64",
		},
		{
			name:     "nullable with pointer",
			lt:       LanguageType{Name: "string", PointerPrefix: "*"},
			nullable: true,
			want:     "*string",
		},
		{
			name:     "nullable already nullable",
			lt:       LanguageType{Name: "sql.NullString", IsNullable: true},
			nullable: true,
			want:     "sql.NullString",
		},
		{
			name:     "rust option",
			lt:       LanguageType{Name: "String", PointerPrefix: "Option<", PointerSuffix: ">"},
			nullable: true,
			want:     "Option<String>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lt.FullType(tt.nullable)
			if got != tt.want {
				t.Errorf("FullType(%v) = %q, want %q", tt.nullable, got, tt.want)
			}
		})
	}
}

func TestSemanticTypeClone(t *testing.T) {
	original := SemanticType{
		Category:    CategoryArray,
		Nullable:    false,
		ElementType: &SemanticType{Category: CategoryText},
		EnumValues:  []string{"a", "b", "c"},
	}

	clone := original.Clone()

	// Verify basic fields are copied
	if clone.Category != original.Category {
		t.Error("Category not copied")
	}

	// Verify ElementType is deep copied (different pointer)
	if clone.ElementType == original.ElementType {
		t.Error("ElementType not deep copied")
	}

	// Verify EnumValues is deep copied
	clone.EnumValues[0] = "changed"
	if original.EnumValues[0] == "changed" {
		t.Error("EnumValues not deep copied")
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		importPath string
		want       string
	}{
		{"github.com/example/types", "types"},
		{"github.com/example/nested/package", "package"},
		{"fmt", "fmt"},
		{"database/sql", "sql"},
	}

	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			got := ExtractPackageName(tt.importPath)
			if got != tt.want {
				t.Errorf("ExtractPackageName(%q) = %q, want %q", tt.importPath, got, tt.want)
			}
		})
	}
}
