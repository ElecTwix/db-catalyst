package config

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/electwix/db-catalyst/internal/fileset"
	toml "github.com/pelletier/go-toml/v2"
)

// Driver identifies the SQLite driver implementation to target.
type Driver string

const (
	// DriverModernC targets modernc.org/sqlite.
	DriverModernC Driver = "modernc"
	// DriverMattN targets github.com/mattn/go-sqlite3.
	DriverMattN Driver = "mattn"
)

var validDrivers = map[Driver]struct{}{
	DriverModernC: {},
	DriverMattN:   {},
}

// Config mirrors the expected db-catalyst TOML schema.
type Config struct {
	Package      string   `toml:"package"`
	Out          string   `toml:"out"`
	SQLiteDriver Driver   `toml:"sqlite_driver"`
	Schemas      []string `toml:"schemas"`
	Queries      []string `toml:"queries"`
}

// LoadOptions tunes config loading behavior.
type LoadOptions struct {
	Strict   bool
	Resolver *fileset.Resolver
}

// JobPlan is the fully-resolved configuration used by downstream stages.
type JobPlan struct {
	Package      string
	Out          string
	SQLiteDriver Driver
	Schemas      []string
	Queries      []string
}

// Result wraps a loaded job plan alongside any non-fatal warnings.
type Result struct {
	Plan     JobPlan
	Warnings []string
}

// Load reads, validates, and resolves a db-catalyst configuration file.
func Load(path string, opts LoadOptions) (Result, error) {
	var res Result

	data, err := os.ReadFile(path)
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}

	unknownKeys, err := collectUnknownKeys(data)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}
	if len(unknownKeys) > 0 {
		slices.Sort(unknownKeys)
		message := fmt.Sprintf("%s: unknown configuration keys: %s", path, strings.Join(unknownKeys, ", "))
		if opts.Strict {
			return res, errors.New(message)
		}
		// TODO: surface warnings through structured logging once CLI wiring exists.
		res.Warnings = append(res.Warnings, message)
	}

	if err := validatePackage(path, cfg.Package); err != nil {
		return res, err
	}

	out, err := resolveOut(path, cfg.Out)
	if err != nil {
		return res, err
	}

	driver, err := resolveDriver(path, cfg.SQLiteDriver)
	if err != nil {
		return res, err
	}

	baseDir := filepath.Dir(path)

	var resolver fileset.Resolver
	if opts.Resolver != nil {
		resolver = *opts.Resolver
	} else {
		resolver, err = fileset.NewOSResolver(baseDir)
		if err != nil {
			return res, fmt.Errorf("%s: %w", path, err)
		}
	}

	schemas, err := resolvePatterns(resolver, "schemas", cfg.Schemas)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}

	queries, err := resolvePatterns(resolver, "queries", cfg.Queries)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}

	res.Plan = JobPlan{
		Package:      cfg.Package,
		Out:          out,
		SQLiteDriver: driver,
		Schemas:      schemas,
		Queries:      queries,
	}

	return res, nil
}

func collectUnknownKeys(data []byte) ([]string, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	known := map[string]struct{}{
		"package":       {},
		"out":           {},
		"sqlite_driver": {},
		"schemas":       {},
		"queries":       {},
	}

	unknown := make([]string, 0)
	for key := range raw {
		if _, ok := known[key]; !ok {
			unknown = append(unknown, key)
		}
	}

	return unknown, nil
}

func validatePackage(path, pkg string) error {
	if pkg == "" {
		return fmt.Errorf("%s: package is required", path)
	}
	if !token.IsIdentifier(pkg) || token.Lookup(pkg) != token.IDENT {
		return fmt.Errorf("%s: invalid package name %q", path, pkg)
	}
	return nil
}

func resolveOut(path, out string) (string, error) {
	if out == "" {
		return "", fmt.Errorf("%s: out is required", path)
	}
	if filepath.IsAbs(out) {
		return "", fmt.Errorf("%s: out must be a relative path", path)
	}

	cleaned := filepath.Clean(out)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: out must not traverse upwards", path)
	}

	baseDir := filepath.Dir(path)
	return filepath.Join(baseDir, cleaned), nil
}

func resolveDriver(path string, driver Driver) (Driver, error) {
	if driver == "" {
		return DriverModernC, nil
	}
	if _, ok := validDrivers[driver]; !ok {
		return "", fmt.Errorf("%s: unsupported sqlite_driver %q", path, driver)
	}
	return driver, nil
}

func resolvePatterns(resolver fileset.Resolver, field string, patterns []string) ([]string, error) {
	paths, err := resolver.Resolve(patterns)
	if err != nil {
		switch {
		case errors.Is(err, fileset.ErrNoPatterns):
			return nil, fmt.Errorf("%s must include at least one pattern", field)
		default:
			var noMatchErr fileset.NoMatchError
			if errors.As(err, &noMatchErr) {
				return nil, fmt.Errorf("%s patterns matched no files: %s", field, strings.Join(noMatchErr.Patterns, ", "))
			}

			var patternErr fileset.PatternError
			if errors.As(err, &patternErr) {
				return nil, fmt.Errorf("%s: invalid glob pattern %q: %v", field, patternErr.Pattern, patternErr.Err)
			}

			return nil, fmt.Errorf("%s: %w", field, err)
		}
	}

	return paths, nil
}
