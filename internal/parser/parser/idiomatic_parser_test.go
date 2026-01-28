package parser_test

import (
	"testing"

	"github.com/electwix/db-catalyst/internal/parser"
)

func TestNewParser(t *testing.T) {
	t.Run("returns parser", func(t *testing.T) {
		p := parser.NewParser()
		if p == nil {
			t.Error("NewParser() returned nil")
		}
	})

	t.Run("with debug enabled", func(t *testing.T) {
		p := parser.NewParser(parser.WithDebug(true))
		if p == nil {
			t.Error("NewParser(WithDebug(true)) returned nil")
		}
	})

	t.Run("with max errors", func(t *testing.T) {
		p := parser.NewParser(parser.WithMaxErrors(5))
		if p == nil {
			t.Error("NewParser(WithMaxErrors(5)) returned nil")
		}
	})
}
