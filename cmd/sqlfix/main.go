// Package main implements the sqlfix tool for automated SQL rewrites.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/logging"
	"github.com/electwix/db-catalyst/internal/sqlfix"
)

func main() {
	ctx := context.Background()

	var (
		configPath string
		dryRun     bool
		verbose    bool
		pathsFlag  string
	)

	flag.StringVar(&configPath, "config", "db-catalyst.toml", "Path to db-catalyst configuration file")
	flag.StringVar(&configPath, "c", "db-catalyst.toml", "Path to db-catalyst configuration file")
	flag.BoolVar(&dryRun, "dry-run", false, "Display planned changes without writing files")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging")
	flag.StringVar(&pathsFlag, "paths", "", "Comma-separated list of SQL files to rewrite")
	flag.Parse()

	paths := make([]string, 0)
	if pathsFlag != "" {
		for _, p := range strings.Split(pathsFlag, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				paths = append(paths, trimmed)
			}
		}
	}

	paths = append(paths, flag.Args()...)

	exitCode := run(ctx, configPath, dryRun, verbose, paths, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

func run(ctx context.Context, configPath string, dryRun, verbose bool, paths []string, stdout, stderr io.Writer) int {
	configResult, err := config.Load(configPath, config.LoadOptions{Strict: false})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}
	plan := configResult.Plan

	if len(paths) == 0 {
		paths = append(paths, plan.Queries...)
	}

	if len(paths) == 0 {
		_, _ = fmt.Fprintln(stderr, "sqlfix: no query files to process")
		return 1
	}

	dedup := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "resolve path %s: %v\n", p, err)
			return 1
		}
		dedup[abs] = struct{}{}
	}

	paths = make([]string, 0, len(dedup))
	for p := range dedup {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	schemaResult, err := sqlfix.LoadSchemaCatalog(plan.Schemas, nil)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "load schema catalog: %v\n", err)
		return 1
	}

	runner := sqlfix.NewRunner()
	runner.SetCatalog(schemaResult.Catalog, schemaResult.Warnings)
	runner.DryRun = dryRun
	runner.Logger = logging.New(logging.Options{Verbose: verbose, Writer: stderr})

	reports, err := runner.Rewrite(ctx, paths)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "rewrite: %v\n", err)
		return 1
	}

	catalogWarnings := runner.CatalogWarnings()
	for _, warn := range catalogWarnings {
		_, _ = fmt.Fprintf(stderr, "%s\n", warn)
	}

	totalAliases := 0
	totalStars := 0
	for _, rep := range reports {
		aliasCount := len(rep.Added)
		starCount := rep.ExpandedStars
		if aliasCount == 0 && starCount == 0 {
			continue
		}
		totalAliases += aliasCount
		totalStars += starCount

		segments := make([]string, 0, 2)
		if aliasCount > 0 {
			segments = append(segments, fmt.Sprintf("added aliases for %d column(s)", aliasCount))
		}
		if starCount > 0 {
			segments = append(segments, fmt.Sprintf("expanded %d star projection(s)", starCount))
		}
		_, _ = fmt.Fprintf(stdout, "%s: %s\n", rep.Path, strings.Join(segments, "; "))
	}

	for _, rep := range reports {
		for _, warn := range rep.Warnings {
			_, _ = fmt.Fprintf(stderr, "%s\n", warn)
		}
		for _, skipped := range rep.Skipped {
			_, _ = fmt.Fprintf(stderr, "%s: skipped %q (%s)\n", rep.Path, skipped.Expr, skipped.Reason)
		}
	}

	if totalAliases == 0 && totalStars == 0 {
		if dryRun {
			_, _ = fmt.Fprintln(stdout, "sqlfix (dry-run): no changes")
		} else {
			_, _ = fmt.Fprintln(stdout, "sqlfix: no changes")
		}
		return 0
	}

	if !dryRun {
		_, _ = fmt.Fprintf(stdout, "sqlfix: added %d alias(es), expanded %d star projection(s)\n", totalAliases, totalStars)
	} else {
		_, _ = fmt.Fprintf(stdout, "sqlfix (dry-run): would add %d alias(es), expand %d star projection(s)\n", totalAliases, totalStars)
	}
	return 0
}
