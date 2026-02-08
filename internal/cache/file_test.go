package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCacheBasic(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Test Set and Get
	key := "test-key"
	value := map[string]string{"foo": "bar"}

	fc.Set(ctx, key, value, time.Hour)

	// Retrieve from cache
	got, ok := fc.Get(ctx, key)
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}

	// Type assertion for the retrieved value
	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}

	if gotMap["foo"] != "bar" {
		t.Errorf("expected foo=bar, got foo=%v", gotMap["foo"])
	}
}

func TestFileCacheMiss(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Try to get a non-existent key
	_, ok := fc.Get(ctx, "non-existent")
	if ok {
		t.Error("expected cache miss, got hit")
	}
}

func TestFileCacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Set with very short TTL
	key := "expiring-key"
	value := "test-value"

	fc.Set(ctx, key, value, time.Millisecond)

	// Should be available immediately
	_, ok := fc.Get(ctx, key)
	if !ok {
		t.Error("expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired now
	_, ok = fc.Get(ctx, key)
	if ok {
		t.Error("expected cache miss after expiration")
	}
}

func TestFileCacheDelete(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	key := "delete-key"
	value := "test-value"

	fc.Set(ctx, key, value, time.Hour)

	// Verify it exists
	_, ok := fc.Get(ctx, key)
	if !ok {
		t.Fatal("expected cache hit before delete")
	}

	// Delete it
	fc.Delete(ctx, key)

	// Verify it's gone
	_, ok = fc.Get(ctx, key)
	if ok {
		t.Error("expected cache miss after delete")
	}
}

func TestFileCacheClear(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Add multiple entries
	for i := range 5 {
		fc.Set(ctx, "key-"+string(rune('a'+i)), "value", time.Hour)
	}

	// Clear the cache
	fc.Clear(ctx)

	// Verify all entries are gone
	for i := range 5 {
		_, ok := fc.Get(ctx, "key-"+string(rune('a'+i)))
		if ok {
			t.Errorf("expected cache miss for key-%c after clear", rune('a'+i))
		}
	}
}

func TestFileCacheStats(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Empty cache - only checking total count
	total, _, _ := fc.Stats()
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}

	// Add some entries
	fc.Set(ctx, "key1", "value1", time.Hour)
	fc.Set(ctx, "key2", map[string]int{"a": 1}, time.Hour)

	total, expired, size := fc.Stats()
	if total != 2 {
		t.Errorf("expected 2 total, got %d", total)
	}
	if expired != 0 {
		t.Errorf("expected 0 expired, got %d", expired)
	}
	if size <= 0 {
		t.Error("expected positive size")
	}

	// Add an expired entry
	fc.Set(ctx, "expired", "value", time.Nanosecond)
	time.Sleep(10 * time.Millisecond)

	total, expired, _ = fc.Stats()
	if total != 3 {
		t.Errorf("expected 3 total, got %d", total)
	}
	if expired != 1 {
		t.Errorf("expected 1 expired, got %d", expired)
	}
}

func TestFileCacheCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Add valid and expired entries
	fc.Set(ctx, "valid", "value", time.Hour)
	fc.Set(ctx, "expired", "value", time.Nanosecond)

	time.Sleep(10 * time.Millisecond)

	// Cleanup
	fc.Cleanup()

	// Verify valid entry still exists
	_, ok := fc.Get(ctx, "valid")
	if !ok {
		t.Error("expected valid entry to remain after cleanup")
	}

	// Expired entry should be removed (or at least not accessible)
	_, ok = fc.Get(ctx, "expired")
	if ok {
		t.Error("expected expired entry to be removed after cleanup")
	}
}

func TestFileCacheKeySanitization(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Test keys with problematic characters
	problematicKeys := []string{
		"key/with/slashes",
		"key\\\\with\\\\backslashes",
		"key:with:colons",
		"key*with*asterisks",
		"key?with?question",
		"key\"with\"quotes",
		"key<with>angle",
		"key|with|pipe",
	}

	for _, key := range problematicKeys {
		value := "value-for-" + key
		fc.Set(ctx, key, value, time.Hour)

		got, ok := fc.Get(ctx, key)
		if !ok {
			t.Errorf("expected cache hit for key %q", key)
			continue
		}

		gotStr, ok := got.(string)
		if !ok || gotStr != value {
			t.Errorf("expected %q for key %q, got %v", value, key, got)
		}
	}
}

func TestFileCacheDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	fc, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache failed: %v", err)
	}

	ctx := context.Background()

	// Add an entry
	fc.Set(ctx, "test-key-1234", "value", time.Hour)

	// Check that the directory structure was created
	// The key "test-key-1234" should create subdirectories based on first 4 chars
	expectedSubDir := filepath.Join(cacheDir, "te", "st")
	if _, err := os.Stat(expectedSubDir); os.IsNotExist(err) {
		t.Errorf("expected subdirectory %s to exist", expectedSubDir)
	}
}

func TestNewFileCacheCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "nonexistent", "cache")

	// Directory doesn't exist yet
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatal("cache directory should not exist yet")
	}

	_, err := NewFileCache(cacheDir)
	if err != nil {
		t.Fatalf("NewFileCache should create directory: %v", err)
	}

	// Directory should now exist
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("cache directory should have been created")
	}
}
