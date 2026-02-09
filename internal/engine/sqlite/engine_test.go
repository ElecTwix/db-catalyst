package sqlite

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

	// SQLite should have small connection pool
	if pool.MaxOpenConns != sqliteMaxOpenConns {
		t.Errorf("MaxOpenConns = %d, want %d", pool.MaxOpenConns, sqliteMaxOpenConns)
	}
	if pool.MaxIdleConns != sqliteMaxIdleConns {
		t.Errorf("MaxIdleConns = %d, want %d", pool.MaxIdleConns, sqliteMaxIdleConns)
	}
	if pool.ConnMaxLifetime != sqliteConnMaxLifetime {
		t.Errorf("ConnMaxLifetime = %v, want %v", pool.ConnMaxLifetime, sqliteConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime != sqliteConnMaxIdleTime {
		t.Errorf("ConnMaxIdleTime = %v, want %v", pool.ConnMaxIdleTime, sqliteConnMaxIdleTime)
	}
}

func TestEngine_IsolationLevels(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	supported, defaultLevel := e.IsolationLevels()

	// SQLite only supports SERIALIZABLE
	if len(supported) != 1 {
		t.Errorf("len(supported) = %d, want 1", len(supported))
	}

	if supported[0] != engine.IsolationLevelSerializable {
		t.Errorf("supported[0] = %v, want Serializable", supported[0])
	}

	if defaultLevel != engine.IsolationLevelSerializable {
		t.Errorf("defaultLevel = %v, want Serializable", defaultLevel)
	}
}

func TestEngine_QueryHints(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hints := e.QueryHints()

	// SQLite does not support query hints
	if hints != nil {
		t.Errorf("QueryHints() = %v, want nil", hints)
	}
}

const sqliteEngineName = "sqlite"

func TestEngine_Name(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := e.Name(); got != sqliteEngineName {
		t.Errorf("Name() = %q, want %q", got, sqliteEngineName)
	}
}

func TestEngine_DefaultDriver(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := "modernc.org/sqlite"
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
		{engine.FeatureArrays, false},
		{engine.FeatureFullTextSearch, false},
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
	if sqliteMaxOpenConns <= 0 {
		t.Error("sqliteMaxOpenConns should be positive")
	}
	if sqliteMaxIdleConns < 0 {
		t.Error("sqliteMaxIdleConns should not be negative")
	}
	if sqliteMaxIdleConns > sqliteMaxOpenConns {
		t.Error("sqliteMaxIdleConns should not exceed sqliteMaxOpenConns")
	}
	if sqliteConnMaxLifetime <= 0 {
		t.Error("sqliteConnMaxLifetime should be positive")
	}
	if sqliteConnMaxIdleTime <= 0 {
		t.Error("sqliteConnMaxIdleTime should be positive")
	}
}

func TestEngine_ConnectionPool_IdleTimeLessThanLifetime(t *testing.T) {
	// Verify idle time is less than or equal to max lifetime
	if sqliteConnMaxIdleTime > sqliteConnMaxLifetime {
		t.Error("sqliteConnMaxIdleTime should not exceed sqliteConnMaxLifetime")
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
