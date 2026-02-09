package parser

import (
	"context"
	"testing"
)

// FuzzParser tests the high-level parser with random inputs.
func FuzzParser(f *testing.F) {
	// Seed corpus with valid SQL
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY);")
	f.Add("CREATE TABLE users (id INTEGER, name TEXT NOT NULL UNIQUE);")
	f.Add("CREATE INDEX idx ON users (name);")
	f.Add("CREATE VIEW v AS SELECT * FROM users;")
	f.Add("ALTER TABLE users ADD COLUMN email TEXT;")
	f.Add("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL CHECK(price > 0));")
	f.Add(`CREATE TABLE orders (
		id INTEGER PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		created_at TEXT DEFAULT CURRENT_TIMESTAMP
	);`)
	// Edge cases
	f.Add("")                    // Empty
	f.Add(";")                   // Just semicolon
	f.Add("-- just a comment")   // Only comment
	f.Add("/* block comment */") // Block comment only
	f.Add("CREATE")              // Incomplete statement
	f.Add("CREATE TABLE")        // Missing table name
	f.Add("CREATE TABLE t ()")   // Empty columns
	f.Add("SELECT * FROM t")     // Non-DDL
	f.Add("DROP TABLE t")        // Unsupported
	// Unicode edge cases
	f.Add("CREATE TABLE 用户 (id INTEGER);")      // Unicode table name
	f.Add("CREATE TABLE t (名称 TEXT);")          // Unicode column name
	f.Add("CREATE TABLE t (id INTEGER); -- 注释") // Unicode comment
	// Malformed inputs
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY;") // Missing paren
	f.Add("CREATE TABLE t id INTEGER);")             // Missing opening paren
	f.Add("CREATE TABLE t (id INTEGER,);")           // Trailing comma

	f.Fuzz(func(t *testing.T, input string) {
		p := NewParser()
		// Parser should never panic
		_, _ = p.Parse(context.Background(), input)
	})
}

// FuzzParserWithDebug tests parser in debug mode.
func FuzzParserWithDebug(f *testing.F) {
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY);", true)
	f.Add("CREATE TABLE t (id INTEGER);", false)

	f.Fuzz(func(t *testing.T, input string, debug bool) {
		p := NewParser(WithDebug(debug))
		_, _ = p.Parse(context.Background(), input)
	})
}

// FuzzParserWithMaxErrors tests parser with different max error limits.
func FuzzParserWithMaxErrors(f *testing.F) {
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY);", 1)
	f.Add("CREATE TABLE t (id INTEGER);", 10)
	f.Add("CREATE TABLE t (id INTEGER);", 100)
	f.Add("CREATE TABLE t (id INTEGER);", 0)
	f.Add("CREATE TABLE t (id INTEGER);", -1)

	f.Fuzz(func(t *testing.T, input string, maxErrors int) {
		p := NewParser(WithMaxErrors(maxErrors))
		_, _ = p.Parse(context.Background(), input)
	})
}
