// Package codegen orchestrates the generation of Go code from SQL.
package codegen

import (
	"context"
	"fmt"
	"slices"

	astbuilder "github.com/electwix/db-catalyst/internal/codegen/ast"
	"github.com/electwix/db-catalyst/internal/codegen/render"
	"github.com/electwix/db-catalyst/internal/codegen/sql"
	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/schema/model"
	"github.com/electwix/db-catalyst/internal/transform"
)

// Generator produces Go code from parsed schemas and queries.
type Generator interface {
	Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error)
}

// PreparedOptions configures prepared statement generation.
type PreparedOptions struct {
	Enabled     bool
	EmitMetrics bool
	ThreadSafe  bool
}

// SQLOptions configures SQL schema output.
type SQLOptions struct {
	Enabled         bool
	Dialect         string
	EmitIFNotExists bool
}

// Options configures the Generator.
type Options struct {
	Package             string
	Database            config.Database
	EmitJSONTags        bool
	EmitEmptySlices     bool
	EmitPointersForNull bool
	Prepared            PreparedOptions
	CustomTypes         []config.CustomTypeMapping
	SQL                 SQLOptions
}

// codegen implements Generator to produce Go code from parsed schemas and queries.
type codegen struct {
	opts Options
}

// File represents a generated source file.
type File struct {
	Path    string
	Content []byte
}

// New creates a new Generator.
func New(opts Options) Generator {
	return &codegen{opts: opts}
}

// Generate builds the AST and renders Go source files.
func (g *codegen) Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	files := make([]File, 0)

	if g.opts.SQL.Enabled {
		sqlFiles, err := g.generateSQL(catalog)
		if err != nil {
			return nil, fmt.Errorf("generate SQL: %w", err)
		}
		files = append(files, sqlFiles...)
	}

	goFiles, err := g.generateGo(ctx, catalog, analyses)
	if err != nil {
		return nil, fmt.Errorf("generate Go: %w", err)
	}
	files = append(files, goFiles...)

	slices.SortFunc(files, func(a, b File) int {
		if a.Path == b.Path {
			return 0
		}
		if a.Path < b.Path {
			return -1
		}
		return 1
	})

	return files, nil
}

func (g *codegen) generateSQL(catalog *model.Catalog) ([]File, error) {
	dialect := sql.DialectSQLite
	switch g.opts.SQL.Dialect {
	case "mysql":
		dialect = sql.DialectMySQL
	case "postgres":
		dialect = sql.DialectPostgres
	case "sqlite":
		dialect = sql.DialectSQLite
	}

	generator := sql.New(sql.Options{
		Dialect:         dialect,
		EmitIFNotExists: g.opts.SQL.EmitIFNotExists,
	})

	sqlFiles, err := generator.Generate(catalog)
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, len(sqlFiles))
	for _, f := range sqlFiles {
		files = append(files, File{Path: f.Path, Content: f.Content})
	}
	return files, nil
}

func (g *codegen) generateGo(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error) {
	transformer := transform.New(g.opts.CustomTypes)
	database := g.opts.Database
	if database == "" {
		database = config.DatabaseSQLite
	}
	typeResolver := astbuilder.NewTypeResolverFull(transformer, database, g.opts.EmitPointersForNull)

	builder := astbuilder.New(astbuilder.Options{
		Package:             g.opts.Package,
		EmitJSONTags:        g.opts.EmitJSONTags,
		EmitEmptySlices:     g.opts.EmitEmptySlices,
		EmitPointersForNull: g.opts.EmitPointersForNull,
		TypeResolver:        typeResolver,
		Prepared: astbuilder.PreparedOptions{
			Enabled:     g.opts.Prepared.Enabled,
			EmitMetrics: g.opts.Prepared.EmitMetrics,
			ThreadSafe:  g.opts.Prepared.ThreadSafe,
		},
	})

	astFiles, err := builder.Build(ctx, catalog, analyses)
	if err != nil {
		return nil, err
	}

	specs := make([]render.Spec, 0, len(astFiles))
	for _, file := range astFiles {
		specs = append(specs, render.Spec{Path: file.Path, Node: file.Node, Raw: file.Raw})
	}

	rendered, err := render.Format(specs)
	if err != nil {
		return nil, fmt.Errorf("render files: %w", err)
	}

	files := make([]File, 0, len(rendered))
	for _, rf := range rendered {
		files = append(files, File{Path: rf.Path, Content: rf.Content})
	}

	return files, nil
}
