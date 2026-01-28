package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache implements Cache using in-memory storage.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]Entry
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]Entry),
	}
}

// Get retrieves a value from the cache.
func (m *MemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.items[key]
	if !ok {
		return nil, false
	}

	if entry.IsExpired() {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in the cache with the given TTL.
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache.
func (m *MemoryCache) Delete(ctx context.Context, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
}

// Clear removes all values from the cache.
func (m *MemoryCache) Clear(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]Entry)
}

// Len returns the number of items in the cache (including expired).
func (m *MemoryCache) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.items)
}

// Cleanup removes expired entries.
func (m *MemoryCache) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, entry := range m.items {
		if now.After(entry.ExpiresAt) {
			delete(m.items, key)
		}
	}
}

// Ensure MemoryCache implements Cache interface
var _ Cache = (*MemoryCache)(nil)
