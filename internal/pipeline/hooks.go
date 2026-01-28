package pipeline

import (
	"context"

	"github.com/electwix/db-catalyst/internal/codegen"
	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// Hooks provides extension points in the pipeline execution.
// Each hook is called at a specific stage and can modify behavior or perform side effects.
type Hooks struct {
	// BeforeParse is called before parsing schema files.
	// Return an error to abort the pipeline.
	BeforeParse func(ctx context.Context, schemaPaths []string) error

	// AfterParse is called after all schemas are parsed.
	// Return an error to abort the pipeline.
	AfterParse func(ctx context.Context, catalog *model.Catalog) error

	// BeforeAnalyze is called before analyzing queries.
	// Return an error to abort the pipeline.
	BeforeAnalyze func(ctx context.Context, queryPaths []string) error

	// AfterAnalyze is called after all queries are analyzed.
	// Return an error to abort the pipeline.
	AfterAnalyze func(ctx context.Context, analyses []analyzer.Result) error

	// BeforeGenerate is called before generating code.
	// Return an error to abort the pipeline.
	BeforeGenerate func(ctx context.Context, analyses []analyzer.Result) error

	// AfterGenerate is called after code is generated.
	// Return an error to abort the pipeline.
	AfterGenerate func(ctx context.Context, files []codegen.File) error

	// BeforeWrite is called before writing files.
	// Return an error to abort the pipeline.
	BeforeWrite func(ctx context.Context, files []codegen.File) error

	// AfterWrite is called after all files are written.
	// This is the final hook, called even if earlier stages failed.
	AfterWrite func(ctx context.Context, summary Summary) error
}

// Chain combines two Hooks, calling h's hooks first, then other's hooks.
// If a hook in h returns an error, other's hook is not called.
func (h Hooks) Chain(other Hooks) Hooks {
	return Hooks{
		BeforeParse:    chainHook(h.BeforeParse, other.BeforeParse),
		AfterParse:     chainHook(h.AfterParse, other.AfterParse),
		BeforeAnalyze:  chainHook(h.BeforeAnalyze, other.BeforeAnalyze),
		AfterAnalyze:   chainHook(h.AfterAnalyze, other.AfterAnalyze),
		BeforeGenerate: chainHook(h.BeforeGenerate, other.BeforeGenerate),
		AfterGenerate:  chainHook(h.AfterGenerate, other.AfterGenerate),
		BeforeWrite:    chainHook(h.BeforeWrite, other.BeforeWrite),
		AfterWrite:     chainHook(h.AfterWrite, other.AfterWrite),
	}
}

// chainHook chains two hooks of the same type.
func chainHook[T any](first, second func(context.Context, T) error) func(context.Context, T) error {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(ctx context.Context, arg T) error {
		if err := first(ctx, arg); err != nil {
			return err
		}
		return second(ctx, arg)
	}
}

// NoHooks returns a Hooks with all nil functions (no-op).
func NoHooks() Hooks {
	return Hooks{}
}
