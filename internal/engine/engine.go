// Package engine defines the database engine abstraction layer.
//
// The engine package provides interfaces that encapsulate all database-specific
// behavior, enabling true multi-database support. Each database dialect
// (SQLite, PostgreSQL, MySQL) implements the Engine interface.
//
// Usage:
//
//	engine, err := engine.New("postgresql")
//	if err != nil {
//	    return err
//	}
//
//	typeMapper := engine.TypeMapper()
//	typeInfo := typeMapper.SQLToGo("UUID", false)
//
//	parser := engine.SchemaParser()
//	catalog, diags, err := parser.Parse(ctx, path, content)
package engine

import (
	"time"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	"github.com/electwix/db-catalyst/internal/types"
)

// Engine encapsulates all database-specific behavior.
// Each database dialect implements this interface to provide its
// unique characteristics for schema parsing, type mapping, and code generation.
type Engine interface {
	// Name returns the engine identifier (e.g., "sqlite", "postgresql", "mysql").
	Name() string

	// TypeMapper returns the type mapper for SQL-to-language conversions.
	TypeMapper() TypeMapper

	// SchemaParser returns the DDL parser for this database dialect.
	SchemaParser() schemaparser.SchemaParser

	// SQLGenerator returns the SQL DDL generator for this dialect.
	SQLGenerator() SQLGenerator

	// DefaultDriver returns the default Go driver import path for this database.
	DefaultDriver() string

	// SupportsFeature reports whether this engine supports a specific feature.
	SupportsFeature(feature Feature) bool

	// ConnectionPool returns recommended connection pool settings.
	ConnectionPool() ConnectionPoolConfig

	// IsolationLevels returns supported isolation levels and default.
	IsolationLevels() (supported []IsolationLevel, defaultLevel IsolationLevel)

	// QueryHints returns available query hints for this database.
	QueryHints() []QueryHint
}

// ConnectionPoolConfig defines recommended connection pool settings for a database.
type ConnectionPoolConfig struct {
	// MaxOpenConns is the maximum number of open connections to the database.
	// A value of 0 means no limit.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of connections in the idle connection pool.
	// A value of 0 means no idle connections are retained.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Expired connections may be closed lazily before reuse.
	// A value of 0 means connections are not closed due to age.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle
	// before being closed.
	// A value of 0 means connections are not closed due to inactivity.
	ConnMaxIdleTime time.Duration
}

// QueryHint represents a database-specific query optimization hint.
type QueryHint struct {
	// Name is the identifier for this hint (e.g., "INDEX", "USE_INDEX").
	Name string

	// Description explains what this hint does and when to use it.
	Description string

	// Syntax shows the SQL syntax for this hint.
	// Example: "/*+ INDEX(table_name index_name) */" for Oracle-style hints.
	Syntax string
}

// TypeMapper handles SQL type to programming language type conversions.
type TypeMapper interface {
	// SQLToGo converts a SQL type to Go type information.
	// The nullable parameter indicates if the column allows NULL values.
	SQLToGo(sqlType string, nullable bool) TypeInfo

	// SQLToSemantic converts a SQL type to a semantic type category.
	// This is used for cross-language type mapping.
	SQLToSemantic(sqlType string, nullable bool) types.SemanticType

	// GetRequiredImports returns the imports needed for generated code.
	// The returned map is from import path to package alias.
	GetRequiredImports() map[string]string

	// SupportsPointersForNull reports whether this engine supports using
	// pointers for nullable types instead of sql.Null* types.
	SupportsPointersForNull() bool
}

// TypeInfo describes a resolved programming language type.
type TypeInfo struct {
	// GoType is the Go type name (e.g., "string", "int64", "uuid.UUID").
	GoType string

	// UsesSQLNull indicates if this type uses database/sql null types
	// (e.g., sql.NullString, sql.NullInt64).
	UsesSQLNull bool

	// Import is the import path if an external package is needed.
	// Empty for standard library types.
	Import string

	// Package is the package name for the import (e.g., "uuid", "sql").
	// Empty for standard library types.
	Package string

	// IsPointer indicates if this type is naturally a pointer.
	IsPointer bool
}

// SQLGenerator generates SQL DDL statements in a specific dialect.
type SQLGenerator interface {
	// GenerateTable creates a CREATE TABLE statement for the given table.
	GenerateTable(table *model.Table) string

	// GenerateIndex creates a CREATE INDEX statement.
	GenerateIndex(index *model.Index, tableName string) string

	// GenerateColumnDef creates a column definition clause.
	GenerateColumnDef(column *model.Column) string

	// Dialect returns the target SQL dialect identifier.
	Dialect() string
}

// QueryAnalyzer provides dialect-specific query validation and optimization.
type QueryAnalyzer interface {
	// ValidateQuery checks if a query is valid for this dialect.
	// Returns a list of diagnostics for any issues found.
	ValidateQuery(query string) []Diagnostic

	// SuggestIndexes recommends indexes that could improve query performance.
	SuggestIndexes(query string, catalog *model.Catalog) []IndexSuggestion
}

// Diagnostic represents an issue found during query analysis.
type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

// Severity indicates the seriousness of a diagnostic.
type Severity int

const (
	// SeverityWarning indicates a potential issue that doesn't prevent code generation.
	SeverityWarning Severity = iota
	// SeverityError indicates a fatal issue that prevents code generation.
	SeverityError
)

// IndexSuggestion represents a recommended index for query optimization.
type IndexSuggestion struct {
	Table   string
	Columns []string
	Reason  string
}

// Options configures an engine instance.
type Options struct {
	// EmitPointersForNull enables pointer types for nullable values.
	EmitPointersForNull bool

	// CustomTypes provides custom type mappings.
	CustomTypes []config.CustomTypeMapping
}

// New creates a new Engine for the specified database dialect.
// Returns an error if the dialect is not supported.
func New(dialect string, opts Options) (Engine, error) {
	return registry.New(dialect, opts)
}

// MustNew creates a new Engine or panics if the dialect is not supported.
// Useful for tests and initialization code.
func MustNew(dialect string, opts Options) Engine {
	engine, err := New(dialect, opts)
	if err != nil {
		panic(err)
	}
	return engine
}

// FromConfig creates an Engine from a configuration.
func FromConfig(cfg config.Database, opts Options) (Engine, error) {
	return New(string(cfg), opts)
}
