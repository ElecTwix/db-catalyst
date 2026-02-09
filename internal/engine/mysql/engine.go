// Package mysql provides the MySQL database engine implementation.
package mysql

import (
	"time"

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

// Default connection pool settings for MySQL.
const (
	mysqlMaxOpenConns    = 25
	mysqlMaxIdleConns    = 5
	mysqlConnMaxLifetime = 1 * time.Hour
	mysqlConnMaxIdleTime = 30 * time.Minute
)

// ConnectionPool returns recommended connection pool settings for MySQL.
// MySQL benefits from moderate-sized connection pools.
func (e *Engine) ConnectionPool() engine.ConnectionPoolConfig {
	return engine.ConnectionPoolConfig{
		MaxOpenConns:    mysqlMaxOpenConns,
		MaxIdleConns:    mysqlMaxIdleConns,
		ConnMaxLifetime: mysqlConnMaxLifetime,
		ConnMaxIdleTime: mysqlConnMaxIdleTime,
	}
}

// IsolationLevels returns supported isolation levels for MySQL.
func (e *Engine) IsolationLevels() (supported []engine.IsolationLevel, defaultLevel engine.IsolationLevel) {
	return []engine.IsolationLevel{
		engine.IsolationLevelReadUncommitted,
		engine.IsolationLevelReadCommitted,
		engine.IsolationLevelRepeatableRead,
		engine.IsolationLevelSerializable,
	}, engine.IsolationLevelRepeatableRead
}

// QueryHints returns available query hints for MySQL.
// MySQL supports index hints and optimizer hints.
func (e *Engine) QueryHints() []engine.QueryHint {
	return []engine.QueryHint{
		// Index hints
		{
			Name:        "USE_INDEX",
			Description: "Suggest which index to use for a table",
			Syntax:      "SELECT ... FROM table USE INDEX (index_name)",
		},
		{
			Name:        "FORCE_INDEX",
			Description: "Force use of a specific index",
			Syntax:      "SELECT ... FROM table FORCE INDEX (index_name)",
		},
		{
			Name:        "IGNORE_INDEX",
			Description: "Ignore specific indexes",
			Syntax:      "SELECT ... FROM table IGNORE INDEX (index_name)",
		},
		// Optimizer hints (MySQL 5.6+)
		{
			Name:        "MAX_EXECUTION_TIME",
			Description: "Set maximum execution time in milliseconds",
			Syntax:      "SELECT /*+ MAX_EXECUTION_TIME(ms) */ ...",
		},
		{
			Name:        "NO_RANGE_OPTIMIZATION",
			Description: "Disable range optimization for a table",
			Syntax:      "SELECT /*+ NO_RANGE_OPTIMIZATION(table_name) */ ...",
		},
		{
			Name:        "ORDER_INDEX",
			Description: "Use index for ORDER BY",
			Syntax:      "SELECT /*+ ORDER_INDEX(table_name index_name) */ ...",
		},
		{
			Name:        "GROUP_INDEX",
			Description: "Use index for GROUP BY",
			Syntax:      "SELECT /*+ GROUP_INDEX(table_name index_name) */ ...",
		},
		{
			Name:        "HASH_JOIN",
			Description: "Use hash join for tables",
			Syntax:      "SELECT /*+ HASH_JOIN(table1, table2) */ ...",
		},
		{
			Name:        "NO_HASH_JOIN",
			Description: "Avoid hash join for tables",
			Syntax:      "SELECT /*+ NO_HASH_JOIN(table1, table2) */ ...",
		},
		{
			Name:        "MERGE",
			Description: "Merge derived tables/subqueries into outer query",
			Syntax:      "SELECT /*+ MERGE(dt) */ ...",
		},
		{
			Name:        "NO_MERGE",
			Description: "Prevent merging of derived tables/subqueries",
			Syntax:      "SELECT /*+ NO_MERGE(dt) */ ...",
		},
	}
}
