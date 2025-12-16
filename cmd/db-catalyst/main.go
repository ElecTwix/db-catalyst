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

	logger := logging.New(logging.Options{
		Verbose: opts.Verbose,
		Writer:  stderr,
	})

	env := pipeline.Environment{
		Logger: logger,
		FSResolver: fileset.NewOSResolver,
		Writer: pipeline.NewOSWriter(),
	}

	pipe := pipeline.Pipeline{Env: env}
	summary, runErr := pipe.Run(ctx, pipeline.RunOptions{
		ConfigPath:   opts.ConfigPath,
		OutOverride:  opts.Out,
		DryRun:       opts.DryRun,
		ListQueries:  opts.ListQueries,
		StrictConfig: opts.StrictConfig,
	})

	printDiagnostics(stderr, summary.Diagnostics)

	if runErr != nil {
		var diagErr *pipeline.DiagnosticsError
		if !errors.As(runErr, &diagErr) {
			_, _ = fmt.Fprintln(stderr, runErr.Error())
		}
		var writeErr *pipeline.WriteError
		if errors.As(runErr, &writeErr) {
			return 2
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

func printDiagnostics(w io.Writer, diags []queryanalyzer.Diagnostic) {
	for _, diag := range diags {
		level := "warning"
		if diag.Severity == queryanalyzer.SeverityError {
			level = "error"
		}
		_, _ = fmt.Fprintf(w, "%s:%d:%d: %s [%s]\n", diag.Path, diag.Line, diag.Column, diag.Message, level)
	}
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
