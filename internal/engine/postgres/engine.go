// Package postgres provides the PostgreSQL database engine implementation.
package postgres

import (
	"github.com/electwix/db-catalyst/internal/engine"
	"github.com/electwix/db-catalyst/internal/schema/diagnostic"
	"github.com/electwix/db-catalyst/internal/schema/parser/postgres"
)

// Engine implements the engine.Engine interface for PostgreSQL.
type Engine struct {
	opts   engine.Options
	parser diagnostic.SchemaParser
}

// New creates a new PostgreSQL engine instance.
func New(opts engine.Options) (engine.Engine, error) {
	return &Engine{
		opts:   opts,
		parser: postgres.New(),
	}, nil
}

// Name returns the engine identifier.
func (e *Engine) Name() string {
	return "postgresql"
}

// TypeMapper returns the PostgreSQL type mapper.
func (e *Engine) TypeMapper() engine.TypeMapper {
	return &typeMapper{opts: e.opts}
}

// SchemaParser returns the native PostgreSQL schema parser.
func (e *Engine) SchemaParser() diagnostic.SchemaParser {
	return e.parser
}

// SQLGenerator returns the PostgreSQL SQL generator.
func (e *Engine) SQLGenerator() engine.SQLGenerator {
	return &sqlGenerator{}
}

// DefaultDriver returns the default Go driver import path for PostgreSQL.
func (e *Engine) DefaultDriver() string {
	return "github.com/jackc/pgx/v5"
}

// SupportsFeature reports whether PostgreSQL supports a specific feature.
func (e *Engine) SupportsFeature(feature engine.Feature) bool {
	// PostgreSQL supports almost everything
	switch feature {
	case engine.FeatureTransactions,
		engine.FeatureForeignKeys,
		engine.FeatureWindowFunctions,
		engine.FeatureCTEs,
		engine.FeatureUpsert,
		engine.FeatureReturning,
		engine.FeatureJSON,
		engine.FeatureArrays,
		engine.FeatureFullTextSearch,
		engine.FeaturePreparedStatements,
		engine.FeatureAutoIncrement,
		engine.FeatureViews,
		engine.FeatureIndexes:
		return true
	default:
		return false
	}
}
