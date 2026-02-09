package mysql

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

	// MySQL should have medium-sized connection pool
	if pool.MaxOpenConns != mysqlMaxOpenConns {
		t.Errorf("MaxOpenConns = %d, want %d", pool.MaxOpenConns, mysqlMaxOpenConns)
	}
	if pool.MaxIdleConns != mysqlMaxIdleConns {
		t.Errorf("MaxIdleConns = %d, want %d", pool.MaxIdleConns, mysqlMaxIdleConns)
	}
	if pool.ConnMaxLifetime != mysqlConnMaxLifetime {
		t.Errorf("ConnMaxLifetime = %v, want %v", pool.ConnMaxLifetime, mysqlConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime != mysqlConnMaxIdleTime {
		t.Errorf("ConnMaxIdleTime = %v, want %v", pool.ConnMaxIdleTime, mysqlConnMaxIdleTime)
	}
}

func TestEngine_IsolationLevels(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	supported, defaultLevel := e.IsolationLevels()

	// MySQL supports ReadUncommitted, ReadCommitted, RepeatableRead, Serializable
	if len(supported) != 4 {
		t.Errorf("len(supported) = %d, want 4", len(supported))
	}

	// Check all expected levels are present
	expectedLevels := map[engine.IsolationLevel]bool{
		engine.IsolationLevelReadUncommitted: false,
		engine.IsolationLevelReadCommitted:   false,
		engine.IsolationLevelRepeatableRead:  false,
		engine.IsolationLevelSerializable:    false,
	}
	for _, level := range supported {
		expectedLevels[level] = true
	}
	for level, found := range expectedLevels {
		if !found {
			t.Errorf("Expected isolation level %v not found", level)
		}
	}

	if defaultLevel != engine.IsolationLevelRepeatableRead {
		t.Errorf("defaultLevel = %v, want RepeatableRead", defaultLevel)
	}
}

func TestEngine_QueryHints(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hints := e.QueryHints()

	// MySQL should have many query hints
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

const mysqlEngineName = "mysql"

func TestEngine_Name(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := e.Name(); got != mysqlEngineName {
		t.Errorf("Name() = %q, want %q", got, mysqlEngineName)
	}
}

func TestEngine_DefaultDriver(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := "github.com/go-sql-driver/mysql"
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
	if mysqlMaxOpenConns <= 0 {
		t.Error("mysqlMaxOpenConns should be positive")
	}
	if mysqlMaxIdleConns < 0 {
		t.Error("mysqlMaxIdleConns should not be negative")
	}
	if mysqlMaxIdleConns > mysqlMaxOpenConns {
		t.Error("mysqlMaxIdleConns should not exceed mysqlMaxOpenConns")
	}
	if mysqlConnMaxLifetime <= 0 {
		t.Error("mysqlConnMaxLifetime should be positive")
	}
	if mysqlConnMaxIdleTime <= 0 {
		t.Error("mysqlConnMaxIdleTime should be positive")
	}
}

func TestEngine_ConnectionPool_IdleTimeLessThanLifetime(t *testing.T) {
	// Verify idle time is less than or equal to max lifetime
	if mysqlConnMaxIdleTime > mysqlConnMaxLifetime {
		t.Error("mysqlConnMaxIdleTime should not exceed mysqlConnMaxLifetime")
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

	// Check for expected MySQL hints
	expectedHints := []string{"USE_INDEX", "FORCE_INDEX", "IGNORE_INDEX", "MAX_EXECUTION_TIME"}
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

func TestEngine_QueryHints_HasIndexHints(t *testing.T) {
	e, err := New(engine.Options{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hints := e.QueryHints()

	// MySQL has specific index hints that should be present
	hasIndexHints := false
	for _, h := range hints {
		if h.Name == "USE_INDEX" || h.Name == "FORCE_INDEX" || h.Name == "IGNORE_INDEX" {
			hasIndexHints = true
			// Verify these use the old-style hint syntax (not comment-based)
			if h.Syntax == "" {
				t.Errorf("Index hint %q has empty syntax", h.Name)
			}
		}
	}

	if !hasIndexHints {
		t.Error("MySQL should have index hints (USE_INDEX, FORCE_INDEX, IGNORE_INDEX)")
	}
}
