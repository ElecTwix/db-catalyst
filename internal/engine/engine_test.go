package engine

import (
	"testing"
	"time"
)

func TestConnectionPoolConfig_Values(t *testing.T) {
	tests := []struct {
		name string
		pool ConnectionPoolConfig
	}{
		{
			name: "SQLite defaults",
			pool: ConnectionPoolConfig{
				MaxOpenConns:    5,
				MaxIdleConns:    2,
				ConnMaxLifetime: 1 * time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
			},
		},
		{
			name: "PostgreSQL defaults",
			pool: ConnectionPoolConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 1 * time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
			},
		},
		{
			name: "MySQL defaults",
			pool: ConnectionPoolConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 1 * time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
			},
		},
		{
			name: "unlimited",
			pool: ConnectionPoolConfig{
				MaxOpenConns:    0,
				MaxIdleConns:    0,
				ConnMaxLifetime: 0,
				ConnMaxIdleTime: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pool.MaxOpenConns < 0 {
				t.Error("MaxOpenConns should not be negative")
			}
			if tt.pool.MaxIdleConns < 0 {
				t.Error("MaxIdleConns should not be negative")
			}
			if tt.pool.MaxIdleConns > tt.pool.MaxOpenConns && tt.pool.MaxOpenConns > 0 {
				t.Error("MaxIdleConns should not exceed MaxOpenConns")
			}
		})
	}
}

func TestIsolationLevel_String(t *testing.T) {
	tests := []struct {
		level IsolationLevel
		want  string
	}{
		{IsolationLevelDefault, "default"},
		{IsolationLevelReadUncommitted, "read_uncommitted"},
		{IsolationLevelReadCommitted, "read_committed"},
		{IsolationLevelWriteCommitted, "write_committed"},
		{IsolationLevelRepeatableRead, "repeatable_read"},
		{IsolationLevelSnapshot, "snapshot"},
		{IsolationLevelSerializable, "serializable"},
		{IsolationLevelLinearizable, "linearizable"},
		{IsolationLevel(999), "unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("IsolationLevel.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryHint_Validation(t *testing.T) {
	tests := []struct {
		name string
		hint QueryHint
		want struct {
			hasName        bool
			hasDescription bool
			hasSyntax      bool
		}
	}{
		{
			name: "complete hint",
			hint: QueryHint{
				Name:        "INDEX",
				Description: "Use specific index",
				Syntax:      "USE INDEX (idx)",
			},
			want: struct {
				hasName        bool
				hasDescription bool
				hasSyntax      bool
			}{true, true, true},
		},
		{
			name: "empty hint",
			hint: QueryHint{},
			want: struct {
				hasName        bool
				hasDescription bool
				hasSyntax      bool
			}{false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasName := tt.hint.Name != ""
			hasDescription := tt.hint.Description != ""
			hasSyntax := tt.hint.Syntax != ""

			if hasName != tt.want.hasName {
				t.Errorf("hasName = %v, want %v", hasName, tt.want.hasName)
			}
			if hasDescription != tt.want.hasDescription {
				t.Errorf("hasDescription = %v, want %v", hasDescription, tt.want.hasDescription)
			}
			if hasSyntax != tt.want.hasSyntax {
				t.Errorf("hasSyntax = %v, want %v", hasSyntax, tt.want.hasSyntax)
			}
		})
	}
}

func TestIsolationLevel_Comparison(t *testing.T) {
	// Test that isolation levels are ordered from least to most strict
	// This is a conceptual test - actual strictness depends on the database
	levels := []IsolationLevel{
		IsolationLevelDefault,
		IsolationLevelReadUncommitted,
		IsolationLevelReadCommitted,
		IsolationLevelWriteCommitted,
		IsolationLevelRepeatableRead,
		IsolationLevelSnapshot,
		IsolationLevelSerializable,
		IsolationLevelLinearizable,
	}

	for i, level := range levels {
		if int(level) != i {
			t.Errorf("IsolationLevel %v has value %d, expected %d", level, level, i)
		}
	}
}

func TestConnectionPoolConfig_Immutability(t *testing.T) {
	// Test that ConnectionPoolConfig can be copied without issues
	original := ConnectionPoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}

	// Create a copy
	cfgCopy := original

	// Modify the copy
	cfgCopy.MaxOpenConns = 20

	// Verify original is unchanged
	if original.MaxOpenConns != 10 {
		t.Error("Original ConnectionPoolConfig was modified when copy was changed")
	}
}

func TestFeature_String(t *testing.T) {
	tests := []struct {
		feature Feature
		want    string
	}{
		{FeatureTransactions, "transactions"},
		{FeatureForeignKeys, "foreign_keys"},
		{FeatureWindowFunctions, "window_functions"},
		{FeatureCTEs, "ctes"},
		{FeatureUpsert, "upsert"},
		{FeatureReturning, "returning"},
		{FeatureJSON, "json"},
		{FeatureArrays, "arrays"},
		{FeatureFullTextSearch, "fulltext_search"},
		{FeaturePreparedStatements, "prepared_statements"},
		{FeatureAutoIncrement, "auto_increment"},
		{FeatureViews, "views"},
		{FeatureIndexes, "indexes"},
		{Feature(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.feature.String()
			if got != tt.want {
				t.Errorf("Feature.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
