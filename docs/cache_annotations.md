# SQL Query Result Caching

db-catalyst now supports query result caching through SQL comment annotations. This feature allows you to cache the results of read queries automatically, reducing database load and improving response times.

## Features

- **Declarative caching**: Add cache annotations directly in your SQL files
- **Automatic cache key generation**: Build cache keys from query parameters
- **TTL support**: Set custom expiration times (seconds, minutes, hours, days)
- **Cache invalidation**: Mark which cache entries should be invalidated on writes
- **Pluggable cache interface**: Use any cache implementation (Redis, in-memory, etc.)

## Usage

### Basic Cache Annotation

Add `@cache` annotation before your query:

```sql
-- GetUser retrieves a user by ID.
-- @cache ttl=5m
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
```

### Cache Key Patterns

Define custom cache key patterns using parameter placeholders:

```sql
-- GetUserByEmail retrieves a user by email.
-- @cache ttl=5m key=user:email:{email}
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
```

### TTL Options

- `ttl=30s` - 30 seconds
- `ttl=5m` - 5 minutes (default)
- `ttl=1h` - 1 hour
- `ttl=7d` - 7 days

### Cache Invalidation

Invalidate cached entries when data changes:

```sql
-- CreateUser inserts a new user.
-- name: CreateUser :exec
INSERT INTO users (name, email) VALUES ($1, $2);

-- UpdateUser updates a user.
-- @cache invalidate=users,user:{id}
-- name: UpdateUser :exec
UPDATE users SET name = $1, email = $2 WHERE id = $3;
```

## Generated Code

When you use cache annotations, db-catalyst generates:

1. **Cache Interface** in `querier.gen.go`:

```go
type Cache interface {
    Get(ctx context.Context, key string) (any, bool)
    Set(ctx context.Context, key string, value any, ttl time.Duration)
    Delete(ctx context.Context, key string)
    Invalidate(ctx context.Context, pattern string)
}
```

2. **Queries struct with cache field**:

```go
type Queries struct {
    db    DBTX
    cache Cache
}
```

3. **Cache-aware query methods**:

```go
func (q *Queries) GetUser(ctx context.Context, id int64) (GetUserRow, error) {
    // Build cache key
    cacheKey := "GetUser:id=" + fmt.Sprintf("%v", id)
    
    // Check cache
    if q.cache != nil {
        if cached, ok := q.cache.Get(ctx, cacheKey); ok {
            if result, ok := cached.(GetUserRow); ok {
                return result, nil
            }
        }
    }
    
    // Execute query
    rows, err := q.db.QueryContext(ctx, queryGetUser, id)
    if err != nil {
        return GetUserRow{}, err
    }
    defer rows.Close()
    
    // ... rest of query logic ...
    
    // Cache result
    if q.cache != nil {
        q.cache.Set(ctx, cacheKey, item, 300)
    }
    
    return item, nil
}
```

## Using the Cache

### In-Memory Cache

```go
import "github.com/electwix/db-catalyst/internal/cache"

// Create cache
c := cache.NewMemoryCache()

// Create queries with cache
queries := New(db)
queries.cache = c

// Use queries - results are automatically cached
user, err := queries.GetUser(ctx, 123)
```

### Redis Cache

Implement the Cache interface with your Redis client:

```go
type RedisCache struct {
    client *redis.Client
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, bool) {
    val, err := r.client.Get(ctx, key).Result()
    if err != nil {
        return nil, false
    }
    // Deserialize and return
    return val, true
}

func (r *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) {
    // Serialize and store
    r.client.Set(ctx, key, value, ttl)
}

func (r *RedisCache) Delete(ctx context.Context, key string) {
    r.client.Del(ctx, key)
}

func (r *RedisCache) Invalidate(ctx context.Context, pattern string) {
    // Delete keys matching pattern
}
```

## Best Practices

1. **Cache read-heavy queries**: Focus on queries that are executed frequently but change rarely
2. **Use appropriate TTLs**: Balance between freshness and performance
3. **Invalidate proactively**: Ensure writes invalidate relevant cache entries
4. **Avoid caching large result sets**: For `:many` queries, consider pagination
5. **Test cache behavior**: Verify cache hit rates in production

## Limitations

- Cache annotations only work with `:one` and `:many` queries
- `:exec` and `:execresult` queries don't cache results (but can invalidate)
- Dynamic slice parameters are excluded from cache keys
- Cache interface methods use `any` type for flexibility

## Examples

See `examples/cache_example/queries.sql` for complete examples.
