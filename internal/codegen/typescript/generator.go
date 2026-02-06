// Package typescript generates TypeScript code using text templates.
package typescript

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
	"camelCase":      toCamelCase,
	"pascalCase":     toPascalCase,
	"typescriptType": getTypescriptType,
}

// Generator produces TypeScript code from SQL schemas and queries
type Generator struct {
	tmpl *template.Template
}

// NewGenerator creates a new TypeScript code generator
func NewGenerator() (*Generator, error) {
	tmpl, err := template.New("typescript").Funcs(templateFuncs).ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Generator{tmpl: tmpl}, nil
}

// GenerateModels generates TypeScript interfaces from table models
func (g *Generator) GenerateModels(tables []*model.Table) ([]File, error) {
	var files []File

	for _, table := range tables {
		tsModel := g.convertTable(table)

		var buf bytes.Buffer
		if err := g.tmpl.ExecuteTemplate(&buf, "model.tmpl", tsModel); err != nil {
			return nil, fmt.Errorf("generate model %s: %w", table.Name, err)
		}

		files = append(files, File{
			Path:    fmt.Sprintf("models/%s.ts", toCamelCase(table.Name)),
			Content: buf.Bytes(),
		})
	}

	// Generate models/index.ts
	if len(tables) > 0 {
		indexContent := g.generateIndexFile(tables)
		files = append(files, File{
			Path:    "models/index.ts",
			Content: []byte(indexContent),
		})
	}

	return files, nil
}

// convertTable converts internal table model to TypeScript template data
func (g *Generator) convertTable(table *model.Table) map[string]any {
	fields := make([]map[string]any, len(table.Columns))

	mapper := &typescriptMapper{}

	for i, col := range table.Columns {
		semantic := g.sqliteToSemantic(col.Type, col.NotNull)
		langType := mapper.Map(semantic)

		// Handle nullable types
		typeName := langType.Name
		if semantic.Nullable && !langType.IsNullable {
			typeName += " | null"
		}

		fields[i] = map[string]any{
			"Name":     toCamelCase(col.Name),
			"Type":     typeName,
			"Nullable": semantic.Nullable,
		}
	}

	return map[string]any{
		"Name":   toPascalCase(table.Name),
		"Fields": fields,
	}
}

// sqliteToSemantic converts SQLite type to semantic type
func (g *Generator) sqliteToSemantic(sqlType string, notNull bool) types.SemanticType {
	mapper := types.NewSQLiteMapper()
	return mapper.Map(sqlType, !notNull)
}

// generateIndexFile generates the models/index.ts file
func (g *Generator) generateIndexFile(tables []*model.Table) string {
	var buf strings.Builder
	buf.WriteString("// Generated models module\n\n")

	for _, table := range tables {
		buf.WriteString(fmt.Sprintf("export * from './%s';\n", toCamelCase(table.Name)))
	}

	return buf.String()
}

// typescriptMapper implements LanguageMapper for TypeScript
type typescriptMapper struct{}

func (m *typescriptMapper) Name() string { return "typescript" }

func (m *typescriptMapper) Map(semantic types.SemanticType) types.LanguageType {
	switch semantic.Category {
	case types.CategoryInteger, types.CategorySmallInteger, types.CategoryTinyInteger:
		return types.LanguageType{Name: "number"}
	case types.CategoryBigInteger, types.CategorySerial, types.CategoryBigSerial:
		return types.LanguageType{Name: "number"} // TypeScript only has number
	case types.CategoryFloat, types.CategoryDouble, types.CategoryDecimal, types.CategoryNumeric:
		return types.LanguageType{Name: "number"}
	case types.CategoryText, types.CategoryChar, types.CategoryVarchar:
		return types.LanguageType{Name: "string"}
	case types.CategoryBlob:
		return types.LanguageType{Name: "Buffer"}
	case types.CategoryBoolean:
		return types.LanguageType{Name: "boolean"}
	case types.CategoryTimestamp, types.CategoryTimestampTZ, types.CategoryDate, types.CategoryTime:
		return types.LanguageType{Name: "Date"}
	case types.CategoryUUID:
		return types.LanguageType{Name: "string"}
	case types.CategoryJSON, types.CategoryJSONB:
		return types.LanguageType{Name: "any", IsNullable: true}
	case types.CategoryEnum:
		return types.LanguageType{Name: "string"}
	case types.CategoryArray:
		if semantic.ElementType != nil {
			elementMapper := &typescriptMapper{}
			elementType := elementMapper.Map(*semantic.ElementType)
			return types.LanguageType{Name: elementType.Name + "[]"}
		}
		return types.LanguageType{Name: "any[]"}
	default:
		return types.LanguageType{Name: "any", IsNullable: true}
	}
}

// Helper functions for naming conventions
func toCamelCase(s string) string {
	// Convert to snake_case first
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	snake := strings.ToLower(result.String())

	// Convert to camelCase
	parts := strings.Split(snake, "_")
	for i := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(parts[i])
		} else {
			//nolint:staticcheck // strings.Title is deprecated but sufficient for ASCII identifiers
			parts[i] = strings.Title(parts[i])
		}
	}
	return strings.Join(parts, "")
}

func toPascalCase(s string) string {
	// Convert to snake_case first
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	snake := strings.ToLower(result.String())

	// Convert to PascalCase
	parts := strings.Split(snake, "_")
	for i := range parts {
		//nolint:staticcheck // strings.Title is deprecated but sufficient for ASCII identifiers
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// getTypescriptType is a template function to convert semantic type to TypeScript type
func getTypescriptType(semantic types.SemanticType) string {
	mapper := &typescriptMapper{}
	langType := mapper.Map(semantic)

	typeName := langType.Name
	if semantic.Nullable && !langType.IsNullable {
		return typeName + " | null"
	}
	return typeName
}
