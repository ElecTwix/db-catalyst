package types

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/config"
)

func TestPostgresMapper(t *testing.T) {
	mapper := NewPostgresMapper()

	tests := []struct {
		name     string
		sqlType  string
		nullable bool
		want     SemanticType
	}{
		// Integer types
		{
			name:     "SMALLINT",
			sqlType:  "SMALLINT",
			nullable: false,
			want:     SemanticType{Category: CategorySmallInteger, Nullable: false},
		},
		{
			name:     "INTEGER",
			sqlType:  "INTEGER",
			nullable: true,
			want:     SemanticType{Category: CategoryInteger, Nullable: true},
		},
		{
			name:     "BIGINT",
			sqlType:  "BIGINT",
			nullable: false,
			want:     SemanticType{Category: CategoryBigInteger, Nullable: false},
		},
		{
			name:     "SERIAL",
			sqlType:  "SERIAL",
			nullable: false,
			want:     SemanticType{Category: CategorySerial, Nullable: false},
		},
		{
			name:     "BIGSERIAL",
			sqlType:  "BIGSERIAL",
			nullable: false,
			want:     SemanticType{Category: CategoryBigSerial, Nullable: false},
		},

		// Float types
		{
			name:     "REAL",
			sqlType:  "REAL",
			nullable: false,
			want:     SemanticType{Category: CategoryFloat, Nullable: false},
		},
		{
			name:     "DOUBLE PRECISION",
			sqlType:  "DOUBLE PRECISION",
			nullable: false,
			want:     SemanticType{Category: CategoryDouble, Nullable: false},
		},

		// Decimal
		{
			name:     "NUMERIC with precision and scale",
			sqlType:  "NUMERIC(10,2)",
			nullable: false,
			want:     SemanticType{Category: CategoryDecimal, Nullable: false, Precision: 10, Scale: 2},
		},
		{
			name:     "MONEY",
			sqlType:  "MONEY",
			nullable: false,
			want:     SemanticType{Category: CategoryDecimal, Nullable: false, Precision: 19, Scale: 2},
		},

		// String types
		{
			name:     "TEXT",
			sqlType:  "TEXT",
			nullable: false,
			want:     SemanticType{Category: CategoryText, Nullable: false},
		},
		{
			name:     "VARCHAR with length",
			sqlType:  "VARCHAR(255)",
			nullable: false,
			want:     SemanticType{Category: CategoryVarchar, Nullable: false, MaxLength: 255},
		},
		{
			name:     "CHAR",
			sqlType:  "CHAR(10)",
			nullable: false,
			want:     SemanticType{Category: CategoryChar, Nullable: false, MaxLength: 10},
		},
		{
			name:     "BYTEA",
			sqlType:  "BYTEA",
			nullable: false,
			want:     SemanticType{Category: CategoryBlob, Nullable: false},
		},

		// Boolean
		{
			name:     "BOOLEAN",
			sqlType:  "BOOLEAN",
			nullable: true,
			want:     SemanticType{Category: CategoryBoolean, Nullable: true},
		},

		// Temporal types
		{
			name:     "DATE",
			sqlType:  "DATE",
			nullable: false,
			want:     SemanticType{Category: CategoryDate, Nullable: false},
		},
		{
			name:     "TIMESTAMP",
			sqlType:  "TIMESTAMP",
			nullable: false,
			want:     SemanticType{Category: CategoryTimestamp, Nullable: false},
		},
		{
			name:     "TIMESTAMPTZ",
			sqlType:  "TIMESTAMPTZ",
			nullable: false,
			want:     SemanticType{Category: CategoryTimestampTZ, Nullable: false},
		},
		{
			name:     "TIME",
			sqlType:  "TIME",
			nullable: false,
			want:     SemanticType{Category: CategoryTime, Nullable: false},
		},
		{
			name:     "INTERVAL",
			sqlType:  "INTERVAL",
			nullable: false,
			want:     SemanticType{Category: CategoryInterval, Nullable: false},
		},

		// Special types
		{
			name:     "UUID",
			sqlType:  "UUID",
			nullable: false,
			want:     SemanticType{Category: CategoryUUID, Nullable: false},
		},
		{
			name:     "JSON",
			sqlType:  "JSON",
			nullable: false,
			want:     SemanticType{Category: CategoryJSON, Nullable: false},
		},
		{
			name:     "JSONB",
			sqlType:  "JSONB",
			nullable: false,
			want:     SemanticType{Category: CategoryJSONB, Nullable: false},
		},
		{
			name:     "XML",
			sqlType:  "XML",
			nullable: false,
			want:     SemanticType{Category: CategoryXML, Nullable: false},
		},

		// Arrays
		{
			name:    "TEXT array",
			sqlType: "TEXT[]",
			want: SemanticType{
				Category: CategoryArray,
				Nullable: false,
				ElementType: &SemanticType{
					Category: CategoryText,
					Nullable: false,
				},
			},
		},
		{
			name:    "INTEGER array",
			sqlType: "INTEGER[]",
			want: SemanticType{
				Category: CategoryArray,
				Nullable: false,
				ElementType: &SemanticType{
					Category: CategoryInteger,
					Nullable: false,
				},
			},
		},

		// Network types (map to string)
		{
			name:     "INET",
			sqlType:  "INET",
			nullable: false,
			want:     SemanticType{Category: CategoryText, Nullable: false},
		},
		{
			name:     "CIDR",
			sqlType:  "CIDR",
			nullable: false,
			want:     SemanticType{Category: CategoryText, Nullable: false},
		},

		// OID types (map to int64)
		{
			name:     "OID",
			sqlType:  "OID",
			nullable: false,
			want:     SemanticType{Category: CategoryBigInteger, Nullable: false},
		},
		{
			name:     "REGCLASS",
			sqlType:  "REGCLASS",
			nullable: false,
			want:     SemanticType{Category: CategoryBigInteger, Nullable: false},
		},

		// Custom type
		{
			name:     "Custom type",
			sqlType:  "MY_CUSTOM_TYPE",
			nullable: false,
			want:     SemanticType{Category: CategoryCustom, Nullable: false, CustomName: "MY_CUSTOM_TYPE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if got.CustomName != tt.want.CustomName {
				t.Errorf("CustomName = %v, want %v", got.CustomName, tt.want.CustomName)
			}

			// Check array element type
			if tt.want.ElementType != nil {
				if got.ElementType == nil {
					t.Error("ElementType is nil, want non-nil")
				} else if got.ElementType.Category != tt.want.ElementType.Category {
					t.Errorf("ElementType.Category = %v, want %v",
						got.ElementType.Category, tt.want.ElementType.Category)
				}
			}
		})
	}
}

func TestParsePostgresType(t *testing.T) {
	tests := []struct {
		sqlType   string
		wantBase  string
		wantLen   int
		wantPrec  int
		wantScale int
	}{
		{"VARCHAR(255)", "VARCHAR", 255, 0, 0},
		{"CHAR(10)", "CHAR", 10, 0, 0},
		{"NUMERIC(10,2)", "NUMERIC", 0, 10, 2},
		{"DECIMAL(5)", "DECIMAL", 0, 5, 0},
		{"TEXT", "TEXT", -1, 0, 0},
		{"INTEGER", "INTEGER", -1, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			base, length, prec, scale := parsePostgresType(tt.sqlType)

			if base != tt.wantBase {
				t.Errorf("base = %q, want %q", base, tt.wantBase)
			}
			if length != tt.wantLen {
				t.Errorf("length = %d, want %d", length, tt.wantLen)
			}
			if prec != tt.wantPrec {
				t.Errorf("precision = %d, want %d", prec, tt.wantPrec)
			}
			if scale != tt.wantScale {
				t.Errorf("scale = %d, want %d", scale, tt.wantScale)
			}
		})
	}
}

func TestGoPostgresMapper(t *testing.T) {
	mapper := NewGoPostgresMapper(nil)

	tests := []struct {
		name        string
		semantic    SemanticType
		wantType    string
		wantImport  string
		wantPackage string
	}{
		{
			name:     "integer",
			semantic: SemanticType{Category: CategoryInteger},
			wantType: "int32",
		},
		{
			name:     "bigint",
			semantic: SemanticType{Category: CategoryBigInteger},
			wantType: "int64",
		},
		{
			name:     "text",
			semantic: SemanticType{Category: CategoryText},
			wantType: "string",
		},
		{
			name:        "timestamp",
			semantic:    SemanticType{Category: CategoryTimestamp},
			wantType:    "time.Time",
			wantImport:  "time",
			wantPackage: "time",
		},
		{
			name:        "date",
			semantic:    SemanticType{Category: CategoryDate},
			wantType:    "pgtype.Date",
			wantImport:  "github.com/jackc/pgx/v5/pgtype",
			wantPackage: "pgtype",
		},
		{
			name:        "uuid",
			semantic:    SemanticType{Category: CategoryUUID},
			wantType:    "uuid.UUID",
			wantImport:  "github.com/google/uuid",
			wantPackage: "uuid",
		},
		{
			name:        "decimal",
			semantic:    SemanticType{Category: CategoryDecimal, Precision: 10, Scale: 2},
			wantType:    "decimal.Decimal",
			wantImport:  "github.com/shopspring/decimal",
			wantPackage: "decimal",
		},
		{
			name:     "bytea",
			semantic: SemanticType{Category: CategoryBlob},
			wantType: "[]byte",
		},
		{
			name:        "json",
			semantic:    SemanticType{Category: CategoryJSON},
			wantType:    "[]byte",
			wantImport:  "encoding/json",
			wantPackage: "json",
		},
		{
			name: "text array",
			semantic: SemanticType{
				Category:    CategoryArray,
				ElementType: &SemanticType{Category: CategoryText},
			},
			wantType: "[]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.Map(tt.semantic)

			if got.Name != tt.wantType {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantType)
			}
			if got.Import != tt.wantImport {
				t.Errorf("Import = %q, want %q", got.Import, tt.wantImport)
			}
			if got.Package != tt.wantPackage {
				t.Errorf("Package = %q, want %q", got.Package, tt.wantPackage)
			}
		})
	}
}

func TestGoPostgresMapperWithCustomTypes(t *testing.T) {
	customTypes := []config.CustomTypeMapping{
		{
			CustomType: "user_id",
			GoType:     "UserID",
			GoImport:   "github.com/example/types",
			GoPackage:  "types",
		},
	}

	mapper := NewGoPostgresMapper(customTypes)

	semantic := SemanticType{
		Category:   CategoryCustom,
		CustomName: "user_id",
	}

	got := mapper.Map(semantic)

	if got.Name != "UserID" {
		t.Errorf("Name = %q, want UserID", got.Name)
	}
	if got.Import != "github.com/example/types" {
		t.Errorf("Import = %q, want github.com/example/types", got.Import)
	}
}

func TestPostgresUtilityFunctions(t *testing.T) {
	t.Run("GetDriverImport", func(t *testing.T) {
		if got := GetDriverImport(); got != "github.com/jackc/pgx/v5/pgxpool" {
			t.Errorf("GetDriverImport() = %q, want github.com/jackc/pgx/v5/pgxpool", got)
		}
	})

	t.Run("PostgresDriverType", func(t *testing.T) {
		if got := PostgresDriverType(); got != "pgx" {
			t.Errorf("PostgresDriverType() = %q, want pgx", got)
		}
	})

	t.Run("GetNullPackage", func(t *testing.T) {
		importPath, pkg := GetNullPackage()
		if importPath != "github.com/jackc/pgx/v5/pgtype" {
			t.Errorf("importPath = %q, want github.com/jackc/pgx/v5/pgtype", importPath)
		}
		if pkg != "pgtype" {
			t.Errorf("package = %q, want pgtype", pkg)
		}
	})

	t.Run("PostgresConnectionString", func(t *testing.T) {
		connStr := PostgresConnectionString("localhost", 5432, "mydb", "user", "pass")
		want := "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
		if connStr != want {
			t.Errorf("PostgresConnectionString() = %q, want %q", connStr, want)
		}
	})
}

func TestGetPgxScanFunc(t *testing.T) {
	tests := []struct {
		semantic SemanticType
		want     string
	}{
		{SemanticType{Category: CategoryInteger}, "ScanInt32"},
		{SemanticType{Category: CategoryBigInteger}, "ScanInt64"},
		{SemanticType{Category: CategoryFloat}, "ScanFloat32"},
		{SemanticType{Category: CategoryDouble}, "ScanFloat64"},
		{SemanticType{Category: CategoryBoolean}, "ScanBool"},
		{SemanticType{Category: CategoryText}, "ScanString"},
		{SemanticType{Category: CategoryTimestamp}, "ScanTime"},
		{SemanticType{Category: CategoryCustom}, "Scan"},
	}

	for _, tt := range tests {
		t.Run(tt.semantic.Category.String(), func(t *testing.T) {
			got := GetPgxScanFunc(tt.semantic)
			if got != tt.want {
				t.Errorf("GetPgxScanFunc() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPgxNativeType(t *testing.T) {
	// Test native types
	nativeTypes := []SemanticTypeCategory{
		CategoryInteger, CategoryBigInteger, CategoryFloat,
		CategoryDouble, CategoryBoolean, CategoryText, CategoryTimestamp,
	}

	for _, cat := range nativeTypes {
		t.Run(cat.String(), func(t *testing.T) {
			if !IsPgxNativeType(SemanticType{Category: cat}) {
				t.Errorf("IsPgxNativeType(%v) = false, want true", cat)
			}
		})
	}

	// Test non-native types
	nonNativeTypes := []SemanticTypeCategory{
		CategoryUUID, CategoryJSON, CategoryArray,
	}

	for _, cat := range nonNativeTypes {
		t.Run(cat.String(), func(t *testing.T) {
			if IsPgxNativeType(SemanticType{Category: cat}) {
				t.Errorf("IsPgxNativeType(%v) = true, want false", cat)
			}
		})
	}
}

func TestGetPostgresArrayType(t *testing.T) {
	tests := []struct {
		elementType string
		want        string
	}{
		{"int", "pgtype.Array[int32]"},
		{"int32", "pgtype.Array[int32]"},
		{"int64", "pgtype.Array[int64]"},
		{"string", "pgtype.Array[string]"},
		{"float64", "pgtype.Array[float64]"},
		{"bool", "pgtype.Array[bool]"},
		{"CustomType", "pgtype.Array[CustomType]"},
	}

	for _, tt := range tests {
		t.Run(tt.elementType, func(t *testing.T) {
			got := GetPostgresArrayType(tt.elementType)
			if got != tt.want {
				t.Errorf("GetPostgresArrayType(%q) = %q, want %q", tt.elementType, got, tt.want)
			}
		})
	}
}
