package postgres

import (
	"context"
	"testing"
)

// FuzzParser tests PostgreSQL parser with random inputs.
func FuzzParser(f *testing.F) {
	// Seed corpus with valid PostgreSQL DDL
	f.Add("CREATE TABLE t (id SERIAL PRIMARY KEY);")
	f.Add("CREATE TABLE users (id BIGSERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL);")
	f.Add("CREATE TABLE products (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), data JSONB);")
	f.Add("CREATE TABLE items (tags TEXT[], counts INTEGER[]);")
	f.Add("CREATE TYPE status AS ENUM ('active', 'inactive');")
	f.Add("CREATE TABLE orders (id SERIAL PRIMARY KEY, state status DEFAULT 'active');")
	f.Add("CREATE DOMAIN email AS TEXT CHECK (VALUE ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$');")
	f.Add("CREATE TABLE users (id SERIAL PRIMARY KEY, email_addr email NOT NULL);")
	f.Add("CREATE INDEX idx ON users USING GIN (data);")
	f.Add("CREATE TABLE posts (id SERIAL PRIMARY KEY, title TEXT, tsvector TSVECTOR);")
	f.Add("CREATE TABLE events (id BIGSERIAL, created_at TIMESTAMPTZ DEFAULT NOW(), amount NUMERIC(10,2));")
	// Edge cases
	f.Add("")
	f.Add("-- comment")
	f.Add("/* comment */")
	f.Add("CREATE TABLE")
	f.Add("CREATE TYPE")
	f.Add("CREATE DOMAIN")
	// Unicode
	f.Add("CREATE TABLE ユーザー (id SERIAL PRIMARY KEY);")
	f.Add("CREATE TABLE users (名前 VARCHAR(100));")
	// Malformed
	f.Add("CREATE TABLE t (id SERIAL PRIMARY KEY;")
	f.Add("CREATE TYPE status AS ENUM ('a',)")
	f.Add("CREATE DOMAIN email AS TEXT CHECK (")

	f.Fuzz(func(t *testing.T, input string) {
		p := New()
		// Parser should never panic
		_, _, _ = p.Parse(context.Background(), "fuzz.sql", []byte(input))
	})
}

// FuzzParserWithComplexTypes tests complex PostgreSQL type scenarios.
func FuzzParserWithComplexTypes(f *testing.F) {
	f.Add("CREATE TABLE t (arr TEXT[][]);")
	f.Add("CREATE TABLE t (nested INTEGER[][][]);")
	f.Add("CREATE TABLE t (range INT4RANGE);")
	f.Add("CREATE TABLE t (cidr CIDR, inet INET);")
	f.Add("CREATE TABLE t (bits BIT(8), varbits BIT VARYING(16));")
	f.Add("CREATE TABLE t (geo POINT, line LINE, box BOX);")
	f.Add("CREATE TABLE t (xml_col XML, json_col JSON);")
	f.Add("CREATE TABLE t (ltree_col LTREE, lquery_col LQUERY);")

	f.Fuzz(func(t *testing.T, input string) {
		p := New()
		_, _, _ = p.Parse(context.Background(), "fuzz.sql", []byte(input))
	})
}

// FuzzParserWithConstraints tests various constraint combinations.
func FuzzParserWithConstraints(f *testing.F) {
	f.Add("CREATE TABLE t (id SERIAL PRIMARY KEY);")
	f.Add("CREATE TABLE t (id SERIAL, CONSTRAINT pk PRIMARY KEY (id));")
	f.Add("CREATE TABLE t (a INTEGER, b INTEGER, CONSTRAINT u UNIQUE (a, b));")
	f.Add("CREATE TABLE t (a INTEGER REFERENCES users(id));")
	f.Add("CREATE TABLE t (a INTEGER REFERENCES users(id) ON DELETE CASCADE);")
	f.Add("CREATE TABLE t (a INTEGER REFERENCES users(id) ON UPDATE SET NULL ON DELETE CASCADE);")
	f.Add("CREATE TABLE t (a INTEGER CHECK (a > 0));")
	f.Add("CREATE TABLE t (a INTEGER, CONSTRAINT chk CHECK (a > 0 AND a < 100));")
	f.Add("CREATE TABLE t (a INTEGER NOT NULL DEFAULT 0);")
	f.Add("CREATE TABLE t (a INTEGER UNIQUE DEFERRABLE INITIALLY DEFERRED);")

	f.Fuzz(func(t *testing.T, input string) {
		p := New()
		_, _, _ = p.Parse(context.Background(), "fuzz.sql", []byte(input))
	})
}
