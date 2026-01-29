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
	"time"

	"github.com/electwix/db-catalyst/internal/cache"
	"github.com/electwix/db-catalyst/internal/codegen"
	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/fileset"
	"github.com/electwix/db-catalyst/internal/logging"
	queryanalyzer "github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	queryparser "github.com/electwix/db-catalyst/internal/query/parser"
	"github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	"github.com/electwix/db-catalyst/internal/transform"
)

const maxFileSize = 100 * 1024 * 1024 // 100MB

// Environment captures external dependencies used by the pipeline.
type Environment struct {
	FSResolver   func(string) (fileset.Resolver, error)
	Logger       logging.Logger
	Writer       Writer
	SchemaParser schemaparser.SchemaParser // injectable schema parser
	Generator    codegen.Generator         // injectable generator
	Cache        cache.Cache               // injectable cache
}

// Writer writes generated files to persistent storage.
type Writer interface {
	WriteFile(path string, data []byte) error
}

// Pipeline orchestrates configuration loading, analysis, and code generation.
type Pipeline struct {
	Env   Environment
	Hooks Hooks // extension hooks
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

	// Call BeforeParse hook
	if p.Hooks.BeforeParse != nil {
		if err := p.Hooks.BeforeParse(ctx, plan.Schemas); err != nil {
			addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("before parse hook: %v", err)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
		}
	}

	catalog, err := p.parseSchemas(ctx, plan, addDiag)
	if err != nil {
		return summary, err
	}

	if firstErrorIndex != -1 {
		return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: nil}
	}

	// Call AfterParse hook
	if p.Hooks.AfterParse != nil {
		if err := p.Hooks.AfterParse(ctx, catalog); err != nil {
			addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("after parse hook: %v", err)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
		}
	}

	// Call BeforeAnalyze hook
	if p.Hooks.BeforeAnalyze != nil {
		if err := p.Hooks.BeforeAnalyze(ctx, plan.Queries); err != nil {
			addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("before analyze hook: %v", err)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
		}
	}

	analyses, err = p.analyzeQueries(ctx, plan, catalog, addDiag)
	if err != nil {
		return summary, err
	}

	// Call AfterAnalyze hook
	if p.Hooks.AfterAnalyze != nil {
		if err := p.Hooks.AfterAnalyze(ctx, analyses); err != nil {
			addDiag(newDiagnostic(absConfigPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("after analyze hook: %v", err)))
			return summary, &DiagnosticsError{Diagnostic: diags[firstErrorIndex], Cause: err}
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

	files, err := p.generateCode(ctx, plan, catalog, analyses, opts, absConfigPath, addDiag)
	if err != nil {
		return summary, err
	}

	return p.writeFiles(ctx, opts, files, summary)
}

// generateCode generates code files from the analyzed queries and catalog.
// It applies custom types transformation if configured.
func (p *Pipeline) generateCode(ctx context.Context, plan config.JobPlan, catalog *model.Catalog, analyses []queryanalyzer.Result, opts RunOptions, configPath string, addDiag func(queryanalyzer.Diagnostic)) ([]codegen.File, error) {
	// Call BeforeGenerate hook
	if p.Hooks.BeforeGenerate != nil {
		if err := p.Hooks.BeforeGenerate(ctx, analyses); err != nil {
			addDiag(newDiagnostic(configPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("before generate hook: %v", err)))
			return nil, fmt.Errorf("before generate hook: %w", err)
		}
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
		return nil, fmt.Errorf("code generation: %w", err)
	}

	// Call AfterGenerate hook
	if p.Hooks.AfterGenerate != nil {
		if err := p.Hooks.AfterGenerate(ctx, generatedFiles); err != nil {
			addDiag(newDiagnostic(configPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("after generate hook: %v", err)))
			return nil, fmt.Errorf("after generate hook: %w", err)
		}
	}

	finalFiles := make([]codegen.File, 0, len(generatedFiles))
	for _, file := range generatedFiles {
		finalPath := filepath.Join(plan.Out, file.Path)
		finalFiles = append(finalFiles, codegen.File{Path: finalPath, Content: file.Content})
	}

	// Generate schema.gen.sql if custom types are defined
	if len(plan.CustomTypes) > 0 {
		transformer := transform.New(plan.CustomTypes)
		for _, schemaPath := range plan.Schemas {
			if sizeErr := checkFileSize(schemaPath); sizeErr != nil {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, sizeErr.Error()))
				return nil, fmt.Errorf("check file size %s: %w", schemaPath, sizeErr)
			}
			contents, readErr := os.ReadFile(filepath.Clean(schemaPath))
			if readErr != nil {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("read schema for transformation: %v", readErr)))
				return nil, fmt.Errorf("read schema %s: %w", schemaPath, readErr)
			}

			// Validate custom types in schema
			missing := transformer.ValidateCustomTypes(contents)
			if len(missing) > 0 {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("custom types used but not defined: %v", missing)))
				return nil, fmt.Errorf("undefined custom types: %v", missing)
			}

			// Transform schema
			transformed, transformErr := transformer.TransformSchema(contents)
			if transformErr != nil {
				addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("transform schema: %v", transformErr)))
				return nil, fmt.Errorf("transform schema %s: %w", schemaPath, transformErr)
			}

			// Add schema.gen.sql to output files
			outSchemaPath := filepath.Join(plan.Out, "schema.gen.sql")
			finalFiles = append(finalFiles, codegen.File{Path: outSchemaPath, Content: transformed})
		}
	}

	return finalFiles, nil
}

// writeFiles writes the generated files to disk.
// It returns the final summary with files populated.
func (p *Pipeline) writeFiles(ctx context.Context, opts RunOptions, files []codegen.File, summary Summary) (Summary, error) {
	summary.Files = files

	// Call BeforeWrite hook
	if p.Hooks.BeforeWrite != nil {
		if err := p.Hooks.BeforeWrite(ctx, files); err != nil {
			return summary, fmt.Errorf("before write hook: %w", err)
		}
	}

	// Call AfterWrite hook (always called, even on error)
	defer func() {
		if p.Hooks.AfterWrite != nil {
			// Don't overwrite the actual error, just log hook error
			_ = p.Hooks.AfterWrite(ctx, summary)
		}
	}()

	if opts.DryRun {
		return summary, nil
	}

	writer := p.Env.Writer
	if writer == nil {
		writer = NewOSWriter()
	}

	for _, file := range files {
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

// parseSchemas parses all schema files from the plan, using cache when available.
// It returns the merged catalog containing all tables and views.
func (p *Pipeline) parseSchemas(ctx context.Context, plan config.JobPlan, addDiag func(queryanalyzer.Diagnostic)) (*model.Catalog, error) {
	// Get or create schema parser
	schemaParser := p.Env.SchemaParser
	if schemaParser == nil {
		var err error
		schemaParser, err = schemaparser.NewSchemaParser("sqlite")
		if err != nil {
			addDiag(newDiagnostic("", 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("create schema parser: %v", err)))
			return nil, err
		}
	}

	catalog := model.NewCatalog()
	for _, schemaPath := range plan.Schemas {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if sizeErr := checkFileSize(schemaPath); sizeErr != nil {
			addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, sizeErr.Error()))
			return nil, sizeErr
		}
		contents, readErr := os.ReadFile(filepath.Clean(schemaPath))
		if readErr != nil {
			addDiag(newDiagnostic(schemaPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("read schema: %v", readErr)))
			return nil, readErr
		}

		// Check cache first
		var parsedCatalog *model.Catalog
		var schemaDiags []schemaparser.Diagnostic
		var parseErr error

		if p.Env.Cache != nil {
			cacheKey := cache.ComputeKeyWithPrefix("schema", contents)
			if cached, ok := p.Env.Cache.Get(ctx, cacheKey); ok {
				if entry, ok := cached.(*schemaCacheEntry); ok {
					parsedCatalog = entry.Catalog
					schemaDiags = entry.Diagnostics
				}
			}
		}

		// Parse if not in cache
		if parsedCatalog == nil {
			parsedCatalog, schemaDiags, parseErr = schemaParser.Parse(ctx, schemaPath, contents)

			// Store in cache
			if p.Env.Cache != nil && parseErr == nil {
				cacheKey := cache.ComputeKeyWithPrefix("schema", contents)
				p.Env.Cache.Set(ctx, cacheKey, &schemaCacheEntry{
					Catalog:     parsedCatalog,
					Diagnostics: schemaDiags,
				}, 5*time.Minute)
			}
		}

		for _, sd := range schemaDiags {
			addDiag(convertSchemaDiagnostic(sd))
		}
		if parseErr != nil {
			return nil, parseErr
		}
		mergeCatalog(catalog, parsedCatalog, addDiag)
	}

	return catalog, nil
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

func checkFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > maxFileSize {
		return fmt.Errorf("file %s exceeds maximum size of %d bytes (%.2f MB)",
			path, maxFileSize, float64(maxFileSize)/(1024*1024))
	}
	return nil
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

// schemaCacheEntry stores cached schema parsing results.
type schemaCacheEntry struct {
	Catalog     *model.Catalog
	Diagnostics []schemaparser.Diagnostic
}

// analyzeQueries parses and analyzes all queries from the plan.
// It reads query files, slices them into blocks, parses each block,
// and analyzes them against the provided catalog.
func (p *Pipeline) analyzeQueries(ctx context.Context, plan config.JobPlan, catalog *model.Catalog, addDiag func(queryanalyzer.Diagnostic)) ([]queryanalyzer.Result, error) {
	queries := make([]queryparser.Query, 0, len(plan.Queries))
	for _, queryPath := range plan.Queries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if sizeErr := checkFileSize(queryPath); sizeErr != nil {
			addDiag(newDiagnostic(queryPath, 1, 1, queryanalyzer.SeverityError, sizeErr.Error()))
			return nil, sizeErr
		}
		contents, readErr := os.ReadFile(filepath.Clean(queryPath))
		if readErr != nil {
			addDiag(newDiagnostic(queryPath, 1, 1, queryanalyzer.SeverityError, fmt.Sprintf("read queries: %v", readErr)))
			return nil, readErr
		}
		blocks, sliceErr := block.Slice(queryPath, contents)
		if sliceErr != nil {
			addDiag(newDiagnostic(queryPath, 1, 1, queryanalyzer.SeverityError, sliceErr.Error()))
			return nil, sliceErr
		}
		for _, blk := range blocks {
			query, queryDiags := queryparser.Parse(blk)
			for _, qd := range queryDiags {
				addDiag(convertQueryDiagnostic(qd))
			}
			queries = append(queries, query)
		}
	}

	// Convert custom types config to map for analyzer
	customTypesMap := make(map[string]config.CustomTypeMapping)
	if len(plan.CustomTypes) > 0 {
		for _, mapping := range plan.CustomTypes {
			customTypesMap[mapping.SQLiteType] = mapping
		}
	}

	analyzer := queryanalyzer.NewWithCustomTypes(catalog, customTypesMap)
	analyses := make([]queryanalyzer.Result, 0, len(queries))
	for _, q := range queries {
		result := analyzer.Analyze(q)
		analyses = append(analyses, result)
		for _, diag := range result.Diagnostics {
			addDiag(diag)
		}
	}

	return analyses, nil
}
