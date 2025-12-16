// Package main implements the sqlfix-sqlc tool for migrating sqlc configurations.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/electwix/db-catalyst/internal/config"
	"github.com/electwix/db-catalyst/internal/sqlfix/overrides"
	"github.com/electwix/db-catalyst/internal/sqlfix/sqlcconfig"
	toml "github.com/pelletier/go-toml/v2"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sqlfix-sqlc", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		sqlcPath string
		dbConfig string
		outPath  string
		dryRun   bool
	)

	fs.StringVar(&sqlcPath, "sqlc-config", "sqlc.yaml", "Path to sqlc configuration file")
	fs.StringVar(&sqlcPath, "sqlc", "sqlc.yaml", "Path to sqlc configuration file")
	fs.StringVar(&dbConfig, "db-config", "db-catalyst.toml", "Path to db-catalyst configuration file")
	fs.StringVar(&dbConfig, "config", "db-catalyst.toml", "Path to db-catalyst configuration file")
	fs.StringVar(&outPath, "out", "", "Destination for merged db-catalyst configuration")
	fs.BoolVar(&dryRun, "dry-run", false, "Print merged configuration without writing file")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if outPath == "" {
		outPath = dbConfig
	}

	sqlcCfg, err := sqlcconfig.Load(sqlcPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "sqlfix-sqlc: %v\n", err)
		return 1
	}

	newMappings, warnings := overrides.ConvertOverrides(sqlcCfg)

	existing, err := readDBCatalystConfig(dbConfig)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			existing = config.Config{}
		} else {
			_, _ = fmt.Fprintf(stderr, "sqlfix-sqlc: %v\n", err)
			return 1
		}
	}

	merged, mergeWarnings := overrides.MergeMappings(existing.CustomTypes.Mappings, newMappings)
	warnings = append(warnings, mergeWarnings...)
	existing.CustomTypes.Mappings = merged

	output, err := toml.Marshal(existing)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "sqlfix-sqlc: marshal db-catalyst config: %v\n", err)
		return 1
	}

	if dryRun {
		if _, err := stdout.Write(output); err != nil {
			_, _ = fmt.Fprintf(stderr, "sqlfix-sqlc: write output: %v\n", err)
			return 1
		}
	} else {
		if err := ensureDir(outPath); err != nil {
			_, _ = fmt.Fprintf(stderr, "sqlfix-sqlc: %v\n", err)
			return 1
		}
		if err := os.WriteFile(outPath, output, 0o600); err != nil {
			_, _ = fmt.Fprintf(stderr, "sqlfix-sqlc: write %s: %v\n", outPath, err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "wrote %s with %d mappings\n", outPath, len(merged))
	}

	if len(warnings) > 0 {
		sort.Strings(warnings)
		for _, w := range warnings {
			_, _ = fmt.Fprintf(stderr, "warning: %s\n", w)
		}
	}

	return 0
}

func readDBCatalystConfig(path string) (config.Config, error) {
	var cfg config.Config
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o750)
}
