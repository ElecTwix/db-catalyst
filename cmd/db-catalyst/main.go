// Package main implements the db-catalyst CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/electwix/db-catalyst/internal/cli"
	"github.com/electwix/db-catalyst/internal/diagnostics"
	"github.com/electwix/db-catalyst/internal/fileset"
	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/pipeline"
	queryanalyzer "github.com/electwix/db-catalyst/internal/query/analyzer"
)

func main() {
	code := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	opts, err := cli.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(stdout, err.Error())
			return 0
		}
		_, _ = fmt.Fprintln(stderr, err.Error())
		return 1
	}

	slogLogger := logging.New(logging.Options{
		Verbose: opts.Verbose,
		Writer:  stderr,
	})

	env := pipeline.Environment{
		Logger:     logging.NewSlogAdapter(slogLogger),
		FSResolver: fileset.NewOSResolver,
		Writer:     pipeline.NewOSWriter(),
	}

	pipe := pipeline.Pipeline{Env: env}
	summary, runErr := pipe.Run(ctx, pipeline.RunOptions{
		ConfigPath:          opts.ConfigPath,
		OutOverride:         opts.Out,
		DryRun:              opts.DryRun,
		ListQueries:         opts.ListQueries,
		StrictConfig:        opts.StrictConfig,
		NoJSONTags:          opts.NoJSONTags,
		EmitPointersForNull: opts.EmitPointersForNull,
		SQLDialect:          opts.SQLDialect,
		EmitIFNotExists:     opts.EmitIFNotExists,
	})

	printDiagnostics(stderr, summary.Diagnostics, opts.Verbose)

	if runErr != nil {
		var diagErr *pipeline.DiagnosticsError
		if !errors.As(runErr, &diagErr) {
			// For non-diagnostic errors, create a rich diagnostic
			printErrorDiagnostic(stderr, runErr, opts.Verbose)
		}
		var writeErr *pipeline.WriteError
		if errors.As(runErr, &writeErr) {
			return 2 //nolint:mnd // exit code for write errors
		}
		return 1
	}

	if opts.ListQueries {
		printQuerySummary(stdout, summary.Analyses)
		return 0
	}

	if opts.DryRun {
		for _, file := range summary.Files {
			_, _ = fmt.Fprintln(stdout, file.Path)
		}
		return 0
	}

	return 0
}

func printDiagnostics(w io.Writer, diags []queryanalyzer.Diagnostic, verbose bool) {
	if len(diags) == 0 {
		return
	}

	// Convert to rich diagnostics
	collection := diagnostics.CollectionFromQueryAnalyzer(diags)

	// Enrich with suggestions
	diagnostics.EnrichWithSuggestions(collection)

	// In verbose mode, also add context
	if verbose {
		extractor := diagnostics.NewContextExtractor()
		diagnostics.EnrichWithContext(collection, extractor, 2) //nolint:mnd // 2 lines of context
	}

	// Create formatter based on verbosity
	var formatter *diagnostics.Formatter
	if verbose {
		formatter = diagnostics.NewVerboseFormatter()
	} else {
		formatter = diagnostics.NewFormatter()
		formatter.ShowContext = false
		formatter.ShowSuggestions = true
		formatter.ShowNotes = false
		formatter.ShowRelated = false
	}

	// Print all diagnostics
	for _, d := range collection.All() {
		_, _ = fmt.Fprintln(w, formatter.Format(d))
	}

	// Print summary
	if collection.Len() > 0 {
		if verbose {
			formatter.PrintCategorizedSummary(w, collection)
		} else {
			formatter.PrintSummary(w, collection)
		}
	}
}

func printErrorDiagnostic(w io.Writer, err error, verbose bool) {
	// Create a diagnostic from the error
	diag := diagnostics.Error(err.Error()).
		WithCode(diagnostics.ErrCodeGenFailed).
		WithSource("db-catalyst").
		Build()

	formatter := diagnostics.NewFormatter()
	if verbose {
		formatter = diagnostics.NewVerboseFormatter()
	}
	_, _ = fmt.Fprintln(w, formatter.Format(diag))
}

func printQuerySummary(w io.Writer, analyses []queryanalyzer.Result) {
	for _, analysis := range analyses {
		params := formatParams(analysis.Params)
		_, _ = fmt.Fprintf(w, "%s %s %s\n", analysis.Query.Block.Name, analysis.Query.Block.Command.String(), params)
	}
}

func formatParams(params []queryanalyzer.ResultParam) string {
	if len(params) == 0 {
		return "params: none"
	}
	parts := make([]string, 0, len(params))
	for _, param := range params {
		segment := param.Name
		if param.IsVariadic {
			segment += "..."
		}
		segment += ":" + param.GoType
		if param.Nullable {
			segment += "?"
		}
		if param.IsVariadic && param.VariadicCount > 0 {
			segment += fmt.Sprintf("[x%d]", param.VariadicCount)
		}
		parts = append(parts, segment)
	}
	return "params: " + strings.Join(parts, ", ")
}
