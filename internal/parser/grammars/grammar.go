package grammars

import (
	"fmt"
	"strings"
)

// Dialect represents a SQL dialect
type Dialect string

const (
	DialectSQLite     Dialect = "sqlite"
	DialectPostgreSQL Dialect = "postgresql"
	DialectMySQL      Dialect = "mysql"
)

// GrammarDefinition represents a parsed SQL grammar
type GrammarDefinition struct {
	Statements []*StatementDefinition `@@*`
}

// StatementDefinition represents a SQL statement type
type StatementDefinition struct {
	Name    string            `@Ident`
	Pattern string            `@String`
	Rules   []*RuleDefinition `@@*"`
}

// RuleDefinition represents a grammar rule
type RuleDefinition struct {
	Name     string        `@Ident "="`
	Variants []*VariantDef `"|"? @@*`
}

// VariantDef represents a rule variant
type VariantDef struct {
	Tokens []string `@Ident+`
}

// GetDialectGrammar returns the grammar for a given dialect
func GetDialectGrammar(dialect Dialect) (string, error) {
	switch dialect {
	case DialectSQLite:
		return SQLiteGrammar, nil
	case DialectPostgreSQL:
		return PostgreSQLGrammar, nil
	case DialectMySQL:
		return MySQLGrammar, nil
	default:
		return "", fmt.Errorf("unsupported dialect: %s", dialect)
	}
}

// ValidateSyntax checks if SQL matches the dialect's syntax rules
func ValidateSyntax(dialect Dialect, sql string) ([]string, error) {
	_, err := GetDialectGrammar(dialect)
	if err != nil {
		return nil, err
	}

	var issues []string

	upperSQL := strings.ToUpper(sql)

	switch dialect {
	case DialectSQLite:
		issues = validateSQLite(upperSQL)
	case DialectPostgreSQL:
		issues = validatePostgreSQL(upperSQL)
	case DialectMySQL:
		issues = validateMySQL(upperSQL)
	}

	return issues, nil
}

func validateSQLite(sql string) []string {
	var issues []string

	// SQLite-specific validations
	if strings.Contains(sql, " SERIAL") {
		issues = append(issues, "SERIAL is not valid in SQLite (use INTEGER PRIMARY KEY)")
	}
	if strings.Contains(sql, " AUTO_INCREMENT") {
		issues = append(issues, "AUTO_INCREMENT is MySQL syntax (use AUTOINCREMENT)")
	}
	if strings.Contains(sql, " JSONB") {
		issues = append(issues, "JSONB is PostgreSQL-specific (use TEXT or JSON)")
	}

	return issues
}

func validatePostgreSQL(sql string) []string {
	var issues []string

	// PostgreSQL-specific validations
	if strings.Contains(sql, " AUTOINCREMENT") {
		issues = append(issues, "AUTOINCREMENT is SQLite syntax (use SERIAL)")
	}
	if strings.Contains(sql, " WITHOUT ROWID") {
		issues = append(issues, "WITHOUT ROWID is SQLite-specific")
	}
	if strings.Contains(sql, " INTEGER PRIMARY KEY") && strings.Contains(sql, " AUTOINCREMENT") {
		issues = append(issues, "AUTOINCREMENT with INTEGER PRIMARY KEY is SQLite-specific (use SERIAL or BIGSERIAL)")
	}

	return issues
}

func validateMySQL(sql string) []string {
	var issues []string

	// MySQL-specific validations
	if strings.Contains(sql, " SERIAL") {
		issues = append(issues, "SERIAL is PostgreSQL syntax (use AUTO_INCREMENT)")
	}
	if strings.Contains(sql, " WITHOUT ROWID") {
		issues = append(issues, "WITHOUT ROWID is SQLite-specific")
	}
	if strings.Contains(sql, " JSONB") {
		issues = append(issues, "JSONB is PostgreSQL-specific (use JSON)")
	}

	return issues
}
