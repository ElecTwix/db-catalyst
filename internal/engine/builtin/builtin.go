// Package builtin registers all built-in database engines.
//
// Import this package to register the SQLite, PostgreSQL, and MySQL engines:
//
//	import _ "github.com/electwix/db-catalyst/internal/engine/builtin"
//
// This will make the engines available via engine.New().
package builtin

import (
	"errors"

	"github.com/electwix/db-catalyst/internal/engine"
	"github.com/electwix/db-catalyst/internal/engine/postgres"
	"github.com/electwix/db-catalyst/internal/engine/sqlite"
)

//nolint:gochecknoinits // Package registration via init is idiomatic for this use case
func init() {
	RegisterAll()
}

// RegisterAll registers all built-in database engines.
// This is called automatically on package import, but can also be called
// manually for testing or custom initialization.
func RegisterAll() {
	engine.Register("sqlite", sqlite.New)
	engine.Register("postgresql", postgres.New)
	engine.Register("postgres", postgres.New) // Alias
	engine.Register("mysql", func(_ engine.Options) (engine.Engine, error) {
		return nil, errors.New("mysql engine not yet implemented")
	})
}
