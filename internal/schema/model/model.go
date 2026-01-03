// Package model defines normalized schema catalog types produced by the parser.
package model

import (
	"cmp"
	"slices"
	"strings"

	"github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Catalog represents the collection of tables and views discovered in DDL files.
type Catalog struct {
	Tables map[string]*Table
	Views  map[string]*View
}

// NewCatalog constructs a catalog with initialized maps.
func NewCatalog() *Catalog {
	return &Catalog{
		Tables: make(map[string]*Table),
		Views:  make(map[string]*View),
	}
}

// Table models a SQLite table definition with associated constraints.
type Table struct {
	Name         string
	Doc          string
	Columns      []*Column
	PrimaryKey   *PrimaryKey
	UniqueKeys   []*UniqueKey
	ForeignKeys  []*ForeignKey
	Indexes      []*Index
	WithoutRowID bool
	Strict       bool
	Span         tokenizer.Span
}

// Column describes a table column with optional inline constraints.
type Column struct {
	Name       string
	Type       string
	NotNull    bool
	Default    *Value
	References *ForeignKeyRef
	Span       tokenizer.Span
}

// PrimaryKey captures a table's primary key declaration.
type PrimaryKey struct {
	Name    string
	Columns []string
	Span    tokenizer.Span
}

// UniqueKey captures a UNIQUE constraint across one or more columns.
type UniqueKey struct {
	Name    string
	Columns []string
	Span    tokenizer.Span
}

// ForeignKey models a FOREIGN KEY constraint referencing another table.
type ForeignKey struct {
	Name    string
	Columns []string
	Ref     ForeignKeyRef
	Span    tokenizer.Span
}

// ForeignKeyRef describes the referenced table and column set for a foreign key.
type ForeignKeyRef struct {
	Table   string
	Columns []string
	Span    tokenizer.Span
}

// Index describes a CREATE INDEX or CREATE UNIQUE INDEX statement targeting a table.
type Index struct {
	Name    string
	Unique  bool
	Columns []string
	Span    tokenizer.Span
}

// View represents a CREATE VIEW statement along with the raw SQL body.
type View struct {
	Name string
	Doc  string
	SQL  string
	Span tokenizer.Span
}

// ValueKind identifies the literal kind stored in a Value.
type ValueKind int

const (
	// ValueKindUnknown is used when the literal kind cannot be determined.
	ValueKindUnknown ValueKind = iota
	// ValueKindNumber represents numeric literals.
	ValueKindNumber
	// ValueKindString represents single-quoted string literals.
	ValueKindString
	// ValueKindBlob represents blob literals of the form X'...'.
	ValueKindBlob
	// ValueKindKeyword represents keywords used as literal defaults (e.g. CURRENT_TIMESTAMP).
	ValueKindKeyword
)

// Value stores the raw literal used in expressions such as DEFAULT clauses.
type Value struct {
	Kind ValueKind
	Text string
	Span tokenizer.Span
}

// SortColumns provides deterministic ordering of columns by name.
func SortColumns(cols []*Column) {
	slices.SortFunc(cols, func(a, b *Column) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

// SortUniqueKeys provides deterministic ordering of unique constraints by name then columns.
func SortUniqueKeys(keys []*UniqueKey) {
	slices.SortFunc(keys, func(a, b *UniqueKey) int {
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(joinColumns(a.Columns), joinColumns(b.Columns))
	})
}

// SortForeignKeys provides deterministic ordering of foreign key constraints by name then reference.
func SortForeignKeys(keys []*ForeignKey) {
	slices.SortFunc(keys, func(a, b *ForeignKey) int {
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		if c := cmp.Compare(joinColumns(a.Columns), joinColumns(b.Columns)); c != 0 {
			return c
		}
		return cmp.Compare(a.Ref.Table, b.Ref.Table)
	})
}

// SortIndexes provides deterministic ordering of indexes by name.
func SortIndexes(idxs []*Index) {
	slices.SortFunc(idxs, func(a, b *Index) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

func joinColumns(cols []string) string {
	return strings.Join(cols, "\x00")
}
