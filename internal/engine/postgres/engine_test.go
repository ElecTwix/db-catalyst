package postgres

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/engine"
)

func TestEngine_ConnectionPool(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	pool := e.ConnectionPool()

	// PostgreSQL should have medium-sized connection pool
	if pool.MaxOpenConns != postgresMaxOpenConns {
		t.Errorf("MaxOpenConns = %d, want %d", pool.MaxOpenConns, postgresMaxOpenConns)
	}
	if pool.MaxIdleConns != postgresMaxIdleConns {
		t.Errorf("MaxIdleConns = %d, want %d", pool.MaxIdleConns, postgresMaxIdleConns)
	}
	if pool.ConnMaxLifetime != postgresConnMaxLifetime {
		t.Errorf("ConnMaxLifetime = %v, want %v", pool.ConnMaxLifetime, postgresConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime != postgresConnMaxIdleTime {
		t.Errorf("ConnMaxIdleTime = %v, want %v", pool.ConnMaxIdleTime, postgresConnMaxIdleTime)
	}
}

func TestEngine_IsolationLevels(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	supported, defaultLevel := e.IsolationLevels()

	// PostgreSQL supports ReadCommitted, RepeatableRead, Serializable
	if len(supported) != 3 {
		t.Errorf("len(supported) = %d, want 3", len(supported))
	}

	// Check all expected levels are present
	expectedLevels := map[engine.IsolationLevel]bool{
		engine.IsolationLevelReadCommitted:  false,
		engine.IsolationLevelRepeatableRead: false,
		engine.IsolationLevelSerializable:   false,
	}
	for _, level := range supported {
		expectedLevels[level] = true
	}
	for level, found := range expectedLevels {
		if !found {
			t.Errorf("Expected isolation level %v not found", level)
		}
	}

	if defaultLevel != engine.IsolationLevelReadCommitted {
		t.Errorf("defaultLevel = %v, want ReadCommitted", defaultLevel)
	}
}

func TestEngine_QueryHints(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hints := e.QueryHints()

	// PostgreSQL should have several query hints (via pg_hint_plan extension)
	if len(hints) == 0 {
		t.Error("QueryHints() should return non-empty slice")
	}

	// Check that hints have valid structure
	for _, hint := range hints {
		if hint.Name == "" {
			t.Error("Query hint has empty Name")
		}
		if hint.Description == "" {
			t.Error("Query hint has empty Description")
		}
		if hint.Syntax == "" {
			t.Error("Query hint has empty Syntax")
		}
	}
}

const postgresEngineName = "postgresql"

func TestEngine_Name(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := e.Name(); got != postgresEngineName {
		t.Errorf("Name() = %q, want %q", got, postgresEngineName)
	}
}

func TestEngine_DefaultDriver(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := "github.com/jackc/pgx/v5"
	if got := e.DefaultDriver(); got != want {
		t.Errorf("DefaultDriver() = %q, want %q", got, want)
	}
}

func TestEngine_SupportsFeature(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		feature engine.Feature
		want    bool
	}{
		{engine.FeatureTransactions, true},
		{engine.FeatureForeignKeys, true},
		{engine.FeatureWindowFunctions, true},
		{engine.FeatureCTEs, true},
		{engine.FeatureUpsert, true},
		{engine.FeatureReturning, true},
		{engine.FeatureJSON, true},
		{engine.FeatureArrays, true},
		{engine.FeatureFullTextSearch, true},
		{engine.FeaturePreparedStatements, true},
		{engine.FeatureAutoIncrement, true},
		{engine.FeatureViews, true},
		{engine.FeatureIndexes, true},
	}

	for _, tt := range tests {
		t.Run(tt.feature.String(), func(t *testing.T) {
			if got := e.SupportsFeature(tt.feature); got != tt.want {
				t.Errorf("SupportsFeature(%q) = %v, want %v", tt.feature, got, tt.want)
			}
		})
	}
}

func TestEngine_ConnectionPoolConstants(t *testing.T) {
	// Verify constants are reasonable
	if postgresMaxOpenConns <= 0 {
		t.Error("postgresMaxOpenConns should be positive")
	}
	if postgresMaxIdleConns < 0 {
		t.Error("postgresMaxIdleConns should not be negative")
	}
	if postgresMaxIdleConns > postgresMaxOpenConns {
		t.Error("postgresMaxIdleConns should not exceed postgresMaxOpenConns")
	}
	if postgresConnMaxLifetime <= 0 {
		t.Error("postgresConnMaxLifetime should be positive")
	}
	if postgresConnMaxIdleTime <= 0 {
		t.Error("postgresConnMaxIdleTime should be positive")
	}
}

func TestEngine_ConnectionPool_IdleTimeLessThanLifetime(t *testing.T) {
	// Verify idle time is less than or equal to max lifetime
	if postgresConnMaxIdleTime > postgresConnMaxLifetime {
		t.Error("postgresConnMaxIdleTime should not exceed postgresConnMaxLifetime")
	}
}

func TestEngine_TypeMapper_NotNil(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if e.TypeMapper() == nil {
		t.Error("TypeMapper() should not return nil")
	}
}

func TestEngine_SchemaParser_NotNil(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if e.SchemaParser() == nil {
		t.Error("SchemaParser() should not return nil")
	}
}

func TestEngine_SQLGenerator_NotNil(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if e.SQLGenerator() == nil {
		t.Error("SQLGenerator() should not return nil")
	}
}

func TestEngine_QueryHints_ContainsExpected(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hints := e.QueryHints()

	// Check for expected PostgreSQL hints
	expectedHints := []string{"SeqScan", "IndexScan", "BitmapScan", "NestLoop", "HashJoin", "MergeJoin", "Rows", "Set"}
	hintNames := make(map[string]bool)
	for _, h := range hints {
		hintNames[h.Name] = true
	}

	for _, expected := range expectedHints {
		if !hintNames[expected] {
			t.Errorf("Expected hint %q not found", expected)
		}
	}
}
