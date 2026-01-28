// Package cache provides caching infrastructure for db-catalyst.
//
// The cache package defines a generic Cache interface and provides
// implementations for in-memory caching. It's used to speed up
// incremental builds by caching parsed schemas and query analyses.
//
// Usage:
//
//	c := cache.NewMemoryCache()
//	c.Set(ctx, "key", value, time.Hour)
//	if val, ok := c.Get(ctx, "key"); ok {
//	    // use cached value
//	}
package cache
