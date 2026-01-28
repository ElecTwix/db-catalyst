package tokenizer

import (
	"testing"
)

// FuzzScan tests the tokenizer with random inputs.
func FuzzScan(f *testing.F) {
	// Seed corpus with valid SQL
	f.Add("CREATE TABLE users (id INTEGER);")
	f.Add("SELECT * FROM users WHERE id = ?;")
	f.Add("INSERT INTO users (name) VALUES ('test');")
	f.Add("-- comment\nSELECT 1;")
	f.Add("/* block */ SELECT 2;")

	f.Fuzz(func(_ *testing.T, input string) {
		// Tokenizer should never panic
		_, _ = Scan("fuzz", []byte(input), true)
	})
}
