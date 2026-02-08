// Package mysql provides the MySQL database engine implementation.
package mysql

import (
	"github.com/electwix/db-catalyst/internal/engine"
	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/parser/mysql"
)

// Engine implements the engine.Engine interface for MySQL.
type Engine struct {
	opts   engine.Options
	parser diagnostic.SchemaParser
}

// New creates a new MySQL engine instance.
func New(opts engine.Options) (engine.Engine, error) {
	return &Engine{
		opts:   opts,
		parser: mysql.New(),
	}, nil
}

// Name returns the engine identifier.
func (e *Engine) Name() string {
	return "mysql"
}

// TypeMapper returns the MySQL type mapper.
func (e *Engine) TypeMapper() engine.TypeMapper {
	return newTypeMapper(e.opts)
}

// SchemaParser returns the native MySQL schema parser.
func (e *Engine) SchemaParser() diagnostic.SchemaParser {
	return e.parser
}

// SQLGenerator returns the MySQL SQL generator.
func (e *Engine) SQLGenerator() engine.SQLGenerator {
	return &sqlGenerator{}
}

// DefaultDriver returns the default Go driver import path for MySQL.
func (e *Engine) DefaultDriver() string {
	return "github.com/go-sql-driver/mysql"
}

// SupportsFeature reports whether MySQL supports a specific feature.
func (e *Engine) SupportsFeature(feature engine.Feature) bool {
	switch feature {
	case engine.FeatureTransactions,
		engine.FeatureForeignKeys,
		engine.FeatureWindowFunctions, // MySQL 8.0+
		engine.FeatureCTEs,            // MySQL 8.0+
		engine.FeatureUpsert,          // INSERT ... ON DUPLICATE KEY UPDATE
		engine.FeatureReturning,       // MySQL 8.0.19+
		engine.FeatureJSON,            // MySQL 5.7+
		engine.FeatureFullTextSearch,
		engine.FeaturePreparedStatements,
		engine.FeatureAutoIncrement,
		engine.FeatureViews,
		engine.FeatureIndexes:
		return true
	case engine.FeatureArrays:
		return false // MySQL doesn't have native array types
	default:
		return false
	}
}
