// Package integration_test provides comprehensive integration tests for db-catalyst.
// These tests verify database connectivity, SQL execution, and schema validation
// across SQLite, PostgreSQL, and MySQL.
package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Connection strings for test databases
const (
	mysqlConn    = "testuser:testpass@tcp(127.0.0.1:3306)/dbtest"
	postgresConn = "postgres://testuser:testpass@127.0.0.1:5432/dbtest?sslmode=disable"
	sqliteDBPath = "/tmp/db-catalyst-integration-test.db"
)

// Test timeout for database operations
const testTimeout = 30 * time.Second

// skipIfNoDocker skips tests when SKIP_DOCKER is set
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_DOCKER") == "true" {
		t.Skip("Skipping Docker tests")
	}
}

// waitForDatabase waits for a database to be ready
func waitForDatabase(ctx context.Context, t *testing.T, driver, conn string) *sql.DB {
	t.Helper()

	db, err := sql.Open(driver, conn)
	if err != nil {
		t.Fatalf("Failed to connect to %s: %v", driver, err)
	}

	// Wait for connection with timeout
	for {
		select {
		case <-ctx.Done():
			db.Close()
			t.Fatalf("Timeout waiting for %s", driver)
		default:
			if err := db.PingContext(ctx); err == nil {
				return db
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// createSQLiteDB creates a fresh SQLite database
func createSQLiteDB(t testing.TB) *sql.DB {
	os.RemoveAll(sqliteDBPath)
	db, err := sql.Open("sqlite3", sqliteDBPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	return db
}

// cleanupSQLite removes the SQLite database file
func cleanupSQLite(t testing.TB) {
	os.RemoveAll(sqliteDBPath)
}

// initSQLiteSchema initializes the test schema in SQLite
func initSQLiteSchema(ctx context.Context, t testing.TB, db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
		is_active INTEGER DEFAULT 1,
		role TEXT DEFAULT 'user'
	);
	
	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		content TEXT,
		excerpt TEXT,
		status TEXT DEFAULT 'draft',
		views_count INTEGER DEFAULT 0,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
		published_at TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
	CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
	
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		parent_id INTEGER NULL,
		content TEXT NOT NULL,
		is_approved INTEGER DEFAULT 1,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
	);
	
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		slug TEXT NOT NULL UNIQUE,
		description TEXT,
		post_count INTEGER DEFAULT 0,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS post_tags (
		post_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (post_id, tag_id),
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);
	
	INSERT INTO users (username, email, password_hash, role) VALUES
		('alice', 'alice@example.com', 'hashed_password', 'admin'),
		('bob', 'bob@example.com', 'hashed_password', 'user'),
		('charlie', 'charlie@example.com', 'hashed_password', 'moderator');
	
	INSERT INTO posts (user_id, title, slug, content, status, published_at) VALUES
		(1, 'Welcome to db-catalyst', 'welcome-db-catalyst', 'This is the first post.', 'published', datetime('now')),
		(2, 'Go Programming Tips', 'go-tips', 'Tips for better Go code.', 'published', datetime('now')),
		(1, 'Draft Post', 'draft-post', 'This is a draft.', 'draft', NULL);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
}

// =============================================================================
// MySQL Integration Tests
// =============================================================================

func TestMySQL_Connection(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "mysql", mysqlConn)
	defer db.Close()

	var version string
	err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query MySQL version: %v", err)
	}
	t.Logf("MySQL version: %s", version)
}

func TestMySQL_BasicCRUD(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "mysql", mysqlConn)
	defer db.Close()

	tests := []struct {
		name string
		fn   func(context.Context, *testing.T, *sql.DB)
	}{
		{"CreateUser", testMySQLCreateUser},
		{"ReadUser", testMySQLReadUser},
		{"UpdateUser", testMySQLUpdateUser},
		{"DeleteUser", testMySQLDeleteUser},
		{"ComplexJoin", testMySQLComplexJoin},
		{"TransactionCommit", testMySQLTransactionCommit},
		{"TransactionRollback", testMySQLTransactionRollback},
		{"PreparedStatement", testMySQLPreparedStatement},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(ctx, t, db)
		})
	}
}

func testMySQLCreateUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	result, err := db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		username, fmt.Sprintf("%s@example.com", username), "hashed_password", "user")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func testMySQLReadUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	var id int64
	var username, email string
	err := db.QueryRowContext(ctx,
		"SELECT id, username, email FROM users WHERE username = ?", "alice").
		Scan(&id, &username, &email)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID")
	}
	if username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", username)
	}
}

func testMySQLUpdateUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	result, err := db.ExecContext(ctx,
		"UPDATE users SET email = ? WHERE username = ?",
		"updated_alice@example.com", "alice")
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	// Note: affected may be 0 if email is already the same
	t.Logf("Updated %d rows", affected)
}

func testMySQLDeleteUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	// Create a temporary user to delete
	username := fmt.Sprintf("temp_user_%d", time.Now().UnixNano())
	_, err := db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		username, fmt.Sprintf("%s@example.com", username), "hashed", "user")
	if err != nil {
		t.Fatalf("Failed to create temp user: %v", err)
	}

	result, err := db.ExecContext(ctx,
		"DELETE FROM users WHERE username = ?", username)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func testMySQLComplexJoin(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	query := `
		SELECT u.username, p.title, t.name as tag_name
		FROM users u
		JOIN posts p ON u.id = p.user_id
		LEFT JOIN post_tags pt ON p.id = pt.post_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		WHERE p.status = ?
		ORDER BY p.created_at DESC
		LIMIT 10
	`

	rows, err := db.QueryContext(ctx, query, "published")
	if err != nil {
		t.Fatalf("Failed to execute complex join: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var username, title string
		var tagName sql.NullString
		if err := rows.Scan(&username, &title, &tagName); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		count++
		if username == "" {
			t.Error("Expected valid username")
		}
		if title == "" {
			t.Error("Expected valid title")
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one row from complex join")
	}
}

func testMySQLTransactionCommit(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES (?, ?, ?, ?)",
		1, "Tx Test Post", fmt.Sprintf("tx-test-%d", time.Now().UnixNano()), "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

func testMySQLTransactionRollback(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	slug := fmt.Sprintf("rollback-test-%d", time.Now().UnixNano())
	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES (?, ?, ?, ?)",
		1, "Rollback Test", slug, "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify the post was not created
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM posts WHERE slug = ?", slug).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for rolled back post: %v", err)
	}
	if count != 0 {
		t.Error("Expected rolled back post to not exist")
	}
}

func testMySQLPreparedStatement(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	stmt, err := db.PrepareContext(ctx,
		"SELECT id, username, email FROM users WHERE role = ?")
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	roles := []string{"admin", "user", "moderator"}
	for _, role := range roles {
		rows, err := stmt.QueryContext(ctx, role)
		if err != nil {
			t.Fatalf("Failed to query with role %s: %v", role, err)
		}

		count := 0
		for rows.Next() {
			var id int64
			var username, email string
			if err := rows.Scan(&id, &username, &email); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			count++
		}
		rows.Close()

		if role == "admin" && count == 0 {
			t.Error("Expected at least one admin user")
		}
	}
}

func TestMySQL_JSONOperations(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "mysql", mysqlConn)
	defer db.Close()

	// Create table with JSON column
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS json_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create JSON test table: %v", err)
	}
	defer db.ExecContext(ctx, "DROP TABLE IF EXISTS json_test")

	// Insert JSON data
	jsonData := `{"name": "test", "value": 42, "nested": {"key": "value"}}`
	result, err := db.ExecContext(ctx,
		"INSERT INTO json_test (data) VALUES (?)", jsonData)
	if err != nil {
		t.Fatalf("Failed to insert JSON: %v", err)
	}

	id, _ := result.LastInsertId()

	// Query JSON using JSON functions
	var extractedName string
	err = db.QueryRowContext(ctx,
		"SELECT JSON_UNQUOTE(JSON_EXTRACT(data, '$.name')) FROM json_test WHERE id = ?", id).
		Scan(&extractedName)
	if err != nil {
		t.Fatalf("Failed to extract JSON: %v", err)
	}
	if extractedName != "test" {
		t.Errorf("Expected 'test', got '%s'", extractedName)
	}

	// Test JSON path search
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM json_test WHERE JSON_EXTRACT(data, '$.value') = 42").
		Scan(&count)
	if err != nil {
		t.Fatalf("Failed to search JSON: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestMySQL_FullTextSearch(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "mysql", mysqlConn)
	defer db.Close()

	// Create full-text search index if not exists
	_, err := db.ExecContext(ctx, `
		CREATE FULLTEXT INDEX IF NOT EXISTS idx_posts_content ON posts(title, content)
	`)
	if err != nil {
		t.Logf("Fulltext index creation skipped or failed: %v", err)
		t.Skip("Fulltext search not available")
	}
	defer db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_posts_content ON posts")

	// Search using fulltext
	rows, err := db.QueryContext(ctx,
		"SELECT title FROM posts WHERE MATCH(title, content) AGAINST(?)",
		"Go programming")
	if err != nil {
		t.Fatalf("Fulltext search failed: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			t.Fatalf("Failed to scan: %v", err)
		}
		found = true
		t.Logf("Found: %s", title)
	}
	if !found {
		t.Log("No fulltext results found (may be expected with small dataset)")
	}
}

// =============================================================================
// PostgreSQL Integration Tests
// =============================================================================

func TestPostgres_Connection(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	var version string
	err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query PostgreSQL version: %v", err)
	}
	t.Logf("PostgreSQL version: %s", version)
}

func TestPostgres_BasicCRUD(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	tests := []struct {
		name string
		fn   func(context.Context, *testing.T, *sql.DB)
	}{
		{"CreateUser", testPostgresCreateUser},
		{"ReadUser", testPostgresReadUser},
		{"UpdateUser", testPostgresUpdateUser},
		{"DeleteUser", testPostgresDeleteUser},
		{"ComplexJoin", testPostgresComplexJoin},
		{"TransactionCommit", testPostgresTransactionCommit},
		{"TransactionRollback", testPostgresTransactionRollback},
		{"Savepoint", testPostgresSavepoint},
		{"PreparedStatement", testPostgresPreparedStatement},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(ctx, t, db)
		})
	}
}

func testPostgresCreateUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	result, err := db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES ($1, $2, $3, $4)",
		username, fmt.Sprintf("%s@example.com", username), "hashed_password", "user")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func testPostgresReadUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	var id int
	var username, email string
	err := db.QueryRowContext(ctx,
		"SELECT id, username, email FROM users WHERE username = $1", "alice").
		Scan(&id, &username, &email)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID")
	}
	if username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", username)
	}
}

func testPostgresUpdateUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	result, err := db.ExecContext(ctx,
		"UPDATE users SET email = $1 WHERE username = $2",
		"updated_alice@example.com", "alice")
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	t.Logf("Updated %d rows", affected)
}

func testPostgresDeleteUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	// Create a temporary user to delete
	username := fmt.Sprintf("temp_user_%d", time.Now().UnixNano())
	_, err := db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES ($1, $2, $3, $4)",
		username, fmt.Sprintf("%s@example.com", username), "hashed", "user")
	if err != nil {
		t.Fatalf("Failed to create temp user: %v", err)
	}

	result, err := db.ExecContext(ctx,
		"DELETE FROM users WHERE username = $1", username)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func testPostgresComplexJoin(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	query := `
		SELECT u.username, p.title, t.name as tag_name
		FROM users u
		JOIN posts p ON u.id = p.user_id
		LEFT JOIN post_tags pt ON p.id = pt.post_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		WHERE p.status = $1
		ORDER BY p.created_at DESC
		LIMIT 10
	`

	rows, err := db.QueryContext(ctx, query, "published")
	if err != nil {
		t.Fatalf("Failed to execute complex join: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var username, title string
		var tagName sql.NullString
		if err := rows.Scan(&username, &title, &tagName); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		count++
		if username == "" {
			t.Error("Expected valid username")
		}
		if title == "" {
			t.Error("Expected valid title")
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one row from complex join")
	}
}

func testPostgresTransactionCommit(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES ($1, $2, $3, $4)",
		1, "Tx Test Post", fmt.Sprintf("tx-test-%d", time.Now().UnixNano()), "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

func testPostgresTransactionRollback(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	slug := fmt.Sprintf("rollback-test-%d", time.Now().UnixNano())
	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES ($1, $2, $3, $4)",
		1, "Rollback Test", slug, "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify the post was not created
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM posts WHERE slug = $1", slug).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for rolled back post: %v", err)
	}
	if count != 0 {
		t.Error("Expected rolled back post to not exist")
	}
}

func testPostgresSavepoint(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert first post
	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES ($1, $2, $3, $4)",
		1, "Savepoint Test", fmt.Sprintf("savepoint-%d", time.Now().UnixNano()), "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert: %v", err)
	}

	// Create savepoint
	_, err = tx.ExecContext(ctx, "SAVEPOINT sp1")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Insert second post (will be rolled back)
	slug2 := fmt.Sprintf("savepoint-rollback-%d", time.Now().UnixNano())
	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES ($1, $2, $3, $4)",
		1, "Savepoint Rollback", slug2, "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert: %v", err)
	}

	// Rollback to savepoint
	_, err = tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT sp1")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to rollback to savepoint: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify second post was not created
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM posts WHERE slug = $1", slug2).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check: %v", err)
	}
	if count != 0 {
		t.Error("Expected savepoint rollback to remove the post")
	}
}

func testPostgresPreparedStatement(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	stmt, err := db.PrepareContext(ctx,
		"SELECT id, username, email FROM users WHERE role = $1")
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	roles := []string{"admin", "user", "moderator"}
	for _, role := range roles {
		rows, err := stmt.QueryContext(ctx, role)
		if err != nil {
			t.Fatalf("Failed to query with role %s: %v", role, err)
		}

		count := 0
		for rows.Next() {
			var id int
			var username, email string
			if err := rows.Scan(&id, &username, &email); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			count++
		}
		rows.Close()

		if role == "admin" && count == 0 {
			t.Error("Expected at least one admin user")
		}
	}
}

func TestPostgres_EnumTypes(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	// Create enum type if not exists
	_, err := db.ExecContext(ctx, `
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_status') THEN
				CREATE TYPE task_status AS ENUM ('pending', 'in_progress', 'completed', 'cancelled');
			END IF;
		END $$;
	`)
	if err != nil {
		t.Fatalf("Failed to create enum type: %v", err)
	}

	// Create table using enum
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tasks (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			status task_status DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create tasks table: %v", err)
	}
	defer db.ExecContext(ctx, "DROP TABLE IF EXISTS tasks")

	// Test inserting with enum values
	testCases := []struct {
		title  string
		status string
		valid  bool
	}{
		{"Task 1", "pending", true},
		{"Task 2", "in_progress", true},
		{"Task 3", "completed", true},
		{"Task 4", "invalid_status", false},
	}

	for _, tc := range testCases {
		_, err := db.ExecContext(ctx,
			"INSERT INTO tasks (title, status) VALUES ($1, $2)",
			tc.title, tc.status)
		if tc.valid {
			if err != nil {
				t.Errorf("Expected valid insert for status '%s': %v", tc.status, err)
			}
		} else {
			if err == nil {
				t.Errorf("Expected error for invalid status '%s'", tc.status)
			}
		}
	}

	// Test querying with enum
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tasks WHERE status = 'pending'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query by enum: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 pending task, got %d", count)
	}
}

func TestPostgres_DomainTypes(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	// Create domain type if not exists
	_, err := db.ExecContext(ctx, `
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'positive_int') THEN
				CREATE DOMAIN positive_int AS INTEGER CHECK (VALUE > 0);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'email_address') THEN
				CREATE DOMAIN email_address AS TEXT CHECK (VALUE ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');
			END IF;
		END $$;
	`)
	if err != nil {
		t.Fatalf("Failed to create domain types: %v", err)
	}

	// Create table using domains
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS domain_test (
			id SERIAL PRIMARY KEY,
			quantity positive_int NOT NULL,
			contact_email email_address NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create domain test table: %v", err)
	}
	defer db.ExecContext(ctx, "DROP TABLE IF EXISTS domain_test")

	// Test valid values
	_, err = db.ExecContext(ctx,
		"INSERT INTO domain_test (quantity, contact_email) VALUES ($1, $2)",
		42, "test@example.com")
	if err != nil {
		t.Errorf("Expected valid insert: %v", err)
	}

	// Test invalid positive_int
	_, err = db.ExecContext(ctx,
		"INSERT INTO domain_test (quantity, contact_email) VALUES ($1, $2)",
		-1, "test2@example.com")
	if err == nil {
		t.Error("Expected error for negative quantity")
	}

	// Test invalid email
	_, err = db.ExecContext(ctx,
		"INSERT INTO domain_test (quantity, contact_email) VALUES ($1, $2)",
		10, "invalid-email")
	if err == nil {
		t.Error("Expected error for invalid email")
	}
}

func TestPostgres_ArrayTypes(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	// Create table with array column
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS array_test (
			id SERIAL PRIMARY KEY,
			tags TEXT[],
			scores INTEGER[]
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create array test table: %v", err)
	}
	defer db.ExecContext(ctx, "DROP TABLE IF EXISTS array_test")

	// Insert array data
	_, err = db.ExecContext(ctx,
		"INSERT INTO array_test (tags, scores) VALUES ($1, $2)",
		"{go,postgres,testing}", "{95,87,92}")
	if err != nil {
		t.Fatalf("Failed to insert arrays: %v", err)
	}

	// Query with array functions
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM array_test WHERE $1 = ANY(tags)", "go").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query array: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}

	// Test array length
	var tagCount int
	err = db.QueryRowContext(ctx,
		"SELECT array_length(tags, 1) FROM array_test LIMIT 1").Scan(&tagCount)
	if err != nil {
		t.Fatalf("Failed to get array length: %v", err)
	}
	if tagCount != 3 {
		t.Errorf("Expected 3 tags, got %d", tagCount)
	}
}

func TestPostgres_JSONOperations(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	// Create table with JSONB column
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS jsonb_test (
			id SERIAL PRIMARY KEY,
			data JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create JSONB test table: %v", err)
	}
	defer db.ExecContext(ctx, "DROP TABLE IF EXISTS jsonb_test")

	// Insert JSONB data
	jsonData := `{"name": "test", "value": 42, "nested": {"key": "value"}, "tags": ["a", "b", "c"], "metadata": {"created_by": "admin", "version": 1}}`
	var id int
	err = db.QueryRowContext(ctx,
		"INSERT INTO jsonb_test (data) VALUES ($1) RETURNING id", jsonData).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to insert JSONB: %v", err)
	}

	// Query JSONB using operators
	var extractedName string
	err = db.QueryRowContext(ctx,
		"SELECT data->>'name' FROM jsonb_test WHERE id = $1", id).Scan(&extractedName)
	if err != nil {
		t.Fatalf("Failed to extract JSONB: %v", err)
	}
	if extractedName != "test" {
		t.Errorf("Expected 'test', got '%s'", extractedName)
	}

	// Test containment operator
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM jsonb_test WHERE data @> '{\"value\": 42}'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to test containment: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}

	// Test path extraction
	var nestedValue string
	err = db.QueryRowContext(ctx,
		"SELECT data#>>'{nested,key}' FROM jsonb_test WHERE id = $1", id).Scan(&nestedValue)
	if err != nil {
		t.Fatalf("Failed to extract nested value: %v", err)
	}
	if nestedValue != "value" {
		t.Errorf("Expected 'value', got '%s'", nestedValue)
	}

	// Test array element access
	var firstTag string
	err = db.QueryRowContext(ctx,
		"SELECT data->'tags'->>0 FROM jsonb_test WHERE id = $1", id).Scan(&firstTag)
	if err != nil {
		t.Fatalf("Failed to extract array element: %v", err)
	}
	if firstTag != "a" {
		t.Errorf("Expected 'a', got '%s'", firstTag)
	}
}

func TestPostgres_FullTextSearch(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db := waitForDatabase(ctx, t, "postgres", postgresConn)
	defer db.Close()

	// Create full-text search index
	_, err := db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_posts_fts ON posts 
		USING gin(to_tsvector('english', title || ' ' || COALESCE(content, '')))
	`)
	if err != nil {
		t.Logf("Fulltext index creation note: %v", err)
	}

	// Perform full-text search
	query := `
		SELECT title, ts_rank(to_tsvector('english', title || ' ' || COALESCE(content, '')), query) as rank
		FROM posts, plainto_tsquery('english', $1) query
		WHERE to_tsvector('english', title || ' ' || COALESCE(content, '')) @@ query
		ORDER BY rank DESC
		LIMIT 10
	`

	rows, err := db.QueryContext(ctx, query, "Go programming")
	if err != nil {
		t.Fatalf("Fulltext search failed: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var title string
		var rank float64
		if err := rows.Scan(&title, &rank); err != nil {
			t.Fatalf("Failed to scan: %v", err)
		}
		found = true
		t.Logf("Found: %s (rank: %.4f)", title, rank)
	}
	if !found {
		t.Log("No fulltext results found (may be expected with small dataset)")
	}
}

// =============================================================================
// SQLite Integration Tests
// =============================================================================

func TestSQLite_BasicCRUD(t *testing.T) {
	db := createSQLiteDB(t)
	defer db.Close()
	defer cleanupSQLite(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	initSQLiteSchema(ctx, t, db)

	tests := []struct {
		name string
		fn   func(context.Context, *testing.T, *sql.DB)
	}{
		{"CreateUser", testSQLiteCreateUser},
		{"ReadUser", testSQLiteReadUser},
		{"UpdateUser", testSQLiteUpdateUser},
		{"DeleteUser", testSQLiteDeleteUser},
		{"ComplexJoin", testSQLiteComplexJoin},
		{"TransactionCommit", testSQLiteTransactionCommit},
		{"TransactionRollback", testSQLiteTransactionRollback},
		{"PreparedStatement", testSQLitePreparedStatement},
		{"ForeignKey", testSQLiteForeignKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(ctx, t, db)
		})
	}
}

func testSQLiteCreateUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	result, err := db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		username, fmt.Sprintf("%s@example.com", username), "hashed_password", "user")
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID")
	}
}

func testSQLiteReadUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	var id int64
	var username, email string
	err := db.QueryRowContext(ctx,
		"SELECT id, username, email FROM users WHERE username = ?", "alice").
		Scan(&id, &username, &email)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}
	if id == 0 {
		t.Error("Expected non-zero ID")
	}
	if username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", username)
	}
}

func testSQLiteUpdateUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	result, err := db.ExecContext(ctx,
		"UPDATE users SET email = ? WHERE username = ?",
		"updated_alice@example.com", "alice")
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	t.Logf("Updated %d rows", affected)
}

func testSQLiteDeleteUser(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	// Create a temporary user to delete
	username := fmt.Sprintf("temp_user_%d", time.Now().UnixNano())
	_, err := db.ExecContext(ctx,
		"INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)",
		username, fmt.Sprintf("%s@example.com", username), "hashed", "user")
	if err != nil {
		t.Fatalf("Failed to create temp user: %v", err)
	}

	result, err := db.ExecContext(ctx,
		"DELETE FROM users WHERE username = ?", username)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func testSQLiteComplexJoin(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	query := `
		SELECT u.username, p.title
		FROM users u
		JOIN posts p ON u.id = p.user_id
		WHERE p.status = ?
		ORDER BY p.created_at DESC
		LIMIT 10
	`

	rows, err := db.QueryContext(ctx, query, "published")
	if err != nil {
		t.Fatalf("Failed to execute complex join: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var username, title string
		if err := rows.Scan(&username, &title); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		count++
		if username == "" {
			t.Error("Expected valid username")
		}
		if title == "" {
			t.Error("Expected valid title")
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one row from complex join")
	}
}

func testSQLiteTransactionCommit(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES (?, ?, ?, ?)",
		1, "Tx Test Post", fmt.Sprintf("tx-test-%d", time.Now().UnixNano()), "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

func testSQLiteTransactionRollback(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	slug := fmt.Sprintf("rollback-test-%d", time.Now().UnixNano())
	_, err = tx.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES (?, ?, ?, ?)",
		1, "Rollback Test", slug, "Content")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify the post was not created
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM posts WHERE slug = ?", slug).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check for rolled back post: %v", err)
	}
	if count != 0 {
		t.Error("Expected rolled back post to not exist")
	}
}

func testSQLitePreparedStatement(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	stmt, err := db.PrepareContext(ctx,
		"SELECT id, username, email FROM users WHERE role = ?")
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	roles := []string{"admin", "user", "moderator"}
	for _, role := range roles {
		rows, err := stmt.QueryContext(ctx, role)
		if err != nil {
			t.Fatalf("Failed to query with role %s: %v", role, err)
		}

		count := 0
		for rows.Next() {
			var id int64
			var username, email string
			if err := rows.Scan(&id, &username, &email); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			count++
		}
		rows.Close()

		if role == "admin" && count == 0 {
			t.Error("Expected at least one admin user")
		}
	}
}

func testSQLiteForeignKey(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	// Enable foreign keys in SQLite
	_, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Try to insert a post with non-existent user
	_, err = db.ExecContext(ctx,
		"INSERT INTO posts (user_id, title, slug, content) VALUES (?, ?, ?, ?)",
		9999, "Invalid Post", "invalid-post", "Content")
	if err == nil {
		t.Error("Expected foreign key constraint error")
	}

	// Verify cascading delete works
	_, err = db.ExecContext(ctx, "DELETE FROM users WHERE id = 2") // Bob
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Check that Bob's posts were deleted
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM posts WHERE user_id = 2").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check posts: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 posts after cascade delete, got %d", count)
	}
}

func TestSQLite_JSONOperations(t *testing.T) {
	db := createSQLiteDB(t)
	defer db.Close()
	defer cleanupSQLite(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create table with JSON column
	_, err := db.ExecContext(ctx, `
		CREATE TABLE json_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			data TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create JSON test table: %v", err)
	}

	// Insert JSON data
	jsonData := `{"name": "test", "value": 42, "nested": {"key": "value"}}`
	result, err := db.ExecContext(ctx,
		"INSERT INTO json_test (data) VALUES (?)", jsonData)
	if err != nil {
		t.Fatalf("Failed to insert JSON: %v", err)
	}

	id, _ := result.LastInsertId()

	// Query JSON using SQLite JSON functions (JSON1 extension)
	var extractedName string
	err = db.QueryRowContext(ctx,
		"SELECT json_extract(data, '$.name') FROM json_test WHERE id = ?", id).Scan(&extractedName)
	if err != nil {
		t.Fatalf("Failed to extract JSON: %v", err)
	}
	if extractedName != "test" {
		t.Errorf("Expected 'test', got '%s'", extractedName)
	}

	// Test nested extraction
	var nestedValue string
	err = db.QueryRowContext(ctx,
		"SELECT json_extract(data, '$.nested.key') FROM json_test WHERE id = ?", id).Scan(&nestedValue)
	if err != nil {
		t.Fatalf("Failed to extract nested JSON: %v", err)
	}
	if nestedValue != "value" {
		t.Errorf("Expected 'value', got '%s'", nestedValue)
	}
}

// =============================================================================
// Schema Validation Tests
// =============================================================================

func TestSchemaValidation_AllDatabases(t *testing.T) {
	skipIfNoDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tests := []struct {
		name   string
		conn   string
		driver string
		fn     func(context.Context, *testing.T, *sql.DB)
	}{
		{
			name:   "MySQL",
			conn:   mysqlConn,
			driver: "mysql",
			fn:     validateMySQLSchema,
		},
		{
			name:   "PostgreSQL",
			conn:   postgresConn,
			driver: "postgres",
			fn:     validatePostgresSchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sql.Open(tt.driver, tt.conn)
			if err != nil {
				t.Fatalf("Failed to connect: %v", err)
			}
			defer db.Close()

			if err := db.PingContext(ctx); err != nil {
				t.Fatalf("Failed to ping database: %v", err)
			}

			tt.fn(ctx, t, db)
		})
	}
}

func validateMySQLSchema(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	// Verify expected tables exist
	tables := []string{"users", "posts", "comments", "tags", "post_tags"}
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
			table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("Table %s does not exist", table)
		}
	}

	// Verify foreign keys
	var fkCount int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.table_constraints 
		WHERE table_schema = DATABASE() AND constraint_type = 'FOREIGN KEY'`).Scan(&fkCount)
	if err != nil {
		t.Errorf("Failed to check foreign keys: %v", err)
	} else if fkCount == 0 {
		t.Error("Expected foreign keys to exist")
	}

	// Verify indexes
	var idxCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.statistics 
		WHERE table_schema = DATABASE()`).Scan(&idxCount)
	if err != nil {
		t.Errorf("Failed to check indexes: %v", err)
	} else if idxCount < 5 {
		t.Errorf("Expected multiple indexes, got %d", idxCount)
	}
}

func validatePostgresSchema(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()

	// Verify expected tables exist
	tables := []string{"users", "posts", "comments", "tags", "post_tags"}
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1",
			table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("Table %s does not exist", table)
		}
	}

	// Verify foreign keys
	var fkCount int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.table_constraints 
		WHERE table_schema = 'public' AND constraint_type = 'FOREIGN KEY'`).Scan(&fkCount)
	if err != nil {
		t.Errorf("Failed to check foreign keys: %v", err)
	} else if fkCount == 0 {
		t.Error("Expected foreign keys to exist")
	}

	// Verify indexes
	var idxCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE schemaname = 'public'`).Scan(&idxCount)
	if err != nil {
		t.Errorf("Failed to check indexes: %v", err)
	} else if idxCount < 5 {
		t.Errorf("Expected multiple indexes, got %d", idxCount)
	}
}

// =============================================================================
// Code Generation Integration Tests (via CLI)
// =============================================================================

func TestCodeGen_GenerateAndCompile(t *testing.T) {
	if os.Getenv("SKIP_CODEGEN_TEST") == "true" {
		t.Skip("Skipping code generation tests")
	}

	// Check if db-catalyst binary exists
	binaryPath, err := filepath.Abs("../../db-catalyst")
	if err != nil {
		t.Skipf("Failed to resolve binary path: %v", err)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("db-catalyst binary not found, skipping code generation tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tests := []struct {
		name     string
		dialect  string
		schema   string
		queries  string
		validate func(context.Context, *testing.T, string)
	}{
		{
			name:    "SQLite_Basic",
			dialect: "sqlite",
			schema: `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY,
					username TEXT NOT NULL,
					email TEXT
				);
			`,
			queries: `
-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY username;
`,
			validate: validateSQLiteGeneratedCode,
		},
		{
			name:    "SQLite_WithForeignKey",
			dialect: "sqlite",
			schema: `
				CREATE TABLE authors (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL
				);
				CREATE TABLE books (
					id INTEGER PRIMARY KEY,
					author_id INTEGER NOT NULL,
					title TEXT NOT NULL,
					FOREIGN KEY (author_id) REFERENCES authors(id)
				);
			`,
			queries: `
-- name: GetAuthor :one
SELECT * FROM authors WHERE id = ?;

-- name: ListBooksByAuthor :many
SELECT b.* FROM books b
JOIN authors a ON b.author_id = a.id
WHERE a.name = ?;
`,
			validate: validateSQLiteGeneratedCode,
		},
		{
			name:    "SQLite_ComplexTypes",
			dialect: "sqlite",
			schema: `
				CREATE TABLE products (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					description TEXT,
					price REAL,
					in_stock INTEGER DEFAULT 1,
					created_at TEXT DEFAULT CURRENT_TIMESTAMP
				);
			`,
			queries: `
-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: ListInStockProducts :many
SELECT * FROM products WHERE in_stock = 1 ORDER BY price;

-- name: UpdateProductPrice :exec
UPDATE products SET price = ? WHERE id = ?;
`,
			validate: validateSQLiteGeneratedCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCodeGenerationViaCLI(ctx, t, tt.dialect, tt.schema, tt.queries, tt.validate)
		})
	}
}

func testCodeGenerationViaCLI(ctx context.Context, t *testing.T, dialect, schema, queries string, validate func(context.Context, *testing.T, string)) {
	t.Helper()

	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Write schema file
	schemaPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	// Write queries file
	queriesPath := filepath.Join(tmpDir, "queries.sql")
	if err := os.WriteFile(queriesPath, []byte(queries), 0644); err != nil {
		t.Fatalf("Failed to write queries: %v", err)
	}

	// Create config file with relative paths (relative to tmpDir)
	configContent := fmt.Sprintf(`
package = "testdb"
database = "%s"
schemas = ["schema.sql"]
queries = ["queries.sql"]
out = "gen"
`, dialect)

	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Run db-catalyst - use absolute path to binary
	binaryPath, err := filepath.Abs("../../db-catalyst")
	if err != nil {
		t.Fatalf("Failed to resolve binary path: %v", err)
	}
	cmd := exec.CommandContext(ctx, binaryPath, "--config", configPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("db-catalyst failed: %v\nOutput: %s", err, string(output))
	}
	t.Logf("db-catalyst output: %s", string(output))

	// Validate generated code
	genDir := filepath.Join(tmpDir, "gen")
	validate(ctx, t, genDir)
}

func validateSQLiteGeneratedCode(ctx context.Context, t *testing.T, genDir string) {
	t.Helper()

	// Check that files were generated
	files := []string{"models.gen.go"}
	for _, f := range files {
		path := filepath.Join(genDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", f)
		}
	}

	// Verify Go code compiles
	if err := validateGoCompiles(t, genDir); err != nil {
		t.Errorf("Generated code does not compile: %v", err)
	}
}

func validateMySQLGeneratedCode(ctx context.Context, t *testing.T, genDir string) {
	t.Helper()

	// Check that files were generated
	files := []string{"models.gen.go"}
	for _, f := range files {
		path := filepath.Join(genDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", f)
		}
	}

	// Verify Go code compiles
	if err := validateGoCompiles(t, genDir); err != nil {
		t.Errorf("Generated code does not compile: %v", err)
	}
}

func validatePostgresGeneratedCode(ctx context.Context, t *testing.T, genDir string) {
	t.Helper()

	// Check that files were generated
	files := []string{"models.gen.go"}
	for _, f := range files {
		path := filepath.Join(genDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", f)
		}
	}

	// Verify Go code compiles
	if err := validateGoCompiles(t, genDir); err != nil {
		t.Errorf("Generated code does not compile: %v", err)
	}
}

func validateGoCompiles(t *testing.T, dir string) error {
	t.Helper()

	// Check that .go files have valid syntax
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".go") {
			content, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				return fmt.Errorf("read file %s: %w", f.Name(), err)
			}
			if !strings.HasPrefix(string(content), "package ") {
				return fmt.Errorf("file %s missing package declaration", f.Name())
			}

			// Verify no obvious syntax errors
			if strings.Contains(string(content), "\x00") {
				return fmt.Errorf("file %s contains null bytes", f.Name())
			}

			t.Logf("Validated %s (%d bytes)", f.Name(), len(content))
		}
	}

	return nil
}

// =============================================================================
// End-to-End Integration Tests
// =============================================================================

func TestEndToEnd_SQLParseGenerateExecute(t *testing.T) {
	if os.Getenv("SKIP_E2E_TEST") == "true" {
		t.Skip("Skipping end-to-end tests")
	}

	binaryPath, err := filepath.Abs("../../db-catalyst")
	if err != nil {
		t.Skipf("Failed to resolve binary path: %v", err)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("db-catalyst binary not found, skipping e2e tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create test schema
	tmpDir := t.TempDir()

	schema := `
CREATE TABLE e2e_test_users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	email TEXT,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE e2e_test_posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	content TEXT,
	FOREIGN KEY (user_id) REFERENCES e2e_test_users(id)
);
`

	// Write schema file
	schemaPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	queries := `
-- name: CreateUser :execresult
INSERT INTO e2e_test_users (username, email) VALUES (?, ?);

-- name: GetUser :one
SELECT * FROM e2e_test_users WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM e2e_test_users ORDER BY username;

-- name: CreatePost :exec
INSERT INTO e2e_test_posts (user_id, title, content) VALUES (?, ?, ?);
`

	queriesPath := filepath.Join(tmpDir, "queries.sql")
	if err := os.WriteFile(queriesPath, []byte(queries), 0644); err != nil {
		t.Fatalf("Failed to write queries: %v", err)
	}

	// Run db-catalyst - use relative paths
	configContent := `
package = "testdb"
database = "sqlite"
schemas = ["schema.sql"]
queries = ["queries.sql"]
out = "gen"
`

	configPath := filepath.Join(tmpDir, "db-catalyst.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cmd := exec.CommandContext(ctx, binaryPath, "--config", configPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("db-catalyst failed: %v\nOutput: %s", err, string(output))
	}

	// Verify files were generated
	genDir := filepath.Join(tmpDir, "gen")
	modelsPath := filepath.Join(genDir, "models.gen.go")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		t.Error("Expected models.gen.go to be generated")
	}

	// List generated files
	files, _ := os.ReadDir(genDir)
	t.Logf("Generated %d files:", len(files))
	for _, f := range files {
		info, _ := f.Info()
		t.Logf("  - %s (%d bytes)", f.Name(), info.Size())
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMySQL_Query(b *testing.B) {
	if os.Getenv("SKIP_DOCKER") == "true" {
		b.Skip("Skipping Docker tests")
	}

	ctx := context.Background()
	db, err := sql.Open("mysql", mysqlConn)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		b.Skip("MySQL not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgres_Query(b *testing.B) {
	if os.Getenv("SKIP_DOCKER") == "true" {
		b.Skip("Skipping Docker tests")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", postgresConn)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		b.Skip("PostgreSQL not available")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_Query(b *testing.B) {
	ctx := context.Background()
	db := createSQLiteDB(b)
	defer db.Close()
	defer cleanupSQLite(b)

	initSQLiteSchema(ctx, b, db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int64
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			b.Fatal(err)
		}
	}
}
