package codegen

import (
	"context"
	"fmt"
	"slices"

	astbuilder "github.com/electwix/db-catalyst/internal/codegen/ast"
	"github.com/electwix/db-catalyst/internal/codegen/render"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

type PreparedOptions struct {
	Enabled     bool
	EmitMetrics bool
	ThreadSafe  bool
}

type Options struct {
	Package      string
	EmitJSONTags bool
	Prepared     PreparedOptions
}

type Generator struct {
	opts Options
}

type File struct {
	Path    string
	Content []byte
}

func New(opts Options) *Generator {
	return &Generator{opts: opts}
}

func (g *Generator) Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	builder := astbuilder.New(astbuilder.Options{
		Package:      g.opts.Package,
		EmitJSONTags: g.opts.EmitJSONTags,
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
