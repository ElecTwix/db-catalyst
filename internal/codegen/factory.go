package codegen

import (
	"context"
	"fmt"

	"github.com/electwix/db-catalyst/internal/codegen/rust"
	"github.com/electwix/db-catalyst/internal/codegen/typescript"
	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// GeneratorFactory creates language-specific generators.
type GeneratorFactory struct {
	opts Options
}

// NewGeneratorFactory creates a new generator factory.
func NewGeneratorFactory(opts Options) *GeneratorFactory {
	return &GeneratorFactory{opts: opts}
}

// Create returns a generator for the specified language.
func (f *GeneratorFactory) Create(lang config.Language) (Generator, error) {
	switch lang {
	case config.LanguageGo, "": // Default to Go
		return New(f.opts), nil
	case config.LanguageRust:
		return f.createRustGenerator()
	case config.LanguageTypeScript:
		return f.createTypeScriptGenerator()
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// createRustGenerator creates a Rust generator wrapper.
func (f *GeneratorFactory) createRustGenerator() (Generator, error) {
	gen, err := rust.NewGenerator()
	if err != nil {
		return nil, fmt.Errorf("create rust generator: %w", err)
	}

	// Wrap the rust generator to match the interface
	return &rustGeneratorWrapper{gen: gen}, nil
}

// createTypeScriptGenerator creates a TypeScript generator wrapper.
func (f *GeneratorFactory) createTypeScriptGenerator() (Generator, error) {
	gen, err := typescript.NewGenerator()
	if err != nil {
		return nil, fmt.Errorf("create typescript generator: %w", err)
	}

	// Wrap the typescript generator to match the interface
	return &typescriptGeneratorWrapper{gen: gen}, nil
}

// rustGeneratorWrapper wraps the rust generator to match the Generator interface.
type rustGeneratorWrapper struct {
	gen *rust.Generator
}

// Generate generates Rust code.
func (w *rustGeneratorWrapper) Generate(_ context.Context, catalog *model.Catalog, _ []analyzer.Result) ([]File, error) {
	// Convert catalog tables map to slice
	tables := make([]*model.Table, 0, len(catalog.Tables))
	for _, table := range catalog.Tables {
		tables = append(tables, table)
	}

	// Convert catalog tables to rust table models
	files, err := w.gen.GenerateModels(tables)
	if err != nil {
		return nil, fmt.Errorf("generate rust models: %w", err)
	}

	// Convert rust files to codegen files
	result := make([]File, len(files))
	for i, f := range files {
		result[i] = File{Path: f.Path, Content: f.Content}
	}

	return result, nil
}

// typescriptGeneratorWrapper wraps the typescript generator to match the Generator interface.
type typescriptGeneratorWrapper struct {
	gen *typescript.Generator
}

// Generate generates TypeScript code.
func (w *typescriptGeneratorWrapper) Generate(_ context.Context, catalog *model.Catalog, _ []analyzer.Result) ([]File, error) {
	// Convert catalog tables map to slice
	tables := make([]*model.Table, 0, len(catalog.Tables))
	for _, table := range catalog.Tables {
		tables = append(tables, table)
	}

	// Convert catalog tables to typescript table models
	files, err := w.gen.GenerateModels(tables)
	if err != nil {
		return nil, fmt.Errorf("generate typescript models: %w", err)
	}

	// Convert typescript files to codegen files
	result := make([]File, len(files))
	for i, f := range files {
		result[i] = File{Path: f.Path, Content: f.Content}
	}

	return result, nil
}
