package sqlfix

import (
	"fmt"
	"os"

	"github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	schematokenizer "github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

type SchemaCatalog struct {
	Catalog  *model.Catalog
	Warnings []string
}

func LoadSchemaCatalog(paths []string, readFile func(string) ([]byte, error)) (SchemaCatalog, error) {
	result := SchemaCatalog{Catalog: model.NewCatalog()}
	if len(paths) == 0 {
		return result, nil
	}
	reader := readFile
	if reader == nil {
		reader = os.ReadFile
	}
	for _, path := range paths {
		contents, err := reader(path)
		if err != nil {
			return result, fmt.Errorf("read schema %s: %w", path, err)
		}

		tokens, err := schematokenizer.Scan(path, contents, true)
		if err != nil {
			return result, fmt.Errorf("scan schema %s: %w", path, err)
		}

		catalog, diags, err := schemaparser.Parse(path, tokens)
		for _, diag := range diags {
			result.Warnings = append(result.Warnings, formatSchemaDiagnostic(diag))
		}
		if err != nil {
			return result, fmt.Errorf("parse schema %s: %w", path, err)
		}

		mergeSchemaCatalog(result.Catalog, catalog, &result.Warnings)
	}
	return result, nil
}

func formatSchemaDiagnostic(d schemaparser.Diagnostic) string {
	return fmt.Sprintf("%s:%d:%d: %s", d.Path, d.Line, d.Column, d.Message)
}

func mergeSchemaCatalog(dest, src *model.Catalog, warnings *[]string) {
	if dest == nil || src == nil {
		return
	}
	if dest.Tables == nil {
		dest.Tables = make(map[string]*model.Table)
	}
	for key, table := range src.Tables {
		if existing, ok := dest.Tables[key]; ok {
			message := fmt.Sprintf("%s:%d:%d: duplicate table %q (previous definition at %s:%d:%d)", table.Span.File, table.Span.StartLine, table.Span.StartColumn, table.Name, existing.Span.File, existing.Span.StartLine, existing.Span.StartColumn)
			*warnings = append(*warnings, message)
			continue
		}
		dest.Tables[key] = table
	}
	if dest.Views == nil {
		dest.Views = make(map[string]*model.View)
	}
	for key, view := range src.Views {
		if existing, ok := dest.Views[key]; ok {
			message := fmt.Sprintf("%s:%d:%d: duplicate view %q (previous definition at %s:%d:%d)", view.Span.File, view.Span.StartLine, view.Span.StartColumn, view.Name, existing.Span.File, existing.Span.StartLine, existing.Span.StartColumn)
			*warnings = append(*warnings, message)
			continue
		}
		dest.Views[key] = view
	}
}
