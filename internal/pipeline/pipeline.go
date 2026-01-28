// Package pipeline orchestrates the entire code generation process.
package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"log/slog"

	"github.com/electwix/db-catalyst/internal/codegen"
	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/fileset"
	queryanalyzer "github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	"github.com/electwix/db-catalyst/internal/transform"
)

// Environment captures external dependencies used by the pipeline.
type Environment struct {
	FSResolver   func(string) (fileset.Resolver, error)
	Logger       *slog.Logger
	Writer       Writer
	SchemaParser schemaparser.SchemaParser // injectable schema parser
	Generator    codegen.Generator         // injectable generator
}

// Writer writes generated files to persistent storage.
type Writer interface {
	WriteFile(path string, data []byte) error
}

// Pipeline orchestrates configuration loading, analysis, and code generation.
type Pipeline struct {
	Env Environment
}

// Summary captures generated files and diagnostics collected during a run.
type Summary struct {
	Files       []codegen.File
	Diagnostics []queryanalyzer.Diagnostic
	Analyses    []queryanalyzer.Result
}

// RunOptions configures a pipeline execution.
type RunOptions struct {
	ConfigPath      string
	OutOverride     string
	DryRun          bool
	ListQueries     bool
	StrictConfig    bool
	NoJSONTags      bool
	SQLDialect      string
	EmitIFNotExists bool
}

// DiagnosticsError indicates that errors were reported via diagnostics.
type DiagnosticsError struct {
	Diagnostic queryanalyzer.Diagnostic
	Cause      error
}

func (e *DiagnosticsError) Error() string {
	d := e.Diagnostic
	return fmt.Sprintf("%s:%d:%d: %s", d.Path, d.Line, d.Column, d.Message)
}

func (e *DiagnosticsError) Unwrap() error {
	return e.Cause
}

// WriteError wraps failures encountered while writing generated files.
type WriteError struct {
	Path string
	Err  error
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("write %s: %v", e.Path, e.Err)
}

func (e *WriteError) Unwrap() error {
	return e.Err
}

// NewOSWriter returns a Writer that performs atomic writes on the local filesystem.
func NewOSWriter() Writer {
	return &osWriter{perm: 0o644}
}

type osWriter struct {
	perm fs.FileMode
}

func (w *osWriter) WriteFile(path string, data []byte) error {
	if path == "" {
		return errors.New("pipeline: empty path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".db-catalyst-")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
		_ = tmp.Close()
	}()
	if w.perm != 0 {
		if err := tmp.Chmod(w.perm); err != nil {
			return fmt.Errorf("chmod temp file: %w", err)
		}
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	success = true
	return nil
}

// Run executes the pipeline according to the provided options.
func (p *Pipeline) Run(ctx context.Context, opts RunOptions) (summary Summary, err error) {
	diags := make([]queryanalyzer.Diagnostic, 0, 8)
	analyses := make([]queryanalyzer.Result, 0, 4)
	firstErrorIndex := -1

	addDiag := func(d queryanalyzer.Diagnostic) {
		if d.Path == "" {
			d.Path = opts.ConfigPath
		}
		if d.Line <= 0 {
			d.Line = 1
		}
		if d.Column <= 0 {
			d.Column = 1
		}
		diags = append(diags, d)
		if d.Severity == queryanalyzer.SeverityError && firstErrorIndex == -1 {
			firstErrorIndex = len(diags) - 1
		}
	}

	finalize := func() {
		summary.Diagnostics = append([]queryanalyzer.Diagnostic(nil), diags...)
		summary.Analyses = append([]queryanalyzer.Result(nil), analyses...)
	}
	defer finalize()

	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = "db-catalyst.toml"
	}
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		addDiag(newDiagnostic(configPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("resolve config path: %v", err)))
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
	}

	baseDir := filepath.Dir(absConfigPath)
	resolverFn := p.Env.FSResolver
	if resolverFn == nil {
		resolverFn = fileset.NewOSResolver
	}

	resolver, err := resolverFn(baseDir)
	if err != nil {
		addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("resolve filesystem: %v", err)))
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
	}

	loadResult, err := config.Load(absConfigPath, config.LoadOptions{Strict: opts.StrictConfig, Resolver: &resolver})
	if err != nil {
		addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, err.Error()))
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
	}
	for _, warning := range loadResult.Warnings {
		addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityWarning, warning))
	}

	plan := loadResult.Plan
	// CLI flag overrides config setting
	if opts.NoJSONTags {
		plan.EmitJSONTags = false
	}
	outDir := plan.Out
	if opts.OutOverride != "" {
		override := opts.OutOverride
		if !filepath.IsAbs(override) {
			override = filepath.Join(baseDir, override)
		}
		outDir = filepath.Clean(override)
	}
	plan.Out = outDir

	if err := ctx.Err(); err != nil {
		return summary, err
	}

	// Get or create schema parser
	schemaParser := p.Env.SchemaParser
	if schemaParser == nil {
		var err error
		schemaParser, err = schemaparser.NewSchemaParser("sqlite")
		if err != nil {
			addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("create schema parser: %v", err)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
		}
	}

	catalog := model.NewCatalog()
	for _, schemaPath := range plan.Schemas {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		contents, readErr := os.ReadFile(filepath.Clean(schemaPath))
		if readErr != nil {
			addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("read schema: %v", readErr)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: readErr}
		}

		parsedCatalog, schemaDiags, parseErr := schemaParser.Parse(ctx, schemaPath, contents)
		for _, sd := range schemaDiags {
			addDiag(convertSchemaDiagnostic(sd))
		}
		if parseErr != nil {
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: parseErr}
		}
		mergeCatalog(catalog, parsedCatalog, addDiag)
	}

	if firstErrorIndex != -1 {
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: nil}
	}

	queries := make([]queryparser.Query, 0, len(plan.Queries))
	for _, queryPath := range plan.Queries {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		contents, readErr := os.ReadFile(filepath.Clean(queryPath))
		if readErr != nil {
			addDiag(newDiagnostic(queryPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("read queries: %v", readErr)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: readErr}
		}
		blocks, sliceErr := block.Slice(queryPath, contents)
		if sliceErr != nil {
			addDiag(newDiagnostic(queryPath, 1, 1, queryanalyzer.SeverityError, sliceErr.Error()))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: sliceErr}
		}
		for _, blk := range blocks {
			query, queryDiags := queryparser.Parse(blk)
			for _, qd := range queryDiags {
				addDiag(convertQueryDiagnostic(qd))
			}
			queries = append(queries, query)
		}
	}

	if firstErrorIndex != -1 {
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: nil}
	}

	// Convert custom types config to map for analyzer
	customTypesMap := make(map[string]config.CustomTypeMapping)
	if len(plan.CustomTypes) > 0 {
		for _, mapping := range plan.CustomTypes {
			customTypesMap[mapping.SQLiteType] = mapping
		}
	}

	analyzer := queryanalyzer.NewWithCustomTypes(catalog, customTypesMap)
	for _, q := range queries {
		result := analyzer.Analyze(q)
		analyses = append(analyses, result)
		for _, diag := range result.Diagnostics {
			addDiag(diag)
		}
	}

	if opts.ListQueries {
		if firstErrorIndex != -1 {
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: nil}
		}
		return summary, nil
	}

	if firstErrorIndex != -1 {
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: nil}
	}

	// Get or create generator
	var generator codegen.Generator
	if p.Env.Generator != nil {
		generator = p.Env.Generator
	} else {
		generator = codegen.New(codegen.Options{
			Package:             plan.Package,
			EmitJSONTags:        plan.EmitJSONTags,
			EmitEmptySlices:     plan.PreparedQueries.EmitEmptySlices,
			EmitPointersForNull: plan.EmitPointersForNull,
			CustomTypes:         plan.CustomTypes,
			Prepared: codegen.PreparedOptions{
				Enabled:     plan.PreparedQueries.Enabled,
				EmitMetrics: plan.PreparedQueries.Metrics,
				ThreadSafe:  plan.PreparedQueries.ThreadSafe,
			},
			SQL: codegen.SQLOptions{
				Enabled:         plan.SQLDialect != "",
				Dialect:         plan.SQLDialect,
				EmitIFNotExists: opts.EmitIFNotExists,
			},
		})
	}

	generatedFiles, err := generator.Generate(ctx, catalog, analyses)
	if err != nil {
		return summary, fmt.Errorf("code generation: %w", err)
	}

	finalFiles := make([]codegen.File, 0, len(generatedFiles))
	for _, file := range generatedFiles {
		finalPath := filepath.Join(outDir, file.Path)
		finalFiles = append(finalFiles, codegen.File{Path: finalPath, Content: file.Content})
	}

	// Generate schema.gen.sql if custom types are defined
	if len(plan.CustomTypes) > 0 {
		transformer := transform.New(plan.CustomTypes)
		for _, schemaPath := range plan.Schemas {
			contents, readErr := os.ReadFile(filepath.Clean(schemaPath))
			if readErr != nil {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("read schema for transformation: %v", readErr)))
				return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: readErr}
			}

			// Validate custom types in schema
			missing := transformer.ValidateCustomTypes(contents)
			if len(missing) > 0 {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("custom types used but not defined: %v", missing)))
				return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: fmt.Errorf("undefined custom types")}
			}

			// Transform schema
			transformed, transformErr := transformer.TransformSchema(contents)
			if transformErr != nil {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("transform schema: %v", transformErr)))
				return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: transformErr}
			}

			// Add schema.gen.sql to output files
			schemaPath := filepath.Join(outDir, "schema.gen.sql")
			finalFiles = append(finalFiles, codegen.File{Path: schemaPath, Content: transformed})
		}
	}

	summary.Files = finalFiles

	if opts.DryRun {
		return summary, nil
	}

	writer := p.Env.Writer
	if writer == nil {
		writer = NewOSWriter()
	}

	for _, file := range finalFiles {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		same, cmpErr := fileMatches(file.Path, file.Content)
		if cmpErr != nil {
			return summary, &WriteError{Path: file.Path, Err: cmpErr}
		}
		if same {
			continue
		}
		if err := writer.WriteFile(file.Path, file.Content); err != nil {
			return summary, &WriteError{Path: file.Path, Err: err}
		}
	}

	return summary, nil
}

func convertSchemaDiagnostic(d schemaparser.Diagnostic) queryanalyzer.Diagnostic {
	severity := queryanalyzer.SeverityWarning
	if d.Severity == schemaparser.SeverityError {
		severity = queryanalyzer.SeverityError
	}
	return newDiagnostic(d.Path, d.Line, d.Column, severity, d.Message)
}

func convertQueryDiagnostic(d queryparser.Diagnostic) queryanalyzer.Diagnostic {
	severity := queryanalyzer.SeverityWarning
	if d.Severity == queryparser.SeverityError {
		severity = queryanalyzer.SeverityError
	}
	return newDiagnostic(d.Path, d.Line, d.Column, severity, d.Message)
}

func newDiagnostic(path string, line, column int, severity queryanalyzer.Severity, message string) queryanalyzer.Diagnostic {
	return queryanalyzer.Diagnostic{
		Path:     path,
		Line:     line,
		Column:   column,
		Message:  message,
		Severity: severity,
	}
}

func mergeCatalog(dest, src *model.Catalog, addDiag func(queryanalyzer.Diagnostic)) {
	if src == nil {
		return
	}
	for key, table := range src.Tables {
		if existing, ok := dest.Tables[key]; ok {
			message := fmt.Sprintf("duplicate table %q (previous definition at %s:%d:%d)", table.Name, existing.Span.File, existing.Span.StartLine, existing.Span.StartColumn)
			addDiag(newDiagnostic(table.Span.File, table.Span.StartLine, table.Span.StartColumn, queryanalyzer.SeverityError, message))
			continue
		}
		dest.Tables[key] = table
	}
	for key, view := range src.Views {
		if existing, ok := dest.Views[key]; ok {
			message := fmt.Sprintf("duplicate view %q (previous definition at %s:%d:%d)", view.Name, existing.Span.File, existing.Span.StartLine, existing.Span.StartColumn)
			addDiag(newDiagnostic(view.Span.File, view.Span.StartLine, view.Span.StartColumn, queryanalyzer.SeverityError, message))
			continue
		}
		dest.Views[key] = view
	}
}

func fileMatches(path string, content []byte) (bool, error) {
	existing, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(existing, content), nil
}
