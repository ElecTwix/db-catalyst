// Package cache provides caching for parsed schemas and query ASTs.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Cache provides a generic caching interface.
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration)
	Delete(ctx context.Context, key string)
	Clear(ctx context.Context)
}

// Keyable is implemented by types that can provide a cache key.
type Keyable interface {
	CacheKey() string
}

// ComputeKey generates a cache key from content using SHA-256.
func ComputeKey(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:16]) // use first 128 bits
}

// ComputeKeyWithPrefix generates a cache key with a prefix.
func ComputeKeyWithPrefix(prefix string, content []byte) string {
	return fmt.Sprintf("%s:%s", prefix, ComputeKey(content))
}

// Entry represents a cached entry with expiration.
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired returns true if the entry has expired.
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}
