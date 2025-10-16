package fileset

import (
	"errors"
	"io/fs"
	"testing"

	"testing/fstest"
)

func TestResolverResolveSuccess(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"schemas/books.sql":        &fstest.MapFile{Mode: fs.ModePerm},
		"schemas/users.sql":        &fstest.MapFile{Mode: fs.ModePerm},
		"queries/find_user.sql":    &fstest.MapFile{Mode: fs.ModePerm},
		"queries/list_users.sql":   &fstest.MapFile{Mode: fs.ModePerm},
		"queries/archive.sql":      &fstest.MapFile{Mode: fs.ModePerm},
		"queries/legacy/query.sql": &fstest.MapFile{Mode: fs.ModePerm},
	}

	resolver := NewResolver(fsys)
	patterns := []string{
		"queries/*.sql",
		"schemas/*.sql",
		"queries/find_user.sql",
	}

	paths, err := resolver.Resolve(patterns)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	expected := []string{
		"queries/archive.sql",
		"queries/find_user.sql",
		"queries/list_users.sql",
		"schemas/books.sql",
		"schemas/users.sql",
	}

	if len(paths) != len(expected) {
		t.Fatalf("expected %d paths, got %d (%v)", len(expected), len(paths), paths)
	}

	for i, want := range expected {
		if paths[i] != want {
			t.Fatalf("unexpected path at %d: want %q, got %q", i, want, paths[i])
		}
	}
}

func TestResolverResolveNoMatches(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"queries/find_user.sql": &fstest.MapFile{Mode: fs.ModePerm},
	}

	resolver := NewResolver(fsys)
	patterns := []string{
		"schemas/*.sql",
		"queries/nope.sql",
	}

	_, err := resolver.Resolve(patterns)
	if err == nil {
		t.Fatal("expected error for missing patterns")
	}

	var noMatchErr NoMatchError
	if !errors.As(err, &noMatchErr) {
		t.Fatalf("expected NoMatchError, got %T: %v", err, err)
	}

	if len(noMatchErr.Patterns) != 2 {
		t.Fatalf("unexpected patterns length: %v", noMatchErr.Patterns)
	}

	if noMatchErr.Patterns[0] != "schemas/*.sql" || noMatchErr.Patterns[1] != "queries/nope.sql" {
		t.Fatalf("unexpected missing patterns: %v", noMatchErr.Patterns)
	}
}

func TestResolverResolveInvalidPattern(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(fstest.MapFS{})

	_, err := resolver.Resolve([]string{"["})
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}

	var patternErr PatternError
	if !errors.As(err, &patternErr) {
		t.Fatalf("expected PatternError, got %T: %v", err, err)
	}

	if patternErr.Pattern != "[" {
		t.Fatalf("unexpected pattern on error: %q", patternErr.Pattern)
	}
}

func TestResolverResolveNoPatterns(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(fstest.MapFS{})

	_, err := resolver.Resolve(nil)
	if !errors.Is(err, ErrNoPatterns) {
		t.Fatalf("expected ErrNoPatterns, got %v", err)
	}
}
