// Package postgres provides the PostgreSQL database engine implementation.
package postgres

import (
	"time"

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

// Default connection pool settings for PostgreSQL.
const (
	postgresMaxOpenConns    = 25
	postgresMaxIdleConns    = 5
	postgresConnMaxLifetime = 1 * time.Hour
	postgresConnMaxIdleTime = 30 * time.Minute
)

// ConnectionPool returns recommended connection pool settings for PostgreSQL.
// PostgreSQL benefits from moderate-sized connection pools.
func (e *Engine) ConnectionPool() engine.ConnectionPoolConfig {
	return engine.ConnectionPoolConfig{
		MaxOpenConns:    postgresMaxOpenConns,
		MaxIdleConns:    postgresMaxIdleConns,
		ConnMaxLifetime: postgresConnMaxLifetime,
		ConnMaxIdleTime: postgresConnMaxIdleTime,
	}
}

// IsolationLevels returns supported isolation levels for PostgreSQL.
func (e *Engine) IsolationLevels() (supported []engine.IsolationLevel, defaultLevel engine.IsolationLevel) {
	return []engine.IsolationLevel{
		engine.IsolationLevelReadCommitted,
		engine.IsolationLevelRepeatableRead,
		engine.IsolationLevelSerializable,
	}, engine.IsolationLevelReadCommitted
}

// QueryHints returns available query hints for PostgreSQL.
// PostgreSQL has limited query hint support via optimizer hints extension.
func (e *Engine) QueryHints() []engine.QueryHint {
	// PostgreSQL doesn't support standard query hints natively,
	// but pg_hint_plan extension provides some support
	return []engine.QueryHint{
		{
			Name:        "SeqScan",
			Description: "Force sequential scan on a table",
			Syntax:      "/*+ SeqScan(table_name) */",
		},
		{
			Name:        "IndexScan",
			Description: "Force index scan on a table",
			Syntax:      "/*+ IndexScan(table_name index_name) */",
		},
		{
			Name:        "BitmapScan",
			Description: "Force bitmap scan on a table",
			Syntax:      "/*+ BitmapScan(table_name index_name) */",
		},
		{
			Name:        "NestLoop",
			Description: "Force nested loop join",
			Syntax:      "/*+ NestLoop(table1 table2) */",
		},
		{
			Name:        "HashJoin",
			Description: "Force hash join",
			Syntax:      "/*+ HashJoin(table1 table2) */",
		},
		{
			Name:        "MergeJoin",
			Description: "Force merge join",
			Syntax:      "/*+ MergeJoin(table1 table2) */",
		},
		{
			Name:        "Rows",
			Description: "Provide row count estimate for better planning",
			Syntax:      "/*+ Rows(table1 table2 #rows) */",
		},
		{
			Name:        "Set",
			Description: "Set planner configuration parameter",
			Syntax:      "/*+ Set(param value) */",
		},
	}
}
