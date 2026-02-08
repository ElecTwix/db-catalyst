package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File permission constants for cache operations.
const (
	cacheDirPerm  = 0o750 // Directory permissions: rwxr-x---
	cacheFilePerm = 0o600 // File permissions: rw-------
)

// Minimum length for creating subdirectory structure in cache keys.
const minKeyLengthForSubdir = 4

// FileCache implements Cache using file system storage.
// It stores cache entries as JSON files in a directory structure.
type FileCache struct {
	baseDir string
}

// cacheEntry is the on-disk format for cached values.
type cacheEntry struct {
	Value     json.RawMessage `json:"value"`
	ExpiresAt time.Time       `json:"expires_at"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewFileCache creates a new file-based cache.
// The baseDir is where cache files will be stored.
func NewFileCache(baseDir string) (*FileCache, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(baseDir, cacheDirPerm); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	return &FileCache{
		baseDir: baseDir,
	}, nil
}

// Get retrieves a value from the cache.
func (f *FileCache) Get(_ context.Context, key string) (any, bool) {
	path := f.keyToPath(key)

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Clean up expired entry
		_ = os.Remove(path)
		return nil, false
	}

	// Unmarshal the actual value
	var value any
	if err := json.Unmarshal(entry.Value, &value); err != nil {
		return nil, false
	}

	return value, true
}

// Set stores a value in the cache with the given TTL.
func (f *FileCache) Set(_ context.Context, key string, value any, ttl time.Duration) {
	path := f.keyToPath(key)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, cacheDirPerm); err != nil {
		return
	}

	// Marshal the value
	valueData, err := json.Marshal(value)
	if err != nil {
		return
	}

	entry := cacheEntry{
		Value:     valueData,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	// Write atomically using temp file
	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, cacheFilePerm); err != nil {
		return
	}

	_ = os.Rename(tempFile, path)
}

// Delete removes a value from the cache.
func (f *FileCache) Delete(_ context.Context, key string) {
	path := f.keyToPath(key)
	_ = os.Remove(path)
}

// Clear removes all values from the cache.
func (f *FileCache) Clear(_ context.Context) {
	_ = os.RemoveAll(f.baseDir)
	_ = os.MkdirAll(f.baseDir, cacheDirPerm)
}

// keyToPath converts a cache key to a file path.
// It creates a directory structure to avoid too many files in one directory.
func (f *FileCache) keyToPath(key string) string {
	// Sanitize key for filesystem
	safeKey := sanitizeKey(key)

	// Create a 2-level directory structure using first 4 chars of key
	// This prevents having too many files in a single directory
	if len(safeKey) >= minKeyLengthForSubdir {
		subDir := filepath.Join(f.baseDir, safeKey[:2], safeKey[2:4])
		return filepath.Join(subDir, safeKey+".json")
	}

	return filepath.Join(f.baseDir, safeKey+".json")
}

// sanitizeKey makes a key safe for use as a filename.
func sanitizeKey(key string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(key)
}

// Stats returns statistics about the cache.
func (f *FileCache) Stats() (total int, expired int, size int64) {
	_ = filepath.Walk(f.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		total++
		size += info.Size()

		// Check if expired
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil
		}

		var entry cacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil
		}

		if time.Now().After(entry.ExpiresAt) {
			expired++
		}

		return nil
	})

	return total, expired, size
}

// Cleanup removes expired entries.
func (f *FileCache) Cleanup() {
	_ = filepath.Walk(f.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil
		}

		var entry cacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			_ = os.Remove(path)
			return nil
		}

		if time.Now().After(entry.ExpiresAt) {
			_ = os.Remove(path)
		}

		return nil
	})
}

// Ensure FileCache implements Cache interface.
var _ Cache = (*FileCache)(nil)
