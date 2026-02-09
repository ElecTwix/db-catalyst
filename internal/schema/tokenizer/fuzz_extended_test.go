package tokenizer

import (
	"testing"
)

// FuzzScanExtended tests tokenizer with comprehensive inputs.
func FuzzScanExtended(f *testing.F) {
	// Valid SQL tokens
	f.Add("SELECT * FROM users;")
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY);")
	f.Add("INSERT INTO t VALUES (1, 'test', 3.14);")
	f.Add("-- single line comment\nSELECT 1;")
	f.Add("/* block comment */ SELECT 2;")
	f.Add("/* multi\nline\ncomment */ SELECT 3;")
	f.Add("SELECT 'string' FROM t;")
	f.Add("SELECT \"quoted identifier\" FROM t;")
	f.Add("SELECT `backtick` FROM t;")
	// Edge cases
	f.Add("")
	f.Add(";")
	f.Add("   ") // Whitespace only
	f.Add("--")
	f.Add("/*")
	f.Add("/**/")
	f.Add("'")
	f.Add("''")
	f.Add("'unclosed")
	f.Add("\"")
	f.Add("`")
	// Special characters
	f.Add("!@#$%^&*()")
	f.Add("+-*/=<>")
	f.Add("[]{}|\\")
	f.Add("::")
	f.Add("||")
	f.Add(":=")
	f.Add("->")
	f.Add("->>")
	// Unicode
	f.Add("SELECT '日本語' FROM t;")
	f.Add("SELECT 用户 FROM t;")
	f.Add("/* 注释 */ SELECT 1;")
	f.Add("-- 注释\nSELECT 1;")
	// Binary and hex literals
	f.Add("SELECT x'ABCD';")
	f.Add("SELECT X'1234';")
	f.Add("SELECT 0x1234;")
	// Numbers
	f.Add("SELECT 1, 1.5, 1e10, 1.5e-10;")
	f.Add("SELECT .5, 5., +5, -5;")
	// Escapes
	f.Add("SELECT 'it''s';") // Single quote escape
	f.Add(`SELECT 'line1\nline2';`)
	// Very long inputs
	f.Add(string(make([]byte, 10000)))
	f.Add(string(make([]byte, 100000)))
	// Mixed valid/invalid
	f.Add("SELECT \x00 FROM t;") // Null byte
	f.Add("SELECT \xff FROM t;") // Invalid UTF-8

	f.Fuzz(func(t *testing.T, input string) {
		// Tokenizer should never panic
		_, _ = Scan("fuzz", []byte(input), true)
	})
}

// FuzzScanWithOptions tests tokenizer with different options.
func FuzzScanWithOptions(f *testing.F) {
	f.Add("SELECT * FROM users;", true)
	f.Add("SELECT * FROM users;", false)
	f.Add("-- comment\nSELECT 1;", true)
	f.Add("-- comment\nSELECT 1;", false)

	f.Fuzz(func(t *testing.T, input string, includeComments bool) {
		_, _ = Scan("fuzz", []byte(input), includeComments)
	})
}
