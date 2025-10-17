package render

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/printer"
	"go/token"

	"golang.org/x/tools/imports"
)

// Spec describes an AST file to render.
type Spec struct {
	Path string
	Node *goast.File
	Raw  []byte
}

// File contains the rendered Go source for a path.
type File struct {
	Path    string
	Content []byte
}

// Format renders all provided AST files using go/printer and goimports.
func Format(specs []Spec) ([]File, error) {
	rendered := make([]File, 0, len(specs))
	for _, spec := range specs {
		if len(spec.Raw) > 0 {
			rawCopy := append([]byte(nil), spec.Raw...)
			rendered = append(rendered, File{Path: spec.Path, Content: rawCopy})
			continue
		}
		if spec.Node == nil {
			return nil, fmt.Errorf("render %s: nil AST node", spec.Path)
		}
		fset := token.NewFileSet()
		var buf bytes.Buffer
		cfg := &printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}
		if err := cfg.Fprint(&buf, fset, spec.Node); err != nil {
			return nil, fmt.Errorf("render %s: %w", spec.Path, err)
		}
		formatted, err := imports.Process("", buf.Bytes(), nil)
		if err != nil {
			return nil, fmt.Errorf("goimports %s: %w", spec.Path, err)
		}
		rendered = append(rendered, File{Path: spec.Path, Content: formatted})
	}
	return rendered, nil
}
