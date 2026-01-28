package tokenizer

import (
	"testing"
)

func BenchmarkScan_Simple(b *testing.B) {
	input := []byte("CREATE TABLE users (id INTEGER PRIMARY KEY);")

	b.ResetTimer()
	for b.Loop() {
		_, _ = Scan("bench.sql", input, true)
	}
}

func BenchmarkScan_Complex(b *testing.B) {
	input := []byte(`
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users (email);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT,
    published BOOLEAN DEFAULT FALSE
);
`)

	b.ResetTimer()
	for b.Loop() {
		_, _ = Scan("bench.sql", input, true)
	}
}
