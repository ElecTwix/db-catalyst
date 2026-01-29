package codegen

import "github.com/electwix/db-catalyst/internal/codegen/ast"

// ExportedIdentifier converts raw input into a public Go identifier.
func ExportedIdentifier(raw string) string { return ast.ExportedIdentifier(raw) }

// UnexportedIdentifier converts raw input into a private Go identifier.
func UnexportedIdentifier(raw string) string { return ast.UnexportedIdentifier(raw) }

// FileName converts the raw name into a snake_case file name segment.
func FileName(raw string) string { return ast.FileName(raw) }

// UniqueName ensures returned identifier does not collide with previous values.
func UniqueName(base string, used map[string]int) (string, error) { return ast.UniqueName(base, used) }
