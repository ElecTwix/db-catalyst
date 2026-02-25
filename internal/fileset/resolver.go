// Package fileset handles file path resolution and glob expansion.
package fileset

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Resolver resolves glob patterns against an fs.FS and rewrites the discovered
// paths using a join function for deterministic, de-duplicated results.
type Resolver struct {
	fsys fs.FS
	join func(name string) string
}

// ErrNoPatterns indicates that Resolve was invoked without any glob patterns.
var ErrNoPatterns = errors.New("fileset: no patterns provided")

// PatternError wraps syntax issues reported while evaluating a glob pattern.
type PatternError struct {
	Pattern string
	Err     error
}

// Error implements the error interface.
func (e PatternError) Error() string {
	return fmt.Sprintf("invalid glob pattern %q: %v", e.Pattern, e.Err)
}

// Unwrap returns the underlying error.
func (e PatternError) Unwrap() error { return e.Err }

// NoMatchError describes which patterns failed to yield any results.
type NoMatchError struct {
	Patterns []string
}

// Error implements the error interface.
func (e NoMatchError) Error() string {
	return "patterns matched no files: " + strings.Join(e.Patterns, ", ")
}

// NewResolver constructs a Resolver against the provided filesystem without any
// path rewriting, preserving the original match names. Useful for tests.
func NewResolver(fsys fs.FS) Resolver {
	return Resolver{
		fsys: fsys,
		join: func(name string) string { return name },
	}
}

// NewOSResolver constructs a Resolver rooted at base that returns absolute OS
// paths for each match.
func NewOSResolver(base string) (Resolver, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return Resolver{}, fmt.Errorf("resolve base %q: %w", base, err)
	}

	info, err := os.Stat(absBase)
	if err != nil {
		return Resolver{}, fmt.Errorf("stat base %q: %w", absBase, err)
	}
	if !info.IsDir() {
		return Resolver{}, fmt.Errorf("base %q is not a directory", absBase)
	}

	return Resolver{
		fsys: os.DirFS(absBase),
		join: func(name string) string {
			if filepath.IsAbs(name) {
				return filepath.Clean(name)
			}
			return filepath.Join(absBase, filepath.FromSlash(name))
		},
	}, nil
}

// Resolve evaluates each glob pattern, accumulating matches, and returns a
// deterministically sorted list of de-duplicated paths.
func (r Resolver) Resolve(patterns []string) ([]string, error) {
	if r.fsys == nil {
		return nil, errors.New("fileset: resolver has no filesystem")
	}

	if len(patterns) == 0 {
		return nil, ErrNoPatterns
	}

	joinFn := r.join
	if joinFn == nil {
		joinFn = func(name string) string { return name }
	}

	combined := make([]string, 0)
	missing := make([]string, 0)

	for _, pattern := range patterns {
		globPattern := filepath.ToSlash(pattern)

		matches, err := fs.Glob(r.fsys, globPattern)
		if err != nil {
			return nil, PatternError{Pattern: pattern, Err: err}
		}

		if len(matches) == 0 {
			missing = append(missing, pattern)
			continue
		}

		for _, match := range matches {
			combined = append(combined, joinFn(match))
		}
	}

	if len(missing) > 0 {
		return nil, NoMatchError{Patterns: append([]string(nil), missing...)}
	}

	unique := dedupePreserveOrder(combined)
	slices.Sort(unique)
	unique = slices.Compact(unique)
	return unique, nil
}

func dedupePreserveOrder(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}
