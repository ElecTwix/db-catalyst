package engine

// Feature represents a database capability that may vary between engines.
type Feature int

const (
	// FeatureTransactions indicates support for ACID transactions.
	FeatureTransactions Feature = iota

	// FeatureForeignKeys indicates support for foreign key constraints.
	FeatureForeignKeys

	// FeatureWindowFunctions indicates support for window functions (ROW_NUMBER, etc).
	FeatureWindowFunctions

	// FeatureCTEs indicates support for Common Table Expressions (WITH clauses).
	FeatureCTEs

	// FeatureUpsert indicates support for upsert operations (INSERT ON CONFLICT/ON DUPLICATE KEY).
	FeatureUpsert

	// FeatureReturning indicates support for RETURNING clause.
	FeatureReturning

	// FeatureJSON indicates support for JSON/JSONB data types and operations.
	FeatureJSON

	// FeatureArrays indicates support for array data types.
	FeatureArrays

	// FeatureFullTextSearch indicates support for full-text search.
	FeatureFullTextSearch

	// FeaturePreparedStatements indicates support for prepared statements.
	FeaturePreparedStatements

	// FeatureAutoIncrement indicates support for auto-increment/serial columns.
	FeatureAutoIncrement

	// FeatureViews indicates support for CREATE VIEW statements.
	FeatureViews

	// FeatureIndexes indicates support for CREATE INDEX statements.
	FeatureIndexes
)

// featureNames maps features to human-readable names.
var featureNames = map[Feature]string{
	FeatureTransactions:       "transactions",
	FeatureForeignKeys:        "foreign_keys",
	FeatureWindowFunctions:    "window_functions",
	FeatureCTEs:               "ctes",
	FeatureUpsert:             "upsert",
	FeatureReturning:          "returning",
	FeatureJSON:               "json",
	FeatureArrays:             "arrays",
	FeatureFullTextSearch:     "fulltext_search",
	FeaturePreparedStatements: "prepared_statements",
	FeatureAutoIncrement:      "auto_increment",
	FeatureViews:              "views",
	FeatureIndexes:            "indexes",
}

// String returns the human-readable name of a feature.
func (f Feature) String() string {
	if name, ok := featureNames[f]; ok {
		return name
	}
	return "unknown"
}
