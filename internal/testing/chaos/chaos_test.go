package chaos_test

import (
	"context"
	"testing"

	"github.com/electwix/db-catalyst/internal/parser/languages/graphql"
	"github.com/electwix/db-catalyst/internal/query/block"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	"github.com/electwix/db-catalyst/internal/schema/parser/mysql"
	"github.com/electwix/db-catalyst/internal/schema/parser/postgres"
	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
	"github.com/electwix/db-catalyst/internal/testing/chaos"
)

// TestTokenizerChaos tests tokenizer with corrupt inputs.
func TestTokenizerChaos(t *testing.T) {
	validInputs := [][]byte{
		[]byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"),
		[]byte("SELECT * FROM users WHERE id = ?;"),
		[]byte("-- comment\nSELECT 1;"),
		[]byte("/* block */ SELECT 2;"),
		[]byte("INSERT INTO t VALUES (1, 'test');"),
	}

	corruptor := chaos.NewCorruptor(42)

	for _, valid := range validInputs {
		// Generate corrupted versions
		corpus := corruptor.GenerateCorpus(valid, 100)

		for _, corrupted := range corpus {
			// Should never panic
			_, _ = tokenizer.Scan("chaos", corrupted, true)
		}
	}
}

// TestSchemaParserChaos tests schema parser with corrupt inputs.
func TestSchemaParserChaos(t *testing.T) {
	validInputs := [][]byte{
		[]byte("CREATE TABLE t (id INTEGER PRIMARY KEY);"),
		[]byte("CREATE TABLE users (id INTEGER, name TEXT);"),
		[]byte("CREATE INDEX idx ON users (name);"),
	}

	corruptor := chaos.NewCorruptor(42)

	for _, valid := range validInputs {
		corpus := corruptor.GenerateCorpus(valid, 50)

		for _, corrupted := range corpus {
			tokens, _ := tokenizer.Scan("chaos", corrupted, true)
			// Should never panic
			_, _, _ = schemaparser.Parse("chaos", tokens)
		}
	}
}

// TestPostgresParserChaos tests PostgreSQL parser with corrupt inputs.
func TestPostgresParserChaos(t *testing.T) {
	validInputs := [][]byte{
		[]byte("CREATE TABLE t (id SERIAL PRIMARY KEY);"),
		[]byte("CREATE TYPE status AS ENUM ('a', 'b');"),
		[]byte("CREATE DOMAIN email AS TEXT CHECK (VALUE ~* '@'));"),
	}

	corruptor := chaos.NewCorruptor(43)
	p := postgres.New()

	for _, valid := range validInputs {
		corpus := corruptor.GenerateCorpus(valid, 50)

		for _, corrupted := range corpus {
			// Should never panic
			_, _, _ = p.Parse(context.Background(), "chaos.sql", corrupted)
		}
	}
}

// TestMySQLParserChaos tests MySQL parser with corrupt inputs.
func TestMySQLParserChaos(t *testing.T) {
	validInputs := [][]byte{
		[]byte("CREATE TABLE t (id INT AUTO_INCREMENT PRIMARY KEY);"),
		[]byte("CREATE TABLE users (id INT, name VARCHAR(100));"),
		[]byte("CREATE TABLE products (status ENUM('a', 'b'));"),
	}

	corruptor := chaos.NewCorruptor(44)
	p := mysql.New()

	for _, valid := range validInputs {
		corpus := corruptor.GenerateCorpus(valid, 50)

		for _, corrupted := range corpus {
			// Should never panic
			_, _, _ = p.Parse(context.Background(), "chaos.sql", corrupted)
		}
	}
}

// TestQueryParserChaos tests query parser with corrupt inputs.
func TestQueryParserChaos(t *testing.T) {
	validInputs := []string{
		"-- name: GetUser :one\nSELECT * FROM users WHERE id = :id;",
		"-- name: ListUsers :many\nSELECT * FROM users;",
		"-- name: CreateUser :exec\nINSERT INTO users (name) VALUES (:name);",
	}

	corruptor := chaos.NewCorruptor(45)

	for _, valid := range validInputs {
		corpus := corruptor.GenerateCorpus([]byte(valid), 50)

		for _, corrupted := range corpus {
			blk := block.Block{
				Path:   "chaos.sql",
				Line:   1,
				Column: 1,
				SQL:    string(corrupted),
			}
			// Should never panic
			_, _ = queryparser.Parse(blk)
		}
	}
}

// TestGraphQLParserChaos tests GraphQL parser with corrupt inputs.
func TestGraphQLParserChaos(t *testing.T) {
	validInputs := [][]byte{
		[]byte("type User { id: ID name: String }"),
		[]byte("type Query { user(id: ID): User }"),
		[]byte("enum Status { ACTIVE INACTIVE }"),
	}

	corruptor := chaos.NewCorruptor(46)
	p, err := graphql.NewParser()
	if err != nil {
		t.Fatalf("Failed to create GraphQL parser: %v", err)
	}

	for _, valid := range validInputs {
		corpus := corruptor.GenerateCorpus(valid, 50)

		for _, corrupted := range corpus {
			// Should never panic
			_, _ = p.ParseSchema(context.Background(), string(corrupted))
		}
	}
}

// TestChaosWithSpecificCorruptions tests specific corruption patterns.
func TestChaosWithSpecificCorruptions(t *testing.T) {
	valid := []byte("CREATE TABLE t (id INTEGER PRIMARY KEY);")
	corruptor := chaos.NewCorruptor(47)

	tests := []struct {
		name      string
		mutation  int
		intensity int
	}{
		{"byte_flip", 0, 5},
		{"byte_delete", 1, 5},
		{"byte_insert", 2, 5},
		{"byte_replace", 3, 5},
		{"utf8_corrupt", 4, 5},
		{"truncation", 5, 3},
		{"bit_inversion", 6, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				corrupted := corruptor.CorruptN(valid, tt.intensity)
				tokens, _ := tokenizer.Scan("chaos", corrupted, true)
				_, _, _ = schemaparser.Parse("chaos", tokens)
			}
		})
	}
}

// BenchmarkChaosCorruption benchmarks the corruption operations.
func BenchmarkChaosCorruption(b *testing.B) {
	valid := []byte("CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT NOT NULL);")
	corruptor := chaos.NewCorruptor(42)

	b.Run("Corrupt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = corruptor.Corrupt(valid)
		}
	})

	b.Run("CorruptN", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = corruptor.CorruptN(valid, 5)
		}
	})

	b.Run("GenerateCorpus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = corruptor.GenerateCorpus(valid, 100)
		}
	})
}
