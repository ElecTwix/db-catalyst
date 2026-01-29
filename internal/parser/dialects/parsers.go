// Package dialects implements SQL dialect parsers.
package dialects

import (
	"context"
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/electwix/db-catalyst/internal/parser/grammars"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// DialectParser defines the interface for parsing SQL dialects
type DialectParser interface {
	ParseDDL(ctx context.Context, sql string) (*model.Catalog, error)
	Validate(sql string) ([]string, error)
	Dialect() grammars.Dialect
}

// BaseParser provides common functionality for dialect parsers.
type BaseParser struct {
	dialect grammars.Dialect
}

// NewBaseParser creates a new BaseParser for the given dialect.
func NewBaseParser(dialect grammars.Dialect) *BaseParser {
	return &BaseParser{dialect: dialect}
}

// Dialect returns the dialect of the parser.
func (b *BaseParser) Dialect() grammars.Dialect {
	return b.dialect
}

// Validate validates the SQL syntax for the parser's dialect.
func (b *BaseParser) Validate(sql string) ([]string, error) {
	return grammars.ValidateSyntax(b.dialect, sql)
}

// SQLLexer defines the SQL lexer configuration
var SQLLexer = lexer.MustStateful(lexer.Rules{
	"Root": {
		//nolint:govet // Participle DSL uses unkeyed fields
		{"Whitespace", `[ \t\r\n]+`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"Comment", `--[^\n]*`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"BlockComment", `/\*[\s\S]*?\*/`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"String", `'[^']*'`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"Ident", `[a-zA-Z_][a-zA-Z0-9_]*`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"Number", `[0-9]+(?:\.[0-9]+)?`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"Symbol", `[\(\)\[\]\{\},;:.]`, nil},
		//nolint:govet // Participle DSL uses unkeyed fields
		{"Operator", `[\+\-\*/=<>!]+`, nil},
	},
})

// CreateTable represents a parsed CREATE TABLE statement
//
//nolint:govet // Participle struct tags are DSL, not reflect tags
type CreateTable struct {
	Keyword string    `@"CREATE"`
	Table   string    `@"TABLE"`
	Name    string    `@Ident`
	Columns []*Column `"(" @@ ("," @@)* ")"`
}

// Column represents a table column
//
//nolint:govet // Participle struct tags are DSL, not reflect tags
type Column struct {
	Name       string `@Ident`
	Type       string `@Ident`
	Constraint string `(@("PRIMARY" "KEY") | "NOT" "NULL" | "UNIQUE")?`
}

// Constraint represents a column constraint
//
//nolint:govet // Participle struct tags are DSL, not reflect tags
type Constraint struct {
	Type string `(@Ident ("KEY" | @Ident?) | "NOT" "NULL" | "UNIQUE")`
}

// SQLiteParser implements parsing for SQLite dialect.
type SQLiteParser struct {
	*BaseParser
	parser *participle.Parser[CreateTable]
}

// NewSQLiteParser creates a new SQLite parser.
func NewSQLiteParser() *SQLiteParser {
	parser, err := participle.Build[CreateTable](
		participle.Lexer(SQLLexer),
		participle.CaseInsensitive("CREATE", "TABLE", "PRIMARY", "KEY", "NOT", "NULL", "UNIQUE"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to build SQLite parser: %v", err))
	}

	return &SQLiteParser{
		BaseParser: NewBaseParser(grammars.DialectSQLite),
		parser:     parser,
	}
}

// ParseDDL parses SQLite DDL and returns a catalog.
//
//nolint:revive // Context parameter reserved for future use
func (s *SQLiteParser) ParseDDL(_ context.Context, sql string) (*model.Catalog, error) {
	stmt, err := s.parser.ParseString("", sql)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQLite DDL: %w", err)
	}

	catalog := model.NewCatalog()
	table := &model.Table{
		Name:    stmt.Name,
		Columns: make([]*model.Column, 0, len(stmt.Columns)),
	}

	for _, col := range stmt.Columns {
		table.Columns = append(table.Columns, &model.Column{
			Name: col.Name,
			Type: col.Type,
		})
	}

	catalog.Tables[stmt.Name] = table
	return catalog, nil
}

// PostgreSQLParser implements parsing for PostgreSQL dialect.
type PostgreSQLParser struct {
	*BaseParser
	parser *participle.Parser[CreateTable]
}

// NewPostgreSQLParser creates a new PostgreSQL parser.
func NewPostgreSQLParser() *PostgreSQLParser {
	parser, err := participle.Build[CreateTable](
		participle.Lexer(SQLLexer),
		participle.CaseInsensitive("CREATE", "TABLE", "PRIMARY", "KEY", "NOT", "NULL", "UNIQUE"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to build PostgreSQL parser: %v", err))
	}

	return &PostgreSQLParser{
		BaseParser: NewBaseParser(grammars.DialectPostgreSQL),
		parser:     parser,
	}
}

// ParseDDL parses PostgreSQL DDL and returns a catalog.
//
//nolint:revive // Context parameter reserved for future use
func (p *PostgreSQLParser) ParseDDL(_ context.Context, sql string) (*model.Catalog, error) {
	stmt, err := p.parser.ParseString("", sql)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL DDL: %w", err)
	}

	catalog := model.NewCatalog()
	table := &model.Table{
		Name:    stmt.Name,
		Columns: make([]*model.Column, 0, len(stmt.Columns)),
	}

	for _, col := range stmt.Columns {
		table.Columns = append(table.Columns, &model.Column{
			Name: col.Name,
			Type: col.Type,
		})
	}

	catalog.Tables[stmt.Name] = table
	return catalog, nil
}

// MySQLParser implements parsing for MySQL dialect.
type MySQLParser struct {
	*BaseParser
	parser *participle.Parser[CreateTable]
}

// NewMySQLParser creates a new MySQL parser.
func NewMySQLParser() *MySQLParser {
	parser, err := participle.Build[CreateTable](
		participle.Lexer(SQLLexer),
		participle.CaseInsensitive("CREATE", "TABLE", "PRIMARY", "KEY", "NOT", "NULL", "UNIQUE"),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to build MySQL parser: %v", err))
	}

	return &MySQLParser{
		BaseParser: NewBaseParser(grammars.DialectMySQL),
		parser:     parser,
	}
}

// ParseDDL parses MySQL DDL and returns a catalog.
//
//nolint:revive // Context parameter reserved for future use
func (m *MySQLParser) ParseDDL(_ context.Context, sql string) (*model.Catalog, error) {
	stmt, err := m.parser.ParseString("", sql)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MySQL DDL: %w", err)
	}

	catalog := model.NewCatalog()
	table := &model.Table{
		Name:    stmt.Name,
		Columns: make([]*model.Column, 0, len(stmt.Columns)),
	}

	for _, col := range stmt.Columns {
		table.Columns = append(table.Columns, &model.Column{
			Name: col.Name,
			Type: col.Type,
		})
	}

	catalog.Tables[stmt.Name] = table
	return catalog, nil
}

// NewParser creates a new dialect parser for the specified dialect
func NewParser(dialect grammars.Dialect) (DialectParser, error) {
	switch dialect {
	case grammars.DialectSQLite:
		return NewSQLiteParser(), nil
	case grammars.DialectPostgreSQL:
		return NewPostgreSQLParser(), nil
	case grammars.DialectMySQL:
		return NewMySQLParser(), nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}
}
