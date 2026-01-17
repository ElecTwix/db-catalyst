// Package cache provides caching utilities for the db-catalyst pipeline.
// It includes deterministic caching, schema caching, query caching, and result caching.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
)

// Cache provides deterministic caching for parsed content.
type Cache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

// New creates a new Cache instance.
func New() *Cache {
	return &Cache{
		items: make(map[string][]byte),
	}
}

// Get retrieves cached content by key.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.items[key]
	return val, ok
}

// Put stores content in the cache.
func (c *Cache) Put(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

// Hash computes a deterministic hash for content.
func Hash(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// SchemaCache caches parsed schema results.
type SchemaCache struct {
	cache *Cache
}

// NewSchemaCache creates a new SchemaCache.
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{cache: New()}
}

// Get retrieves cached schema AST if available.
func (c *SchemaCache) Get(filePath string, content []byte) ([]byte, bool) {
	key := schemaKey(filePath, content)
	return c.cache.Get(key)
}

// Put stores schema AST in the cache.
func (c *SchemaCache) Put(filePath string, content []byte, ast []byte) {
	key := schemaKey(filePath, content)
	c.cache.Put(key, ast)
}

func schemaKey(filePath string, content []byte) string {
	return "schema:" + filePath + ":" + Hash(content)
}

// QueryCache caches parsed query results.
type QueryCache struct {
	cache *Cache
}

// NewQueryCache creates a new QueryCache.
func NewQueryCache() *QueryCache {
	return &QueryCache{cache: New()}
}

// Get retrieves cached query AST if available.
func (c *QueryCache) Get(filePath string, content []byte) ([]byte, bool) {
	key := queryKey(filePath, content)
	return c.cache.Get(key)
}

// Put stores query AST in the cache.
func (c *QueryCache) Put(filePath string, content []byte, ast []byte) {
	key := queryKey(filePath, content)
	c.cache.Put(key, ast)
}

func queryKey(filePath string, content []byte) string {
	return "query:" + filePath + ":" + Hash(content)
}

// CachedReader wraps an io.Reader and tracks if content was served from cache.
type CachedReader struct {
	reader  io.Reader
	cached  bool
	content []byte
}

// NewCachedReader creates a new CachedReader.
func NewCachedReader(r io.Reader) (*CachedReader, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &CachedReader{
		reader:  nil,
		cached:  false,
		content: content,
	}, nil
}

// Content returns the reader content.
func (r *CachedReader) Content() []byte {
	return r.content
}

// Cached returns whether this content was from cache.
func (r *CachedReader) Cached() bool {
	return r.cached
}

// SetCached marks the content as from cache.
func (r *CachedReader) SetCached() {
	r.cached = true
}

// Cacheable is an interface for objects that can be cached.
type Cacheable interface {
	// CacheKey returns a unique key for this object.
	CacheKey() string
}

// ResultCache provides caching for pipeline results.
type ResultCache struct {
	mu    sync.RWMutex
	items map[string]ResultEntry
}

// ResultEntry represents a cached pipeline result.
type ResultEntry struct {
	Files   []byte
	Catalog []byte
}

// NewResultCache creates a new ResultCache.
func NewResultCache() *ResultCache {
	return &ResultCache{
		items: make(map[string]ResultEntry),
	}
}

// Get retrieves a cached result.
func (c *ResultCache) Get(configHash string) (ResultEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.items[configHash]
	return val, ok
}

// Put stores a result in the cache.
func (c *ResultCache) Put(configHash string, entry ResultEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[configHash] = entry
}
