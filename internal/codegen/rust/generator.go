// Package rust generates Rust code using text templates.
package rust

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/types"
)

// File represents a generated file
type File struct {
	Path    string
	Content []byte
}

// TemplateFuncs provides helper functions for templates
var templateFuncs = template.FuncMap{
	"snakeCase":  toSnakeCase,
	"pascalCase": toPascalCase,
	"rustType":   getRustType,
}

// Generator produces Rust code from SQL schemas and queries
type Generator struct {
	tmpl *template.Template
}

// NewGenerator creates a new Rust code generator
func NewGenerator() (*Generator, error) {
	tmpl, err := template.New("rust").Funcs(templateFuncs).ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Generator{tmpl: tmpl}, nil
}

// TableModel represents a table to generate struct for
type TableModel struct {
	Name    string
	Columns []ColumnModel
}

// ColumnModel represents a column in a table
type ColumnModel struct {
	Name         string
	SemanticType types.SemanticType
}

// GenerateModels generates Rust structs from table models
func (g *Generator) GenerateModels(tables []*model.Table) ([]File, error) {
	var files []File

	for _, table := range tables {
		rustModel := g.convertTable(table)

		var buf bytes.Buffer
		if err := g.tmpl.ExecuteTemplate(&buf, "model.tmpl", rustModel); err != nil {
			return nil, fmt.Errorf("generate model %s: %w", table.Name, err)
		}

		files = append(files, File{
			Path:    fmt.Sprintf("models/%s.rs", toSnakeCase(table.Name)),
			Content: buf.Bytes(),
		})
	}

	// Generate models/mod.rs
	if len(tables) > 0 {
		modContent := g.generateModFile(tables)
		files = append(files, File{
			Path:    "models/mod.rs",
			Content: []byte(modContent),
		})
	}

	return files, nil
}

// convertTable converts internal table model to Rust template data
func (g *Generator) convertTable(table *model.Table) map[string]interface{} {
	fields := make([]map[string]interface{}, len(table.Columns))

	mapper := &rustMapper{}

	for i, col := range table.Columns {
		semantic := g.sqliteToSemantic(col.Type, col.NotNull)
		langType := mapper.Map(semantic)

		// Handle nullable types
		typeName := langType.Name
		if semantic.Nullable && !langType.IsNullable {
			typeName = fmt.Sprintf("Option<%s>", typeName)
		}

		fields[i] = map[string]interface{}{
			"Name":     toSnakeCase(col.Name),
			"Type":     typeName,
			"Nullable": semantic.Nullable,
		}
	}

	return map[string]interface{}{
		"Name":   toPascalCase(table.Name),
		"Fields": fields,
	}
}

// sqliteToSemantic converts SQLite type to semantic type
func (g *Generator) sqliteToSemantic(sqlType string, notNull bool) types.SemanticType {
	mapper := types.NewSQLiteMapper()
	return mapper.Map(sqlType, !notNull)
}

// generateModFile generates the models/mod.rs file
func (g *Generator) generateModFile(tables []*model.Table) string {
	var buf strings.Builder
	buf.WriteString("// Generated models module\n\n")

	for _, table := range tables {
		buf.WriteString(fmt.Sprintf("pub mod %s;\n", toSnakeCase(table.Name)))
	}

	buf.WriteString("\n// Re-exports\n")
	for _, table := range tables {
		name := toPascalCase(table.Name)
		buf.WriteString(fmt.Sprintf("pub use %s::%s;\n", toSnakeCase(table.Name), name))
	}

	return buf.String()
}

// rustMapper implements LanguageMapper for Rust
type rustMapper struct{}

func (m *rustMapper) Name() string { return "rust" }

func (m *rustMapper) Map(semantic types.SemanticType) types.LanguageType {
	switch semantic.Category {
	case types.CategoryInteger:
		return types.LanguageType{Name: "i32"}
	case types.CategoryBigInteger, types.CategorySerial, types.CategoryBigSerial:
		return types.LanguageType{Name: "i64"}
	case types.CategorySmallInteger:
		return types.LanguageType{Name: "i16"}
	case types.CategoryTinyInteger:
		return types.LanguageType{Name: "i8"}
	case types.CategoryFloat:
		return types.LanguageType{Name: "f32"}
	case types.CategoryDouble:
		return types.LanguageType{Name: "f64"}
	case types.CategoryDecimal, types.CategoryNumeric:
		return types.LanguageType{
			Name:    "rust_decimal::Decimal",
			Import:  "rust_decimal",
			Package: "rust_decimal",
		}
	case types.CategoryText, types.CategoryChar, types.CategoryVarchar:
		return types.LanguageType{Name: "String"}
	case types.CategoryBlob:
		return types.LanguageType{Name: "Vec<u8>"}
	case types.CategoryBoolean:
		return types.LanguageType{Name: "bool"}
	case types.CategoryTimestamp, types.CategoryTimestampTZ:
		return types.LanguageType{
			Name:    "chrono::DateTime<chrono::Utc>",
			Import:  "chrono",
			Package: "chrono",
		}
	case types.CategoryDate:
		return types.LanguageType{
			Name:    "chrono::NaiveDate",
			Import:  "chrono",
			Package: "chrono",
		}
	case types.CategoryTime:
		return types.LanguageType{
			Name:    "chrono::NaiveTime",
			Import:  "chrono",
			Package: "chrono",
		}
	case types.CategoryUUID:
		return types.LanguageType{
			Name:    "uuid::Uuid",
			Import:  "uuid",
			Package: "uuid",
		}
	case types.CategoryJSON, types.CategoryJSONB:
		return types.LanguageType{
			Name:    "serde_json::Value",
			Import:  "serde_json",
			Package: "serde_json",
		}
	case types.CategoryEnum:
		return types.LanguageType{Name: "String"}
	default:
		return types.LanguageType{Name: "serde_json::Value"}
	}
}

// Helper functions for naming conventions
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toPascalCase(s string) string {
	// First convert to snake_case
	snake := toSnakeCase(s)
	parts := strings.Split(snake, "_")
	for i := range parts {
		//nolint:staticcheck // strings.Title is deprecated but sufficient for ASCII identifiers
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// getRustType is a template function to convert semantic type to Rust type
func getRustType(semantic types.SemanticType) string {
	mapper := &rustMapper{}
	langType := mapper.Map(semantic)

	typeName := langType.Name
	if semantic.Nullable && !langType.IsNullable {
		return fmt.Sprintf("Option<%s>", typeName)
	}
	return typeName
}
