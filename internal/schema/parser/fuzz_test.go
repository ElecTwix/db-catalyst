package parser

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// FuzzParse tests the parser with random inputs.
func FuzzParse(f *testing.F) {
	// Seed corpus with valid DDL
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY);")
	f.Add("CREATE TABLE users (id INTEGER, name TEXT NOT NULL);")
	f.Add("CREATE INDEX idx ON users (name);")
	f.Add("CREATE VIEW v AS SELECT * FROM users;")
	f.Add("ALTER TABLE users ADD COLUMN email TEXT;")

	f.Fuzz(func(t *testing.T, input string) {
		tokens, _ := tokenizer.Scan("fuzz", []byte(input), true)
		_, _, _ = Parse("fuzz", tokens)
		// Parser should never panic
	})
}
