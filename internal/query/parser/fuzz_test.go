package parser

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/query/block"
)

// FuzzParse tests the query parser with random inputs.
func FuzzParse(f *testing.F) {
	f.Add("-- name: GetUser :one\nSELECT * FROM users WHERE id = :id;")
	f.Add("-- name: ListUsers :many\nSELECT * FROM users;")
	f.Add("-- name: CreateUser :exec\nINSERT INTO users (name) VALUES (:name);")

	f.Fuzz(func(_ *testing.T, input string) {
		blk := block.Block{
			Path:   "fuzz.sql",
			Line:   1,
			Column: 1,
			SQL:    input,
		}
		_, _ = Parse(blk)
		// Should never panic
	})
}
