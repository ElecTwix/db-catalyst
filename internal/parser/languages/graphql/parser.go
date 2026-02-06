// Package graphql implements a GraphQL schema parser.
//
//nolint:goconst // SQL type names are naturally repeated
package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// Schema represents a parsed GraphQL schema.
//
//nolint:govet // Participle struct tags are DSL, not reflect tags
type Schema struct {
	Types []*TypeDefinition `@@*`
}

// TypeDefinition represents a GraphQL type definition.
//
//nolint:govet // Participle struct tags are DSL, not reflect tags
type TypeDefinition struct {
	Name   string             `@("type" @Ident)`
	Fields []*FieldDefinition `"{" @@+ "}"`
}

// FieldDefinition represents a GraphQL field definition.
//
//nolint:govet // Participle struct tags are DSL, not reflect tags
type FieldDefinition struct {
	Name string `@Ident`
	Type string `@(":" @Ident)`
}

// GraphQLLexer defines the GraphQL lexer configuration.
//
//nolint:govet // Participle DSL uses unkeyed fields
var GraphQLLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Whitespace", `[ \t\r\n]+`},
	{"Comment", `#[^\n]*`},
	{"String", `"[^"]*" `},
	{"Ident", `[A-Za-z_][A-Za-z0-9_]*`},
	{"Number", `-?[0-9]+(\.[0-9]+)?`},
	{"Symbol", `[()\[\]{}:,|]`},
	{"Operator", `[=!@&]`},
})

// Parser implements a GraphQL schema parser.
type Parser struct {
	parser *participle.Parser[Schema]
}

// NewParser creates a new GraphQL parser.
func NewParser() (*Parser, error) {
	parser, err := participle.Build[Schema](
		participle.Lexer(GraphQLLexer),
		participle.CaseInsensitive("type", "interface", "input", "enum", "scalar", "implements", "extends", "true", "false", "null"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build GraphQL parser: %w", err)
	}
	return &Parser{parser: parser}, nil
}

// ParseSchema parses a GraphQL schema and returns a catalog.
func (p *Parser) ParseSchema(_ context.Context, schema string) (*model.Catalog, error) {
	graphqlSchema, err := p.parser.ParseString("", schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL schema: %w", err)
	}

	catalog := model.NewCatalog()

	for _, typ := range graphqlSchema.Types {
		table := &model.Table{
			Name:    typ.Name,
			Columns: make([]*model.Column, 0, len(typ.Fields)),
		}

		for _, field := range typ.Fields {
			column := &model.Column{
				Name: field.Name,
				Type: p.mapGraphQLTypeToSQLite(field.Type),
			}

			if strings.ToLower(field.Name) == "id" {
				table.PrimaryKey = &model.PrimaryKey{
					Name:    "pk_" + typ.Name,
					Columns: []string{field.Name},
				}
			}

			table.Columns = append(table.Columns, column)
		}

		catalog.Tables[typ.Name] = table
	}

	return catalog, nil
}

// Validate validates a GraphQL schema and returns any issues.
func (p *Parser) Validate(schema string) ([]string, error) {
	var issues []string

	_, err := p.parser.ParseString("", schema)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Parse error: %v", err))
	}

	return issues, nil
}

func (p *Parser) mapGraphQLTypeToSQLite(graphqlType string) string {
	baseType := strings.TrimSuffix(strings.TrimSuffix(graphqlType, "!"), "[]")

	switch strings.ToUpper(baseType) {
	case "ID":
		return "INTEGER"
	case "INT", "INTEGER":
		return "INTEGER"
	case "FLOAT":
		return "REAL"
	case "STRING", "VARCHAR", "TEXT":
		return "TEXT"
	case "BOOLEAN", "BOOL":
		return "INTEGER"
	case "DATE":
		return "TEXT"
	case "DATETIME", "TIMESTAMP":
		return "TEXT"
	case "JSON":
		return "TEXT"
	default:
		return "TEXT"
	}
}
