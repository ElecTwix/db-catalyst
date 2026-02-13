// Package cache provides a runtime cache interface for query result caching.
package cache

import (
	"context"
	"time"
)

// Cache is the interface for query result caching.
type Cache interface {
	// Get retrieves a value from the cache.
	Get(ctx context.Context, key string) (any, bool)
	// Set stores a value in the cache with the given TTL.
	Set(ctx context.Context, key string, value any, ttl time.Duration)
	// Delete removes a value from the cache.
	Delete(ctx context.Context, key string)
	// Invalidate removes all values matching the given pattern.
	Invalidate(ctx context.Context, pattern string)
}

// Entry represents a cached entry.
type Entry struct {
	Value      any
	ExpiresAt  time.Time
	Invalidate []string // patterns to invalidate on write
}
