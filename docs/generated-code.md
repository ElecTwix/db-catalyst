# Generated Code Reference

db-catalyst generates idiomatic Go code that looks hand-written.

## Generated Files

| File | Contents |
|------|----------|
| `models.gen.go` | Table structs |
| `querier.gen.go` | Queries struct with methods |
| `_helpers.gen.go` | Row scanning helpers |
| `query_*.go` | Individual query methods |

## Models

```go
// Generated from CREATE TABLE users (...)
type User struct {
    ID        int64
    Username  string
    Email     *string  // nullable
    CreatedAt time.Time
}
```

With JSON tags (default):

```go
type User struct {
    ID        int64   `json:"id"`
    Username  string  `json:"username"`
    Email     *string `json:"email"`
}
```

## Queries Interface

```go
type Queries interface {
    GetUser(ctx context.Context, id int64) (User, error)
    ListUsers(ctx context.Context) ([]User, error)
    CreateUser(ctx context.Context, username, email string) (sql.Result, error)
}
```

## Usage

```go
db, err := sql.Open("sqlite3", "app.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

q := dbcatalyst.New(db)

// Single row
user, err := q.GetUser(ctx, 1)
if err == sql.ErrNoRows {
    // not found
}

// Multiple rows
users, err := q.ListUsers(ctx)
for _, u := range users {
    fmt.Println(u.Username)
}

// Exec
result, err := q.CreateUser(ctx, "alice", "alice@example.com")
id, _ := result.LastInsertId()
```

## Prepared Queries

Enable prepared statement caching:

```toml
[prepared_queries]
enabled = true
```

Generated code:

```go
type PreparedQueries struct {
    db *sql.DB
    // prepared statements cached here
}

func (p *PreparedQueries) GetUser(ctx context.Context, id int64) (User, error) {
    stmt, err := p.getUser(ctx)
    if err != nil {
        return User{}, err
    }
    rows, err := stmt.QueryContext(ctx, id)
    // ...
}
```

## Options

```toml
[prepared_queries]
enabled = true
emit_empty_slices = true    # make([]T, 0) vs var s []T
thread_safe = true          # mutex-protected statement prep
metrics = true              # observability hooks
```

## JSON Tags

Disable with config:

```toml
[generation]
emit_json_tags = false
```

Or CLI flag: `--no-json-tags`

## Thread Safety

With `thread_safe = true`, prepared statements are safely shared across goroutines with mutex protection.
