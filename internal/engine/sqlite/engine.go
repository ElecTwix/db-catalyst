// Package sqlite provides the SQLite database engine implementation.
package sqlite

import (
	"time"

	"github.com/electwix/db-catalyst/internal/engine"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
)

// Engine implements the engine.Engine interface for SQLite.
type Engine struct {
	opts       engine.Options
	typeMapper *typeMapper
	parser     schemaparser.SchemaParser
	sqlGen     *sqlGenerator
}

// New creates a new SQLite engine instance.
func New(opts engine.Options) (engine.Engine, error) {
	e := &Engine{
		opts: opts,
	}

	e.typeMapper = newTypeMapper(opts)

	var err error
	e.parser, err = schemaparser.NewSchemaParser("sqlite")
	if err != nil {
		return nil, err
	}

	e.sqlGen = newSQLGenerator()

	return e, nil
}

// Name returns the engine identifier.
func (e *Engine) Name() string {
	return "sqlite"
}

// TypeMapper returns the SQLite type mapper.
func (e *Engine) TypeMapper() engine.TypeMapper {
	return e.typeMapper
}

// SchemaParser returns the SQLite schema parser.
func (e *Engine) SchemaParser() schemaparser.SchemaParser {
	return e.parser
}

// SQLGenerator returns the SQLite SQL generator.
func (e *Engine) SQLGenerator() engine.SQLGenerator {
	return e.sqlGen
}

// DefaultDriver returns the default Go driver import path for SQLite.
func (e *Engine) DefaultDriver() string {
	return "modernc.org/sqlite"
}

// SupportsFeature reports whether SQLite supports a specific feature.
func (e *Engine) SupportsFeature(feature engine.Feature) bool {
	switch feature {
	case engine.FeatureTransactions,
		engine.FeatureForeignKeys,
		engine.FeatureCTEs,
		engine.FeatureUpsert,
		engine.FeatureReturning,
		engine.FeatureJSON,
		engine.FeatureAutoIncrement,
		engine.FeatureViews,
		engine.FeatureIndexes:
		return true
	case engine.FeatureWindowFunctions:
		return true // SQLite 3.25.0+
	case engine.FeatureArrays,
		engine.FeatureFullTextSearch:
		return false
	case engine.FeaturePreparedStatements:
		return true
	default:
		return false
	}
}

// Default connection pool settings for SQLite.
const (
	sqliteMaxOpenConns    = 5
	sqliteMaxIdleConns    = 2
	sqliteConnMaxLifetime = 1 * time.Hour
	sqliteConnMaxIdleTime = 30 * time.Minute
)

// ConnectionPool returns recommended connection pool settings for SQLite.
// SQLite is file-based and doesn't benefit from large connection pools.
func (e *Engine) ConnectionPool() engine.ConnectionPoolConfig {
	return engine.ConnectionPoolConfig{
		MaxOpenConns:    sqliteMaxOpenConns,
		MaxIdleConns:    sqliteMaxIdleConns,
		ConnMaxLifetime: sqliteConnMaxLifetime,
		ConnMaxIdleTime: sqliteConnMaxIdleTime,
	}
}

// IsolationLevels returns supported isolation levels for SQLite.
// SQLite only supports SERIALIZABLE isolation level internally.
func (e *Engine) IsolationLevels() (supported []engine.IsolationLevel, defaultLevel engine.IsolationLevel) {
	// SQLite only supports SERIALIZABLE isolation level
	return []engine.IsolationLevel{
		engine.IsolationLevelSerializable,
	}, engine.IsolationLevelSerializable
}

// QueryHints returns available query hints for SQLite.
// SQLite does not support query hints.
func (e *Engine) QueryHints() []engine.QueryHint {
	// SQLite does not support query hints
	return nil
}
