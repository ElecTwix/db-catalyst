package tokenizer

import (
	"slices"
	"testing"
)

func TestScanSeq(t *testing.T) {
	sql := `CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT
	);`

	// Collect tokens from iterator
	var tokens []Token
	for tok := range ScanSeq("test.sql", []byte(sql), false) {
		tokens = append(tokens, tok)
		if tok.Kind == KindEOF {
			break
		}
	}

	// Compare with Scan
	expected, err := Scan("test.sql", []byte(sql), false)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(tokens) != len(expected) {
		t.Errorf("ScanSeq returned %d tokens, Scan returned %d", len(tokens), len(expected))
	}

	for i := range min(len(tokens), len(expected)) {
		if tokens[i].Kind != expected[i].Kind {
			t.Errorf("token %d: Kind = %v, want %v", i, tokens[i].Kind, expected[i].Kind)
		}
		if tokens[i].Text != expected[i].Text {
			t.Errorf("token %d: Text = %q, want %q", i, tokens[i].Text, expected[i].Text)
		}
	}
}

func TestScanSeqEarlyTermination(t *testing.T) {
	sql := `CREATE TABLE users (id INTEGER);`

	// Only collect first 3 tokens
	var tokens []Token
	for tok := range ScanSeq("test.sql", []byte(sql), false) {
		tokens = append(tokens, tok)
		if len(tokens) >= 3 {
			break
		}
	}

	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
	}
}

func TestScanSeqLargeFile(t *testing.T) {
	// Create a large SQL file (1000 CREATE TABLE statements)
	var sql string
	for i := 0; i < 1000; i++ {
		sql += "CREATE TABLE t" + string(rune('0'+i%10)) + " (id INTEGER);\n"
	}

	// ScanSeq should handle this without loading all tokens at once
	count := 0
	for range ScanSeq("test.sql", []byte(sql), false) {
		count++
	}

	// Should have many tokens
	if count < 1000 {
		t.Errorf("expected at least 1000 tokens, got %d", count)
	}
}

func TestScanSeqEmpty(t *testing.T) {
	tokens := slices.Collect(ScanSeq("test.sql", []byte(""), false))

	// Should have at least EOF
	if len(tokens) == 0 {
		t.Error("expected at least EOF token")
	}

	last := tokens[len(tokens)-1]
	if last.Kind != KindEOF {
		t.Errorf("expected EOF, got %v", last.Kind)
	}
}

func TestScanSeqInvalidUTF8(t *testing.T) {
	// Invalid UTF-8 should return empty iterator
	tokens := slices.Collect(ScanSeq("test.sql", []byte{0xff, 0xfe}, false))

	if len(tokens) > 0 {
		t.Errorf("expected no tokens for invalid UTF-8, got %d", len(tokens))
	}
}

func BenchmarkScanSeqIterator(b *testing.B) {
	sql := []byte(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)

	b.ResetTimer()
	for range b.N {
		for range ScanSeq("test.sql", sql, false) {
			// Just consume
		}
	}
}

func BenchmarkScanSlice(b *testing.B) {
	sql := []byte(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)

	b.ResetTimer()
	for range b.N {
		_, _ = Scan("test.sql", sql, false)
	}
}
