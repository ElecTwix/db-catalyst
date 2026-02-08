# MySQL Example

This example demonstrates db-catalyst's MySQL support with a blog application schema.

## Features Demonstrated

- **MySQL engine integration**: Using `--database mysql` flag
- **Auto-increment**: INTEGER PRIMARY KEY AUTOINCREMENT
- **Indexes**: Support for INDEX declarations
- **Foreign keys**: Basic FOREIGN KEY syntax
- **Many-to-many**: Posts <-> Tags relationship via junction table
- **Query generation**: Full CRUD operations with JOINs

## Schema Overview

```
users
├── id (INTEGER, AUTOINCREMENT, PK)
├── email (TEXT, UNIQUE)
├── username (TEXT, UNIQUE)
├── password_hash (TEXT)
├── status (TEXT with DEFAULT)
└── created_at (TEXT)

posts
├── id (INTEGER, AUTOINCREMENT, PK)
├── user_id (INTEGER, FK -> users)
├── title (TEXT)
├── content (TEXT)
├── status (TEXT with DEFAULT)
├── view_count (INTEGER with DEFAULT)
└── created_at (TEXT)

comments
├── id (INTEGER, AUTOINCREMENT, PK)
├── post_id (INTEGER, FK -> posts)
├── user_id (INTEGER, FK -> users)
├── content (TEXT)
└── created_at (TEXT)

tags
├── id (INTEGER, AUTOINCREMENT, PK)
├── name (TEXT, UNIQUE)
├── description (TEXT)
└── created_at (TEXT)

post_tags (junction table)
├── post_id (INTEGER, FK -> posts)
├── tag_id (INTEGER, FK -> tags)
├── created_at (TEXT)
└── PK (post_id, tag_id)
```

## Running the Example

1. Generate the code:
```bash
cd examples/mysql
db-catalyst --config db-catalyst.toml
```

2. The generated code will be in the `db/` directory:
- `models.gen.go` - Struct definitions
- `querier.gen.go` - Querier interface
- `query_*.gen.go` - Individual query implementations
- `helpers.gen.go` - Helper functions

## Generated Code

### Models

```go
type Users struct {
    ID           int32
    Email        string
    Username     string
    PasswordHash string
    Status       sql.NullString
    CreatedAt    sql.NullString
}

type Posts struct {
    ID          int32
    UserID      int32
    Title       string
    Content     sql.NullString
    Status      sql.NullString
    ViewCount   *int32
    CreatedAt   sql.NullString
}
```

### Query Interface

```go
type Querier interface {
    // Users
    GetUser(ctx context.Context, id int32) (Users, error)
    GetUserByEmail(ctx context.Context, email string) (Users, error)
    CreateUser(ctx context.Context, arg CreateUserParams) (sql.Result, error)
    ListUsers(ctx context.Context, arg ListUsersParams) ([]Users, error)
    
    // Posts
    GetPost(ctx context.Context, id int32) (Posts, error)
    CreatePost(ctx context.Context, arg CreatePostParams) (sql.Result, error)
    ListPostsByUser(ctx context.Context, arg ListPostsByUserParams) ([]Posts, error)
    IncrementPostViews(ctx context.Context, id int32) error
    
    // Comments
    CreateComment(ctx context.Context, arg CreateCommentParams) (sql.Result, error)
    ListCommentsByPost(ctx context.Context, arg ListCommentsByPostParams) ([]Comments, error)
    
    // Tags
    GetTag(ctx context.Context, id int32) (Tags, error)
    CreateTag(ctx context.Context, arg CreateTagParams) (sql.Result, error)
    GetTagsForPost(ctx context.Context, postID int32) ([]Tags, error)
    GetPostsForTag(ctx context.Context, arg GetPostsForTagParams) ([]Posts, error)
    AddTagToPost(ctx context.Context, arg AddTagToPostParams) error
    RemoveTagFromPost(ctx context.Context, arg RemoveTagFromPostParams) error
}
```

## MySQL-Specific Notes

### Placeholders
MySQL uses `?` placeholders instead of PostgreSQL's `$1`:
```sql
-- MySQL
SELECT * FROM users WHERE id = ?;

-- PostgreSQL
SELECT * FROM users WHERE id = $1;
```

### Driver
The generated code uses `database/sql` with the MySQL driver:
```go
import _ "github.com/go-sql-driver/mysql"

db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/blog")
```

## Configuration

```toml
# db-catalyst.toml
package = "mysqlblog"
out = "db"
database = "mysql"
schemas = ["schemas/*.sql"]
queries = ["queries/*.sql"]

[generation]
emit_prepared_queries = true
emit_json_tags = true
```

## Known Limitations

The current MySQL DDL parser has some limitations:

1. **MySQL-specific types**: ENUM, JSON, SET types are not yet fully supported
2. **Table options**: ENGINE=InnoDB, CHARSET=utf8mb4 clauses may cause parsing issues
3. **Complex defaults**: TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE is not supported
4. **Column attributes**: UNSIGNED, ZEROFILL may not be recognized

**Workaround**: Use SQLite-compatible syntax which MySQL also supports:
```sql
-- Use this (works in both)
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE
);

-- Instead of MySQL-specific syntax
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE
) ENGINE=InnoDB;
```

## Requirements

- MySQL 5.7+ or MariaDB 10.2+
- Go 1.22+
- github.com/go-sql-driver/mysql

## Example Usage

```go
package main

import (
    "context"
    "database/sql"
    "log"
    
    "github.com/electwix/db-catalyst/examples/mysql/db"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    dbConn, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/blog")
    if err != nil {
        log.Fatal(err)
    }
    defer dbConn.Close()
    
    ctx := context.Background()
    querier := db.New(dbConn)
    
    // Create a user
    result, err := querier.CreateUser(ctx, db.CreateUserParams{
        Email:        "user@example.com",
        Username:     "johndoe",
        PasswordHash: "hashed_password",
        Status:       sql.NullString{String: "active", Valid: true},
    })
    
    // Get user by ID
    user, err := querier.GetUser(ctx, 1)
    
    // List posts by user
    posts, err := querier.ListPostsByUser(ctx, db.ListPostsByUserParams{
        UserID: 1,
        Status: sql.NullString{String: "published", Valid: true},
        Limit:  10,
    })
}
```

## Future Improvements

- Full support for MySQL-specific types (ENUM, JSON, SET)
- Support for table options (ENGINE, CHARSET, COLLATE)
- Support for complex DEFAULT expressions
- FULLTEXT index support
- Support for MySQL-specific query hints
