package integration_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	mysqlConn    = "testuser:testpass@tcp(127.0.0.1:3306)/dbtest"
	postgresConn = "postgres://testuser:testpass@127.0.0.1:5432/dbtest?sslmode=disable"
	sqliteDBPath = "/tmp/db-catalyst-integration-test.db"
)

func skipIfNoDocker(t *testing.T) {
	if os.Getenv("SKIP_DOCKER") == "true" {
		t.Skip("Skipping Docker tests")
	}
}

func TestMySQLIntegration(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("mysql", mysqlConn)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping MySQL: %v", err)
	}

	// Test basic operations
	t.Run("CreateUser", func(t *testing.T) {
		result, err := db.ExecContext(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
			"testuser_"+time.Now().Format("150405"), "test@example.com", "hashed")
		if err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}
		id, _ := result.LastInsertId()
		if id == 0 {
			t.Error("Expected non-zero ID")
		}
	})

	t.Run("ReadUser", func(t *testing.T) {
		var id int
		var username, email string
		err := db.QueryRowContext(ctx, "SELECT id, username, email FROM users LIMIT 1").Scan(&id, &username, &email)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}
		if id == 0 {
			t.Error("Expected non-zero ID from query")
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		_, err := db.ExecContext(ctx, "UPDATE users SET email = ? WHERE username = ?",
			"updated@example.com", "alice")
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		// Skip actual deletion to keep test data
		t.Skip("Skipping delete to preserve test data")
	})
}

func TestPostgresIntegration(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", postgresConn)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	// Test basic operations
	t.Run("CreateUser", func(t *testing.T) {
		result, err := db.ExecContext(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3)",
			"testuser_"+time.Now().Format("150405"), "test@example.com", "hashed")
		if err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}
		id, _ := result.RowsAffected()
		if id == 0 {
			t.Error("Expected non-zero rows affected")
		}
	})

	t.Run("ReadUser", func(t *testing.T) {
		var id int
		var username, email string
		err := db.QueryRowContext(ctx, "SELECT id, username, email FROM users LIMIT 1").Scan(&id, &username, &email)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}
		if id == 0 {
			t.Error("Expected non-zero ID from query")
		}
	})

	t.Run("Transaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		_, err = tx.ExecContext(ctx, "INSERT INTO posts (user_id, title, slug, content) VALUES ($1, $2, $3, $4)",
			1, "Test Post", "test-post-"+time.Now().Format("150405"), "Test content")
		if err != nil {
			t.Fatalf("Failed to insert post in transaction: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}
	})
}

func TestSQLiteIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Remove existing database
	os.RemoveAll(sqliteDBPath)

	db, err := sql.Open("sqlite3", sqliteDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	// Create schema
	schema := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Test basic operations
	t.Run("CreateUser", func(t *testing.T) {
		result, err := db.ExecContext(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
			"testuser", "test@example.com", "hashed")
		if err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}
		id, _ := result.LastInsertId()
		if id == 0 {
			t.Error("Expected non-zero ID")
		}
	})

	t.Run("CreatePostWithForeignKey", func(t *testing.T) {
		_, err := db.ExecContext(ctx,
			"INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)",
			1, "Test Post", "Test content")
		if err != nil {
			t.Fatalf("Failed to insert post: %v", err)
		}
	})

	t.Run("JoinQuery", func(t *testing.T) {
		rows, err := db.QueryContext(ctx,
			"SELECT u.username, p.title FROM users u JOIN posts p ON u.id = p.user_id WHERE u.id = ?", 1)
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		if count == 0 {
			t.Error("Expected at least one row from JOIN")
		}
	})

	t.Run("Transactions", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		_, err = tx.ExecContext(ctx, "INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
			"txuser", "tx@example.com", "hashed")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed in transaction: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
	})

	t.Run("PreparedStatements", func(t *testing.T) {
		stmt, err := db.PrepareContext(ctx, "SELECT * FROM users WHERE username = ?")
		if err != nil {
			t.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()

		rows, err := stmt.QueryContext(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to query with prepared statement: %v", err)
		}
		defer rows.Close()

		if !rows.Next() {
			t.Error("Expected at least one row")
		}
	})

	// Cleanup
	os.RemoveAll(sqliteDBPath)
}

func TestGeneratedSchemaValidity(t *testing.T) {
	skipIfNoDocker(t)

	schemas := []struct {
		name   string
		conn   string
		driver string
	}{
		{"MySQL", mysqlConn, "mysql"},
		{"PostgreSQL", postgresConn, "postgres"},
	}

	for _, schema := range schemas {
		t.Run(schema.name, func(t *testing.T) {
			db, err := sql.Open(schema.driver, schema.conn)
			if err != nil {
				t.Fatalf("Failed to connect to %s: %v", schema.name, err)
			}
			defer db.Close()

			// Verify tables exist
			tables := []string{"users", "posts", "comments", "tags", "post_tags"}
			for _, table := range tables {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&count)
				if err != nil {
					t.Errorf("Failed to check table %s: %v", table, err)
					continue
				}
				if count == 0 {
					t.Errorf("Table %s does not exist", table)
				}
			}

			// Verify foreign keys
			t.Run("ForeignKeys", func(t *testing.T) {
				var fkCount int
				err := db.QueryRow(`
					SELECT COUNT(*) FROM information_schema.table_constraints 
					WHERE constraint_type = 'FOREIGN KEY'`).Scan(&fkCount)
				if err != nil {
					t.Errorf("Failed to check foreign keys: %v", err)
				}
				if fkCount == 0 {
					t.Error("Expected foreign keys to exist")
				}
			})

			// Verify indexes
			t.Run("Indexes", func(t *testing.T) {
				var idxCount int
				err := db.QueryRow(`
					SELECT COUNT(*) FROM information_schema.statistics 
					WHERE table_schema = DATABASE()`).Scan(&idxCount)
				if err != nil {
					t.Errorf("Failed to check indexes: %v", err)
				}
				if idxCount < 5 {
					t.Error("Expected multiple indexes")
				}
			})
		})
	}
}
